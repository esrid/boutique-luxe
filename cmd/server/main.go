package main

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/esrid/maison/internal/admin"
	"github.com/esrid/maison/internal/auth"
	"github.com/esrid/maison/internal/cart"
	"github.com/esrid/maison/internal/catalog"
	"github.com/esrid/maison/internal/checkout"
	"github.com/esrid/maison/internal/config"
	"github.com/esrid/maison/internal/db"
	"github.com/esrid/maison/internal/discount"
	"github.com/esrid/maison/internal/httpx"
	"github.com/esrid/maison/internal/money"
	"github.com/esrid/maison/internal/order"
	"github.com/esrid/maison/internal/payment"
	"github.com/esrid/maison/internal/render"
	"github.com/esrid/maison/internal/storefront"
	"github.com/esrid/maison/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	conn, err := db.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := os.MkdirAll(cfg.UploadsDir, 0o755); err != nil {
		return err
	}

	funcs := template.FuncMap{
		"money":      money.FormatCents,
		"moneyPlain": money.FormatCentsPlain,
		// isSelectedCategory compares a product's nullable category ID
		// against a <select> option's ID — html/template's eq can't compare
		// *int64 to int64 directly.
		"isSelectedCategory": func(p *int64, id int64) bool { return p != nil && *p == id },
		// formatOptions renders a variant's options map back into the
		// "key=value, key=value" shorthand the admin form accepts.
		"formatOptions": func(m map[string]string) string {
			keys := slices.Sorted(maps.Keys(m))
			parts := make([]string, len(keys))
			for i, k := range keys {
				parts[i] = k + "=" + m[k]
			}
			return strings.Join(parts, ", ")
		},
	}
	// Two renderers, one per layout: a single glob would parse both
	// base.html and admin_base.html into one template set, and since both
	// define {{"layout"}} the later one silently wins for every page.
	renderer := render.New(web.Templates, "templates/layout/base.html", funcs)
	adminRenderer := render.New(web.Templates, "templates/layout/admin_base.html", funcs)

	catalogStore := catalog.NewStore(conn)
	cartStore := cart.NewStore(conn)
	orderStore := order.NewStore(conn)
	discountStore := discount.NewStore(conn)
	checkoutSvc := checkout.New(conn, cartStore, orderStore, discountStore, payment.MockProvider{})
	adminStore := admin.NewStore(conn)
	signer := auth.NewSigner(cfg.SessionKey)
	storefrontH := storefront.New(renderer, catalogStore, cartStore, checkoutSvc, orderStore)
	adminH := admin.New(adminRenderer, adminStore, catalogStore, orderStore, discountStore, cfg.UploadsDir, signer, cfg.IsProd())

	storefrontMux := http.NewServeMux()
	storefrontMux.HandleFunc("GET /{$}", storefrontH.Home)
	storefrontMux.HandleFunc("GET /products", storefrontH.Products)
	storefrontMux.HandleFunc("GET /products/{slug}", storefrontH.ProductDetail)
	storefrontMux.HandleFunc("GET /cart", storefrontH.CartPage)
	storefrontMux.HandleFunc("POST /cart/items", storefrontH.AddToCart)
	storefrontMux.HandleFunc("POST /cart/items/{variantID}", storefrontH.UpdateCartItem)
	storefrontMux.HandleFunc("POST /cart/items/{variantID}/remove", storefrontH.RemoveCartItem)
	storefrontMux.HandleFunc("GET /checkout", storefrontH.CheckoutPage)
	storefrontMux.HandleFunc("POST /checkout", storefrontH.PlaceOrder)
	storefrontMux.HandleFunc("GET /orders/lookup", storefrontH.OrderLookupPage)
	storefrontMux.HandleFunc("POST /orders/lookup", storefrontH.OrderLookupSubmit)
	storefrontMux.HandleFunc("GET /orders/{orderNumber}", storefrontH.OrderConfirmation)
	storefrontHandler := cart.Middleware(cartStore, signer, cfg.IsProd())(storefrontMux)

	requireAdmin := admin.RequireAuth(adminStore, signer)
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("GET /admin/login", adminH.LoginPage)
	adminMux.HandleFunc("POST /admin/login", adminH.LoginSubmit)
	adminMux.Handle("GET /admin", requireAdmin(http.HandlerFunc(adminH.Dashboard)))
	adminMux.Handle("POST /admin/logout", requireAdmin(http.HandlerFunc(adminH.Logout)))
	adminMux.Handle("GET /admin/categories", requireAdmin(http.HandlerFunc(adminH.CategoriesList)))
	adminMux.Handle("GET /admin/categories/new", requireAdmin(http.HandlerFunc(adminH.CategoryNew)))
	adminMux.Handle("POST /admin/categories", requireAdmin(http.HandlerFunc(adminH.CategoryCreate)))
	adminMux.Handle("GET /admin/categories/{id}/edit", requireAdmin(http.HandlerFunc(adminH.CategoryEdit)))
	adminMux.Handle("POST /admin/categories/{id}", requireAdmin(http.HandlerFunc(adminH.CategoryUpdate)))
	adminMux.Handle("POST /admin/categories/{id}/delete", requireAdmin(http.HandlerFunc(adminH.CategoryDelete)))
	adminMux.Handle("GET /admin/products", requireAdmin(http.HandlerFunc(adminH.ProductsList)))
	adminMux.Handle("GET /admin/products/new", requireAdmin(http.HandlerFunc(adminH.ProductNew)))
	adminMux.Handle("POST /admin/products", requireAdmin(http.HandlerFunc(adminH.ProductCreate)))
	adminMux.Handle("GET /admin/products/{id}/edit", requireAdmin(http.HandlerFunc(adminH.ProductEdit)))
	adminMux.Handle("POST /admin/products/{id}", requireAdmin(http.HandlerFunc(adminH.ProductUpdate)))
	adminMux.Handle("POST /admin/products/{id}/delete", requireAdmin(http.HandlerFunc(adminH.ProductDelete)))
	adminMux.Handle("POST /admin/products/{id}/variants", requireAdmin(http.HandlerFunc(adminH.VariantCreate)))
	adminMux.Handle("POST /admin/products/{id}/variants/{variantID}", requireAdmin(http.HandlerFunc(adminH.VariantUpdate)))
	adminMux.Handle("POST /admin/products/{id}/variants/{variantID}/delete", requireAdmin(http.HandlerFunc(adminH.VariantDelete)))
	adminMux.Handle("POST /admin/products/{id}/images", requireAdmin(http.HandlerFunc(adminH.ImageUpload)))
	adminMux.Handle("POST /admin/products/{id}/images/{imageID}/delete", requireAdmin(http.HandlerFunc(adminH.ImageDelete)))
	adminMux.Handle("POST /admin/products/{id}/images/{imageID}/move", requireAdmin(http.HandlerFunc(adminH.ImageMove)))
	adminMux.Handle("GET /admin/orders", requireAdmin(http.HandlerFunc(adminH.OrdersList)))
	adminMux.Handle("GET /admin/orders/{id}", requireAdmin(http.HandlerFunc(adminH.OrderDetail)))
	adminMux.Handle("POST /admin/orders/{id}/status", requireAdmin(http.HandlerFunc(adminH.OrderUpdateStatus)))
	adminMux.Handle("POST /admin/orders/{id}/tracking", requireAdmin(http.HandlerFunc(adminH.OrderUpdateTracking)))
	adminMux.Handle("GET /admin/inventory", requireAdmin(http.HandlerFunc(adminH.InventoryList)))
	adminMux.Handle("POST /admin/inventory/{variantID}/adjust", requireAdmin(http.HandlerFunc(adminH.InventoryAdjust)))
	adminMux.Handle("GET /admin/discounts", requireAdmin(http.HandlerFunc(adminH.DiscountsList)))
	adminMux.Handle("GET /admin/discounts/new", requireAdmin(http.HandlerFunc(adminH.DiscountNew)))
	adminMux.Handle("POST /admin/discounts", requireAdmin(http.HandlerFunc(adminH.DiscountCreate)))
	adminMux.Handle("GET /admin/discounts/{id}/edit", requireAdmin(http.HandlerFunc(adminH.DiscountEdit)))
	adminMux.Handle("POST /admin/discounts/{id}", requireAdmin(http.HandlerFunc(adminH.DiscountUpdate)))
	adminMux.Handle("POST /admin/discounts/{id}/delete", requireAdmin(http.HandlerFunc(adminH.DiscountDelete)))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz(conn))
	mux.Handle("GET /static/", staticHandler())
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadsDir))))
	mux.Handle("/admin/", adminMux)
	mux.Handle("/admin", adminMux)
	mux.Handle("/", storefrontHandler)

	// CrossOriginProtection (Go 1.25+ stdlib) rejects non-safe cross-origin
	// requests via Sec-Fetch-Site/Origin headers — no tokens, no cookies,
	// replaces what would otherwise be a hand-rolled CSRF scheme.
	csrf := http.NewCrossOriginProtection()
	handler := httpx.Chain(httpx.Recover, httpx.Logging, httpx.SecurityHeaders(cfg.IsProd()))(csrf.Handler(mux))

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// healthz pings the DB so a load balancer/orchestrator can tell a broken
// dependency apart from a merely slow one — a 200 that doesn't check its
// dependencies is a false "healthy" during a DB outage.
func healthz(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := conn.PingContext(ctx); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func staticHandler() http.Handler {
	sub, err := fs.Sub(web.Static, "static")
	if err != nil {
		panic(err) // build-time asset layout is wrong; fail fast
	}
	return http.StripPrefix("/static/", http.FileServerFS(sub))
}
