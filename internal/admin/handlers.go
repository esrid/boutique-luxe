package admin

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/esrid/maison/internal/auth"
	"github.com/esrid/maison/internal/catalog"
	"github.com/esrid/maison/internal/discount"
	"github.com/esrid/maison/internal/httpx"
	"github.com/esrid/maison/internal/order"
	"github.com/esrid/maison/internal/render"
)

type Handlers struct {
	render     *render.Renderer
	store      *Store
	catalog    *catalog.Store
	orders     *order.Store
	discounts  *discount.Store
	uploadsDir string
	signer     auth.Signer
	secure     bool
	loginLimit *httpx.RateLimiter
}

func New(renderer *render.Renderer, store *Store, catalogStore *catalog.Store, orderStore *order.Store, discountStore *discount.Store, uploadsDir string, signer auth.Signer, secure bool) *Handlers {
	return &Handlers{
		render: renderer, store: store, catalog: catalogStore, orders: orderStore, discounts: discountStore,
		uploadsDir: uploadsDir, signer: signer, secure: secure,
		// 5 attempts per 15 minutes per client IP — enough to survive a
		// mistyped password a few times, not enough to brute-force.
		loginLimit: httpx.NewRateLimiter(5, 15*time.Minute),
	}
}

type adminLayout struct {
	Admin *User // nil on the login page
}

func (h *Handlers) layout(r *http.Request) adminLayout {
	return adminLayout{}
}

// protectedLayout is layout() plus the logged-in admin — every handler
// behind RequireAuth uses this instead of layout() so the sidebar can show
// who's logged in.
func (h *Handlers) protectedLayout(r *http.Request) adminLayout {
	l := h.layout(r)
	user := UserFromContext(r.Context())
	l.Admin = &user
	return l
}

type loginData struct {
	adminLayout
	FormError string
	Next      string
}

func (h *Handlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	data := loginData{adminLayout: h.layout(r), Next: r.URL.Query().Get("next")}
	h.renderOrErr(w, "templates/admin/login.html", data)
}

func (h *Handlers) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	next := r.FormValue("next")

	if !h.loginLimit.Allow(clientIP(r)) {
		data := loginData{adminLayout: h.layout(r), FormError: "Too many attempts. Try again later.", Next: next}
		h.renderOrErr(w, "templates/admin/login.html", data)
		return
	}

	user, hash, err := h.store.FindByEmail(r.Context(), email)
	if err == nil && !auth.CheckPassword(hash, password) {
		err = ErrInvalidCredentials
	}
	if errors.Is(err, ErrInvalidCredentials) {
		data := loginData{adminLayout: h.layout(r), FormError: "Invalid email or password.", Next: next}
		h.renderOrErr(w, "templates/admin/login.html", data)
		return
	}
	if err != nil {
		slog.Error("admin login", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken()
	if err != nil {
		slog.Error("generate session token", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := h.store.CreateSession(r.Context(), user.ID, auth.HashToken(token), sessionTTL); err != nil {
		slog.Error("create admin session", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    h.signer.Sign(token),
		Path:     "/admin",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionTTL),
	})

	dest := "/admin"
	if next != "" && strings.HasPrefix(next, "/admin") {
		dest = next
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		if token, err := h.signer.Verify(c.Value); err == nil {
			_ = h.store.DeleteSession(r.Context(), auth.HashToken(token))
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/admin", MaxAge: -1,
		HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, loginPath, http.StatusSeeOther)
}

type dashboardData struct {
	adminLayout
	RevenueTodayCents int64
	RevenueWeekCents  int64
	OrderCount        int
	LowStock          []catalog.InventoryRow
	RecentOrders      []order.OrderSummary
}

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	revenueToday, err := h.orders.RevenueSince(ctx, startOfDay)
	if err != nil {
		h.serverError(w, "revenue today", err)
		return
	}
	revenueWeek, err := h.orders.RevenueSince(ctx, startOfDay.AddDate(0, 0, -7))
	if err != nil {
		h.serverError(w, "revenue week", err)
		return
	}
	orderCount, err := h.orders.CountOrders(ctx)
	if err != nil {
		h.serverError(w, "count orders", err)
		return
	}
	lowStock, err := h.catalog.LowStockVariants(ctx, 10)
	if err != nil {
		h.serverError(w, "low stock", err)
		return
	}
	recent, err := h.orders.RecentOrders(ctx, 10)
	if err != nil {
		h.serverError(w, "recent orders", err)
		return
	}

	data := dashboardData{
		adminLayout:       h.protectedLayout(r),
		RevenueTodayCents: revenueToday,
		RevenueWeekCents:  revenueWeek,
		OrderCount:        orderCount,
		LowStock:          lowStock,
		RecentOrders:      recent,
	}
	h.renderOrErr(w, "templates/admin/dashboard.html", data)
}

func (h *Handlers) renderOrErr(w http.ResponseWriter, page string, data any) {
	if err := h.render.Render(w, http.StatusOK, page, data); err != nil {
		slog.Error("render", "page", page, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// clientIP returns the request's IP for rate-limiting purposes.
//
// ponytail: reads RemoteAddr only, not X-Forwarded-For — trusting that
// header without a known, configured trusted-proxy list lets an attacker
// spoof a fresh IP on every request and bypass the limiter entirely. If
// this app ends up behind a reverse proxy (Traefik/Dokploy), add an
// explicit trusted-proxy check before trusting forwarded headers.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
