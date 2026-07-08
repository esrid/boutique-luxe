package checkout_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/esrid/maison/internal/cart"
	"github.com/esrid/maison/internal/checkout"
	"github.com/esrid/maison/internal/db"
	"github.com/esrid/maison/internal/discount"
	"github.com/esrid/maison/internal/order"
	"github.com/esrid/maison/internal/payment"
)

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func seedVariant(t *testing.T, conn *sql.DB, stock int) int64 {
	t.Helper()
	res, err := conn.Exec(`INSERT INTO products (slug, title, status) VALUES ('t', 'T', 'published')`)
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	productID, _ := res.LastInsertId()

	res, err = conn.Exec(`INSERT INTO product_variants (product_id, sku, name, price_cents, stock_qty) VALUES (?, 'SKU1', 'V', 1000, ?)`,
		productID, stock)
	if err != nil {
		t.Fatalf("seed variant: %v", err)
	}
	variantID, _ := res.LastInsertId()
	return variantID
}

func testInput() checkout.Input {
	return checkout.Input{Email: "a@example.com", Name: "A", Address: "1 St", City: "C", PostalCode: "00000", Country: "US"}
}

func TestPlaceOrder_HappyPath(t *testing.T) {
	ctx := t.Context()
	conn := setupDB(t)
	variantID := seedVariant(t, conn, 5)

	cartStore := cart.NewStore(conn)
	cartID, err := cartStore.Create(ctx, "hash1")
	if err != nil {
		t.Fatalf("create cart: %v", err)
	}
	if err := cartStore.AddItem(ctx, cartID, variantID, 2); err != nil {
		t.Fatalf("add item: %v", err)
	}

	svc := checkout.New(conn, cartStore, order.NewStore(conn), discount.NewStore(conn), payment.MockProvider{})
	ord, err := svc.PlaceOrder(ctx, cartID, testInput())
	if err != nil {
		t.Fatalf("place order: %v", err)
	}
	if ord.Status != order.StatusPaid {
		t.Errorf("status = %q, want paid", ord.Status)
	}
	if ord.TotalCents != 2000 {
		t.Errorf("total = %d, want 2000", ord.TotalCents)
	}
	if ord.PaymentReference == "" {
		t.Error("payment reference not set")
	}

	var stock int
	if err := conn.QueryRow(`SELECT stock_qty FROM product_variants WHERE id = ?`, variantID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 3 {
		t.Errorf("stock = %d, want 3 (5 - 2)", stock)
	}

	c, err := cartStore.Load(ctx, cartID)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Items) != 0 {
		t.Errorf("cart should be cleared after checkout, has %d items", len(c.Items))
	}
}

// TestPlaceOrder_OutOfStockRollsBack simulates a stale cart line (stock
// dropped after the item was added) to prove PlaceOrder's stock re-check
// rejects it and rolls back the whole transaction — no partial order, no
// further stock decrement.
func TestPlaceOrder_OutOfStockRollsBack(t *testing.T) {
	ctx := t.Context()
	conn := setupDB(t)
	variantID := seedVariant(t, conn, 1)

	cartStore := cart.NewStore(conn)
	cartID, err := cartStore.Create(ctx, "hash2")
	if err != nil {
		t.Fatalf("create cart: %v", err)
	}
	if err := cartStore.AddItem(ctx, cartID, variantID, 1); err != nil {
		t.Fatalf("add item: %v", err)
	}
	if _, err := conn.Exec(`UPDATE product_variants SET stock_qty = 0 WHERE id = ?`, variantID); err != nil {
		t.Fatal(err)
	}

	svc := checkout.New(conn, cartStore, order.NewStore(conn), discount.NewStore(conn), payment.MockProvider{})
	_, err = svc.PlaceOrder(ctx, cartID, testInput())
	if !errors.Is(err, checkout.ErrOutOfStock) {
		t.Fatalf("err = %v, want ErrOutOfStock", err)
	}

	var stock int
	if err := conn.QueryRow(`SELECT stock_qty FROM product_variants WHERE id = ?`, variantID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 0 {
		t.Errorf("stock = %d, want unchanged at 0", stock)
	}

	var orderCount int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orderCount); err != nil {
		t.Fatal(err)
	}
	if orderCount != 0 {
		t.Errorf("orders created = %d, want 0 (transaction should have rolled back)", orderCount)
	}
}

func TestPlaceOrder_EmptyCart(t *testing.T) {
	ctx := t.Context()
	conn := setupDB(t)

	cartStore := cart.NewStore(conn)
	cartID, err := cartStore.Create(ctx, "hash3")
	if err != nil {
		t.Fatalf("create cart: %v", err)
	}

	svc := checkout.New(conn, cartStore, order.NewStore(conn), discount.NewStore(conn), payment.MockProvider{})
	_, err = svc.PlaceOrder(ctx, cartID, testInput())
	if !errors.Is(err, checkout.ErrEmptyCart) {
		t.Fatalf("err = %v, want ErrEmptyCart", err)
	}
}

