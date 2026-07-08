// Package storefront holds the public-facing HTTP handlers: browsing,
// cart, checkout.
package storefront

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/esrid/maison/internal/cart"
	"github.com/esrid/maison/internal/catalog"
	"github.com/esrid/maison/internal/checkout"
	"github.com/esrid/maison/internal/order"
	"github.com/esrid/maison/internal/render"
)

type Handlers struct {
	render   *render.Renderer
	catalog  *catalog.Store
	cart     *cart.Store
	checkout *checkout.Service
	orders   *order.Store
}

func New(renderer *render.Renderer, catalogStore *catalog.Store, cartStore *cart.Store, checkoutSvc *checkout.Service, orderStore *order.Store) *Handlers {
	return &Handlers{render: renderer, catalog: catalogStore, cart: cartStore, checkout: checkoutSvc, orders: orderStore}
}

// layout builds the common per-page fields (footer year, mini-cart badge)
// for the current request's cart.
func (h *Handlers) layout(r *http.Request) render.Layout {
	count, err := h.cart.Count(r.Context(), cart.IDFromContext(r.Context()))
	if err != nil {
		slog.Error("count cart items", "err", err)
	}
	return render.NewLayout(count)
}

type homeData struct {
	render.Layout
	Collections []catalog.Category
}

func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	collections, err := h.catalog.ListCategories(r.Context())
	if err != nil {
		slog.Error("list categories for home", "err", err)
		// non-fatal: show home page without collections
	}
	data := homeData{Layout: h.layout(r), Collections: collections}
	h.renderOrErr(w, "templates/storefront/home.html", data)
}

type productsData struct {
	render.Layout
	Products   []catalog.ProductSummary
	Categories []catalog.Category
	Filter     catalog.ProductFilter
	Total      int
	PrevURL    string // "" if no previous page
	NextURL    string // "" if no next page
}

