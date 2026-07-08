# Boutique

A production-shaped e-commerce store — storefront (browse, cart, checkout,
order lookup) plus a Medusa-style admin (products/variants/categories,
orders, inventory, discounts) — built on the Go standard library. The only
non-stdlib dependencies are `golang.org/x/*` packages, `modernc.org/sqlite`
(pure-Go SQLite driver — `database/sql` ships no driver of its own), and
`goose` for migrations.

## Stack

- Go, `net/http.ServeMux` for routing (1.22+ method+path patterns, no
  third-party router)
- SQLite (`modernc.org/sqlite`) + `goose` migrations, embedded and run at
  startup
- `html/template`, base-layout + per-page clone (see `internal/render`)
- Sessions: signed opaque tokens (`crypto/hmac` + `crypto/rand`), hashed
  before storage, revocable server-side
- CSRF: stdlib `http.CrossOriginProtection` (Go 1.25+) — no tokens, no
  cookies, checks `Sec-Fetch-Site`/`Origin`
- Payment: `internal/payment.Provider` interface, `MockProvider` is the only
  implementation — swap in a real Stripe HTTP client without touching
  checkout
- No inline CSS/JS anywhere; vanilla JS is progressive enhancement only —
  every form works with JS disabled

### Toolchain note

`go.mod` currently targets `go 1.27` / `toolchain go1.27rc1` to use
`strings.CutLast` and `url.Values.Clone` (both new in 1.27). Go 1.27 stable
ships ~August 2026 — until then, building this repo requires the RC
toolchain installed locally. Repoint `go.mod` at a stable `go 1.2x` line and
drop the `toolchain` directive if that's a blocker for CI/deployment before
then.

## Running it

```
make run          # starts the server on :8080
make seed          # demo products/categories + a dev admin account
make test          # go test ./...
make test-race     # same, with the race detector
```

First run: `make seed` creates `admin@example.com` / `changeme123` (override
the password via `SEED_ADMIN_PASSWORD`). There's no admin-creation CLI yet —
`cmd/seed` is the only bootstrap path, fine for local dev, not for a real
deploy. Storefront: `http://localhost:8080/`. Admin:
`http://localhost:8080/admin/login`.

### Config (env vars)

| Var | Default | Notes |
|---|---|---|
| `ADDR` | `:8080` | listen address |
| `DATABASE_PATH` | `boutique.db` | SQLite file |
| `UPLOADS_DIR` | `uploads` | runtime-writable dir for product image uploads |
| `SESSION_KEY` | dev-only fallback | HMAC key for session/CSRF signing — **must** be set in `ENV=prod` |
| `ENV` | `dev` | `prod` enables `Secure` cookies + HSTS and requires `SESSION_KEY` |

## Architecture

```
cmd/server/     wiring: config → db → stores → handlers → routes → server
cmd/seed/       dev-only demo data + bootstrap admin account
internal/
  config/       env-based config
  db/           sqlite open + embedded goose migrations
  httpx/        shared middleware: logging, recover, security headers, rate limiter
  auth/         session tokens, HMAC signer, bcrypt password hashing
  catalog/      products, variants, categories, images (storefront read path
                in store.go, admin write path in admin_store.go)
  media/        image upload: decode → downscale → re-encode as JPEG
  cart/         guest cart (cookie-resolved), session middleware
  checkout/     cart → order: stock re-check, decrement, charge, all one tx
  order/        placed orders, status pipeline, admin listing/detail
  discount/     discount codes: admin CRUD + checkout lookup/apply
  payment/      Provider interface + MockProvider
  render/       html/template loader (layout + page, cached per page)
  storefront/   public HTTP handlers
  admin/        admin HTTP handlers (auth, catalog, orders, inventory, discounts)
web/
  templates/    layout/{base,admin_base}.html + storefront/ + admin/
  static/       css/js/img, embedded into the binary at build time
```

Each domain package owns its own SQL — no ORM, no generic repository layer.
Money is always `int64` cents; `internal/money` is the only place that
formats it for display (`FormatCents` for output, `FormatCentsPlain` for
values that round-trip through a form field, which can't contain a `$`
without breaking the parser that reads them back).

### Correctness notes worth knowing

- `internal/db.Open` caps the SQLite connection pool at 1
  (`SetMaxOpenConns(1)`). That single-writer serialization is what actually
  prevents overselling under concurrent checkouts — not row locking. See
  the test `TestPlaceOrder_ConcurrentCheckoutsDoNotOversell` in
  `internal/checkout`.
- Checkout re-validates and decrements stock inside the same transaction as
  order creation and the payment charge — a "failed" charge rolls back the
  stock decrement too.
- Cart stock caps are advisory (never oversell what's shown), not
  reservations; checkout's re-check is the actual guarantee.

## Testing

`go test ./...` covers the money paths with real SQLite (temp DB per test,
no mocking): checkout happy path, out-of-stock rollback, discount
application/rejection, concurrent-checkout oversell prevention, image
decode/downscale/reject, rate limiter windowing.

UI changes should be verified in an actual browser (desktop + ~390px mobile
width) — type checking and `go test` don't catch template/CSS regressions.

## What's out of scope for v1

Customer accounts (guest checkout + order lookup by number+email instead),
multi-currency, multi-warehouse, nested categories, real payment processor
(mock only), outbound email (order confirmation is a page, not an email),
search beyond `LIKE` + filtered columns.
