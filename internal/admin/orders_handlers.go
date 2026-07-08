package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/esrid/maison/internal/order"
)

type ordersListData struct {
	adminLayout
	Orders       []order.OrderSummary
	StatusFilter order.Status
	Statuses     []order.Status
}

var allStatuses = []order.Status{
	order.StatusPending, order.StatusPaid, order.StatusFulfilled,
	order.StatusShipped, order.StatusDelivered, order.StatusCancelled,
}

func (h *Handlers) OrdersList(w http.ResponseWriter, r *http.Request) {
	status := order.Status(r.URL.Query().Get("status"))
	orders, err := h.orders.ListOrders(r.Context(), status)
	if err != nil {
		h.serverError(w, "list orders", err)
		return
	}
	data := ordersListData{adminLayout: h.protectedLayout(r), Orders: orders, StatusFilter: status, Statuses: allStatuses}
	h.renderOrErr(w, "templates/admin/orders_list.html", data)
}

type orderDetailData struct {
	adminLayout
	Order        *order.Order
	NextStatuses []order.Status
	FormError    string
}

func (h *Handlers) OrderDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ord, err := h.orders.GetByID(r.Context(), id)
	if errors.Is(err, order.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.serverError(w, "get order", err)
		return
	}
	data := orderDetailData{
		adminLayout:  h.protectedLayout(r),
		Order:        ord,
		NextStatuses: nextStatuses(ord.Status),
		FormError:    r.URL.Query().Get("error"),
	}
	h.renderOrErr(w, "templates/admin/order_detail.html", data)
}

func nextStatuses(current order.Status) []order.Status {
	var out []order.Status
	for _, s := range allStatuses {
		if order.CanTransition(current, s) {
			out = append(out, s)
		}
	}
	return out
}

func (h *Handlers) OrderUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	newStatus := order.Status(r.FormValue("status"))
	if err := h.orders.UpdateStatus(r.Context(), id, newStatus); err != nil {
		if errors.Is(err, order.ErrInvalidTransition) {
			http.Redirect(w, r, orderDetailURL(id)+"?error="+err.Error(), http.StatusSeeOther)
			return
		}
		h.serverError(w, "update order status", err)
		return
	}
	http.Redirect(w, r, orderDetailURL(id), http.StatusSeeOther)
}

func (h *Handlers) OrderUpdateTracking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	tracking := strings.TrimSpace(r.FormValue("tracking_number"))
	if err := h.orders.UpdateTracking(r.Context(), id, tracking); err != nil {
		h.serverError(w, "update tracking", err)
		return
	}
	http.Redirect(w, r, orderDetailURL(id), http.StatusSeeOther)
}

func orderDetailURL(id int64) string {
	return "/admin/orders/" + strconv.FormatInt(id, 10)
}
