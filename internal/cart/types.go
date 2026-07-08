// Package cart owns the guest shopping cart: one row per browser (resolved
// via a signed cookie, see Middleware), line items, and totals. Stock is
// NOT reserved when added to a cart — only capped so the cart never shows
// a quantity that isn't buyable. Checkout (internal/checkout) does the
// atomic reserve-and-decrement.
package cart

type Item struct {
	VariantID      int64
	ProductSlug    string
	ProductTitle   string
	VariantName    string
	UnitPriceCents int64
	Qty            int
	StockQty       int
	LineTotalCents int64
}

type Cart struct {
	Items         []Item
	SubtotalCents int64
}