func (h *Handlers) Products(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	filter := catalog.ProductFilter{
		CategorySlug:  q.Get("category"),
		MinPriceCents: parseDollarsToCents(q.Get("min")),
		MaxPriceCents: parseDollarsToCents(q.Get("max")),
		Sort:          catalog.Sort(q.Get("sort")),
		Page:          parsePositiveInt(q.Get("page"), 1),
		PageSize:      24,
	}

	products, total, err := h.catalog.ListProducts(ctx, filter)
	if err != nil {
		slog.Error("list products", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	categories, err := h.catalog.ListCategories(ctx)
	if err != nil {
		slog.Error("list categories", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := productsData{
		Layout:     h.layout(r),
		Products:   products,
		Categories: categories,
		Filter:     filter,
		Total:      total,
	}
	if filter.Page > 1 {
		data.PrevURL = pageURL(q, filter.Page-1)
	}
	if filter.Page*filter.PageSize < total {
		data.NextURL = pageURL(q, filter.Page+1)
	}
	h.renderOrErr(w, "templates/storefront/products.html", data)
}

// pageURL rebuilds the current filter query string with only "page"
// swapped, so pager links don't drop the active category/price/sort.
func pageURL(q url.Values, page int) string {
	next := q.Clone()
	next.Set("page", strconv.Itoa(page))
	return "/products?" + next.Encode()
}

type productDetailData struct {
	render.Layout
	Product catalog.Product
}

func (h *Handlers) ProductDetail(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	product, err := h.catalog.GetProductBySlug(r.Context(), slug)
	if errors.Is(err, catalog.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		slog.Error("get product", "err", err, "slug", slug)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := productDetailData{Layout: h.layout(r), Product: *product}
	h.renderOrErr(w, "templates/storefront/product.html", data)
}

type cartPageData struct {
	render.Layout
	Cart *cart.Cart
}

func (h *Handlers) CartPage(w http.ResponseWriter, r *http.Request) {
	c, err := h.cart.Load(r.Context(), cart.IDFromContext(r.Context()))
	if err != nil {
		slog.Error("load cart", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	data := cartPageData{Layout: h.layout(r), Cart: c}
	h.renderOrErr(w, "templates/storefront/cart.html", data)
}

func (h *Handlers) AddToCart(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	variantID, err := strconv.ParseInt(r.FormValue("variant_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid variant", http.StatusBadRequest)
		return
	}
	qty := parsePositiveInt(r.FormValue("qty"), 1)

	if err := h.cart.AddItem(r.Context(), cart.IDFromContext(r.Context()), variantID, qty); err != nil {
		slog.Error("add to cart", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func (h *Handlers) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	variantID, err := strconv.ParseInt(r.PathValue("variantID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid variant", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	qty := parsePositiveInt(r.FormValue("qty"), 1)

	if err := h.cart.UpdateItemQty(r.Context(), cart.IDFromContext(r.Context()), variantID, qty); err != nil {
		slog.Error("update cart item", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func (h *Handlers) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	variantID, err := strconv.ParseInt(r.PathValue("variantID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid variant", http.StatusBadRequest)
		return
	}
	if err := h.cart.RemoveItem(r.Context(), cart.IDFromContext(r.Context()), variantID); err != nil {
		slog.Error("remove cart item", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

type checkoutPageData struct {
	render.Layout
	Cart      *cart.Cart
	FormError string
	Input     checkout.Input
}

func (h *Handlers) CheckoutPage(w http.ResponseWriter, r *http.Request) {
	c, err := h.cart.Load(r.Context(), cart.IDFromContext(r.Context()))
	if err != nil {
		slog.Error("load cart", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(c.Items) == 0 {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}
	data := checkoutPageData{Layout: h.layout(r), Cart: c}
	h.renderOrErr(w, "templates/storefront/checkout.html", data)
}

func (h *Handlers) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	in := checkout.Input{
		Email:        strings.TrimSpace(r.FormValue("email")),
		Name:         strings.TrimSpace(r.FormValue("name")),
		Address:      strings.TrimSpace(r.FormValue("address")),
		City:         strings.TrimSpace(r.FormValue("city")),
		PostalCode:   strings.TrimSpace(r.FormValue("postal_code")),
		Country:      strings.TrimSpace(r.FormValue("country")),
		DiscountCode: strings.TrimSpace(r.FormValue("discount_code")),
	}

	if formErr := validateCheckoutInput(in); formErr != "" {
		h.reshowCheckout(w, r, in, formErr)
		return
	}

	ord, err := h.checkout.PlaceOrder(r.Context(), cart.IDFromContext(r.Context()), in)
	switch {
	case errors.Is(err, checkout.ErrEmptyCart):
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	case errors.Is(err, checkout.ErrOutOfStock):
		h.reshowCheckout(w, r, in, err.Error())
		return
	case errors.Is(err, checkout.ErrInvalidDiscount):
		h.reshowCheckout(w, r, in, "Ce code promo est invalide ou expiré.")
		return
	case err != nil:
		slog.Error("place order", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	dest := "/orders/" + url.PathEscape(ord.OrderNumber) + "?email=" + url.QueryEscape(ord.Email)
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func (h *Handlers) reshowCheckout(w http.ResponseWriter, r *http.Request, in checkout.Input, formErr string) {
	c, err := h.cart.Load(r.Context(), cart.IDFromContext(r.Context()))
	if err != nil {
		slog.Error("load cart", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	data := checkoutPageData{Layout: h.layout(r), Cart: c, FormError: formErr, Input: in}
	h.renderOrErr(w, "templates/storefront/checkout.html", data)
}

func validateCheckoutInput(in checkout.Input) string {
	if in.Email == "" || !strings.Contains(in.Email, "@") {
		return "Entrez une adresse email valide."
	}
	if in.Name == "" || in.Address == "" || in.City == "" || in.PostalCode == "" || in.Country == "" {
		return "Remplissez tous les champs de livraison."
	}
	return ""
}

type orderConfirmationData struct {
	render.Layout
	Order *order.Order
}

func (h *Handlers) OrderConfirmation(w http.ResponseWriter, r *http.Request) {
	orderNumber := r.PathValue("orderNumber")
	email := r.URL.Query().Get("email")

	ord, err := h.orders.GetByNumberAndEmail(r.Context(), orderNumber, email)
	if errors.Is(err, order.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		slog.Error("get order", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := orderConfirmationData{Layout: h.layout(r), Order: ord}
	h.renderOrErr(w, "templates/storefront/order_confirmation.html", data)
}

type orderLookupData struct {
	render.Layout
	FormError string
}

func (h *Handlers) OrderLookupPage(w http.ResponseWriter, r *http.Request) {
	data := orderLookupData{Layout: h.layout(r)}
	h.renderOrErr(w, "templates/storefront/order_lookup.html", data)
}

func (h *Handlers) OrderLookupSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	orderNumber := strings.TrimSpace(r.FormValue("order_number"))
	email := strings.TrimSpace(r.FormValue("email"))

	if _, err := h.orders.GetByNumberAndEmail(r.Context(), orderNumber, email); err != nil {
		data := orderLookupData{Layout: h.layout(r), FormError: "Aucune commande trouvée avec ce numéro et cet email."}
		h.renderOrErr(w, "templates/storefront/order_lookup.html", data)
		return
	}
	dest := "/orders/" + url.PathEscape(orderNumber) + "?email=" + url.QueryEscape(email)
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func (h *Handlers) renderOrErr(w http.ResponseWriter, page string, data any) {
	if err := h.render.Render(w, http.StatusOK, page, data); err != nil {
		slog.Error("render", "page", page, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// parseDollarsToCents reads a plain-dollar query param ("29" or "29.99")
// into cents; empty or invalid input means "no bound" (0). Parsed as
// integers, never float64 — money never goes through floating point here,
// even for a filter bound.
func parseDollarsToCents(s string) int64 {
	if s == "" {
		return 0
	}
	whole, frac, _ := strings.Cut(s, ".")
	dollars, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || dollars < 0 {
		return 0
	}
	cents := int64(0)
	if frac != "" {
		frac = (frac + "00")[:2]
		c, err := strconv.ParseInt(frac, 10, 64)
		if err != nil || c < 0 {
			return 0
		}
		cents = c
	}
	return dollars*100 + cents
}

func parsePositiveInt(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
