package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/esrid/maison/internal/catalog"
)

type inventoryListData struct {
	adminLayout
	Rows      []catalog.InventoryRow
	FormError string
}

func (h *Handlers) InventoryList(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListInventory(r.Context())
	if err != nil {
		h.serverError(w, "list inventory", err)
		return
	}
	data := inventoryListData{adminLayout: h.protectedLayout(r), Rows: rows, FormError: r.URL.Query().Get("error")}
	h.renderOrErr(w, "templates/admin/inventory_list.html", data)
}

func (h *Handlers) InventoryAdjust(w http.ResponseWriter, r *http.Request) {
	variantID, err := strconv.ParseInt(r.PathValue("variantID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	delta, err := strconv.Atoi(r.FormValue("delta"))
	if err != nil {
		http.Redirect(w, r, "/admin/inventory?error=Invalid+quantity.", http.StatusSeeOther)
		return
	}
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		http.Redirect(w, r, "/admin/inventory?error=A+reason+is+required.", http.StatusSeeOther)
		return
	}

	err = h.catalog.AdjustStock(r.Context(), variantID, delta, reason)
	if errors.Is(err, catalog.ErrWouldGoNegative) {
		http.Redirect(w, r, "/admin/inventory?error=That+would+take+stock+below+zero.", http.StatusSeeOther)
		return
	}
	if err != nil {
		h.serverError(w, "adjust stock", err)
		return
	}
	http.Redirect(w, r, "/admin/inventory", http.StatusSeeOther)
}
