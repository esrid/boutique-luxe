package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/esrid/maison/internal/discount"
)

type discountsListData struct {
	adminLayout
	Discounts []discount.Code
}

func (h *Handlers) DiscountsList(w http.ResponseWriter, r *http.Request) {
	codes, err := h.discounts.List(r.Context())
	if err != nil {
		h.serverError(w, "list discount codes", err)
		return
	}
	data := discountsListData{adminLayout: h.protectedLayout(r), Discounts: codes}
	h.renderOrErr(w, "templates/admin/discounts_list.html", data)
}

type discountFormData struct {
	adminLayout
	Discount  discount.Code
	FormError string
	IsNew     bool
}

func (h *Handlers) DiscountNew(w http.ResponseWriter, r *http.Request) {
	data := discountFormData{adminLayout: h.protectedLayout(r), IsNew: true, Discount: discount.Code{Type: discount.TypePercent, Active: true}}
	h.renderOrErr(w, "templates/admin/discount_form.html", data)
}

func (h *Handlers) DiscountCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	params, formErr := parseDiscountForm(r)
	if formErr != "" {
		data := discountFormData{adminLayout: h.protectedLayout(r), IsNew: true, FormError: formErr, Discount: formToCode(params)}
		h.renderOrErr(w, "templates/admin/discount_form.html", data)
		return
	}
	if _, err := h.discounts.Create(r.Context(), params); err != nil {
		h.serverError(w, "create discount code", err)
		return
	}
	http.Redirect(w, r, "/admin/discounts", http.StatusSeeOther)
}

func (h *Handlers) DiscountEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	code, err := h.discounts.GetByID(r.Context(), id)
	if errors.Is(err, discount.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.serverError(w, "get discount code", err)
		return
	}
	data := discountFormData{adminLayout: h.protectedLayout(r), Discount: code}
	h.renderOrErr(w, "templates/admin/discount_form.html", data)
}

func (h *Handlers) DiscountUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	params, formErr := parseDiscountForm(r)
	if formErr != "" {
		c := formToCode(params)
		c.ID = id
		data := discountFormData{adminLayout: h.protectedLayout(r), FormError: formErr, Discount: c}
		h.renderOrErr(w, "templates/admin/discount_form.html", data)
		return
	}
	if err := h.discounts.Update(r.Context(), id, params); err != nil {
		h.serverError(w, "update discount code", err)
		return
	}
	http.Redirect(w, r, "/admin/discounts", http.StatusSeeOther)
}

func (h *Handlers) DiscountDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.discounts.Delete(r.Context(), id); err != nil {
		h.serverError(w, "delete discount code", err)
		return
	}
	http.Redirect(w, r, "/admin/discounts", http.StatusSeeOther)
}

func parseDiscountForm(r *http.Request) (discount.Params, string) {
	code := strings.ToUpper(strings.TrimSpace(r.FormValue("code")))
	if code == "" {
		return discount.Params{}, "Code is required."
	}
	typ := discount.Type(r.FormValue("type"))
	if typ != discount.TypePercent && typ != discount.TypeFixed {
		return discount.Params{}, "Invalid discount type."
	}
	var value int64
	var err error
	if typ == discount.TypePercent {
		value, err = strconv.ParseInt(r.FormValue("value"), 10, 64)
		if err != nil || value < 1 || value > 100 {
			return discount.Params{}, "Percent value must be between 1 and 100."
		}
	} else {
		value, err = parseDollarsToCents(r.FormValue("value"))
		if err != nil || value < 1 {
			return discount.Params{}, "Invalid fixed discount amount."
		}
	}

	var expiresAt *time.Time
	if v := strings.TrimSpace(r.FormValue("expires_at")); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return discount.Params{}, "Invalid expiry date."
		}
		expiresAt = &t
	}

	return discount.Params{
		Code:      code,
		Type:      typ,
		Value:     value,
		Active:    r.FormValue("active") == "on",
		ExpiresAt: expiresAt,
	}, ""
}

func formToCode(p discount.Params) discount.Code {
	return discount.Code{Code: p.Code, Type: p.Type, Value: p.Value, Active: p.Active, ExpiresAt: p.ExpiresAt}
}