func TestPlaceOrder_AppliesValidDiscountCode(t *testing.T) {
	ctx := t.Context()
	conn := setupDB(t)
	variantID := seedVariant(t, conn, 5)

	cartStore := cart.NewStore(conn)
	cartID, err := cartStore.Create(ctx, "hash4")
	if err != nil {
		t.Fatalf("create cart: %v", err)
	}
	if err := cartStore.AddItem(ctx, cartID, variantID, 2); err != nil { // subtotal 2000
		t.Fatalf("add item: %v", err)
	}

	discountStore := discount.NewStore(conn)
	if _, err := discountStore.Create(ctx, discount.Params{Code: "SAVE20", Type: discount.TypePercent, Value: 20, Active: true}); err != nil {
		t.Fatalf("create discount: %v", err)
	}

	in := testInput()
	in.DiscountCode = "save20" // lowercase — lookup must be case-insensitive
	svc := checkout.New(conn, cartStore, order.NewStore(conn), discountStore, payment.MockProvider{})
	ord, err := svc.PlaceOrder(ctx, cartID, in)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}
	if ord.DiscountCents != 400 {
		t.Errorf("discount = %d, want 400 (20%% of 2000)", ord.DiscountCents)
	}
	if ord.TotalCents != 1600 {
		t.Errorf("total = %d, want 1600 (2000 - 400)", ord.TotalCents)
	}
	if ord.DiscountCode != "SAVE20" {
		t.Errorf("discount code = %q, want %q", ord.DiscountCode, "SAVE20")
	}
}

func TestPlaceOrder_RejectsInvalidDiscountCode(t *testing.T) {
	ctx := t.Context()
	conn := setupDB(t)
	variantID := seedVariant(t, conn, 5)

	cartStore := cart.NewStore(conn)
	cartID, err := cartStore.Create(ctx, "hash5")
	if err != nil {
		t.Fatalf("create cart: %v", err)
	}
	if err := cartStore.AddItem(ctx, cartID, variantID, 1); err != nil {
		t.Fatalf("add item: %v", err)
	}

	in := testInput()
	in.DiscountCode = "DOES-NOT-EXIST"
	svc := checkout.New(conn, cartStore, order.NewStore(conn), discount.NewStore(conn), payment.MockProvider{})
	_, err = svc.PlaceOrder(ctx, cartID, in)
	if !errors.Is(err, checkout.ErrInvalidDiscount) {
		t.Fatalf("err = %v, want ErrInvalidDiscount", err)
	}
}

// TestPlaceOrder_ConcurrentCheckoutsDoNotOversell fires two simultaneous
// checkouts at a variant with exactly one unit of stock. Exactly one must
// win; the other must see ErrOutOfStock; stock must land at zero, never
// negative. This is what internal/db.Open's single-connection pool (see
// the ponytail note on PlaceOrder) is actually buying correctness for.
func TestPlaceOrder_ConcurrentCheckoutsDoNotOversell(t *testing.T) {
	ctx := t.Context()
	conn := setupDB(t)
	variantID := seedVariant(t, conn, 1)

	cartStore := cart.NewStore(conn)
	svc := checkout.New(conn, cartStore, order.NewStore(conn), discount.NewStore(conn), payment.MockProvider{})

	const n = 5

	// Set up every cart fully before any PlaceOrder call runs. Interleaving
	// setup with concurrent PlaceOrder calls would let an early goroutine's
	// stock decrement race ahead of a later cart's AddItem, which caps
	// itself at available stock — starving later carts for a reason that
	// has nothing to do with what this test is actually checking.
	cartIDs := make([]int64, n)
	for i := 0; i < n; i++ {
		cartID, err := cartStore.Create(ctx, "concurrent-hash-"+string(rune('a'+i)))
		if err != nil {
			t.Fatalf("create cart %d: %v", i, err)
		}
		if err := cartStore.AddItem(ctx, cartID, variantID, 1); err != nil {
			t.Fatalf("add item %d: %v", i, err)
		}
		cartIDs[i] = cartID
	}

	results := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			in := testInput()
			in.Email = "concurrent@example.com"
			_, err := svc.PlaceOrder(ctx, cartIDs[i], in)
			results[i] = err
		}(i)
	}
	wg.Wait()

	successes, outOfStock := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, checkout.ErrOutOfStock):
			outOfStock++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Errorf("successes = %d, want exactly 1", successes)
	}
	if outOfStock != n-1 {
		t.Errorf("out-of-stock rejections = %d, want %d", outOfStock, n-1)
	}

	var stock int
	if err := conn.QueryRow(`SELECT stock_qty FROM product_variants WHERE id = ?`, variantID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 0 {
		t.Errorf("final stock = %d, want 0 (never negative, never still 1)", stock)
	}
}
