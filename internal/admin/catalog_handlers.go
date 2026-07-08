package admin

import (
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/esrid/maison/internal/catalog"
	"github.com/esrid/maison/internal/media"
)

// --- Categories ---------------------------------------------------------

type categoriesListData struct {
	adminLayout
	Categories []catalog.Category
}

func (h *Handlers) CategoriesList(w http.ResponseWriter, r *http.Request) {
	cats, err := h.catalog.ListCategories(r.Context())
	if err != nil {
		h.serverError(w, "list categories", err)
		return
	}
	data := categoriesListData{adminLayout: h.protectedLayout(r), Categories: cats}
	h.renderOrErr(w, "templates/admin/categories_list.html", data)
}

type categoryFormData struct {
	adminLayout
	Category  catalog.Category
	FormError string
	IsNew     bool
}

func (h *Handlers) CategoryNew(w http.ResponseWriter, r *http.Request) {
	data := categoryFormData{adminLayout: h.protectedLayout(r), IsNew: true}
	h.renderOrErr(w, "templates/admin/category_form.html", data)
}

func (h *Handlers) CategoryCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	slug := strings.TrimSpace(r.FormValue("slug"))
	name := strings.TrimSpace(r.FormValue("name"))
	if slug == "" || name == "" {
		data := categoryFormData{
			adminLayout: h.protectedLayout(r), IsNew: true, FormError: "Slug and name are required.",
			Category: catalog.Category{Slug: slug, Name: name},
		}
		h.renderOrErr(w, "templates/admin/category_form.html", data)
		return
	}
	if _, err := h.catalog.CreateCategory(r.Context(), slug, name); err != nil {
		h.serverError(w, "create category", err)
		return
	}
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

func (h *Handlers) CategoryEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cat, err := h.catalog.GetCategory(r.Context(), id)
	if errors.Is(err, catalog.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.serverError(w, "get category", err)
		return
	}
	data := categoryFormData{adminLayout: h.protectedLayout(r), Category: cat}
	h.renderOrErr(w, "templates/admin/category_form.html", data)
}

func (h *Handlers) CategoryUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	slug := strings.TrimSpace(r.FormValue("slug"))
	name := strings.TrimSpace(r.FormValue("name"))
	if slug == "" || name == "" {
		data := categoryFormData{
			adminLayout: h.protectedLayout(r), FormError: "Slug and name are required.",
			Category: catalog.Category{ID: id, Slug: slug, Name: name},
		}
		h.renderOrErr(w, "templates/admin/category_form.html", data)
		return
	}
	if err := h.catalog.UpdateCategory(r.Context(), id, slug, name); err != nil {
		h.serverError(w, "update category", err)
		return
	}
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

func (h *Handlers) CategoryDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.catalog.DeleteCategory(r.Context(), id); err != nil {
		h.serverError(w, "delete category", err)
		return
	}
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

// --- Products ------------------------------------------------------------

type productsListData struct {
	adminLayout
	Products []catalog.AdminProductRow
}

func (h *Handlers) ProductsList(w http.ResponseWriter, r *http.Request) {
	products, err := h.catalog.ListAllProducts(r.Context())
	if err != nil {
		h.serverError(w, "list products", err)
		return
	}
	data := productsListData{adminLayout: h.protectedLayout(r), Products: products}
	h.renderOrErr(w, "templates/admin/products_list.html", data)
}

type productFormData struct {
	adminLayout
	Product    catalog.Product
	Categories []catalog.Category
	FormError  string
	IsNew      bool
}

func (h *Handlers) ProductNew(w http.ResponseWriter, r *http.Request) {
	cats, err := h.catalog.ListCategories(r.Context())
	if err != nil {
		h.serverError(w, "list categories", err)
		return
	}
	data := productFormData{adminLayout: h.protectedLayout(r), Categories: cats, IsNew: true,
		Product: catalog.Product{Status: catalog.StatusDraft}}
	h.renderOrErr(w, "templates/admin/product_form.html", data)
}

func (h *Handlers) ProductCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	params, formErr := parseProductForm(r)
	if formErr != "" {
		h.reshowProductForm(w, r, catalog.Product{}, true, formErr)
		return
	}
	id, err := h.catalog.CreateProduct(r.Context(), params)
	if err != nil {
		h.serverError(w, "create product", err)
		return
	}
	http.Redirect(w, r, "/admin/products/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

func (h *Handlers) ProductEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	product, err := h.catalog.GetProductByID(r.Context(), id)
	if errors.Is(err, catalog.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.serverError(w, "get product", err)
		return
	}
	cats, err := h.catalog.ListCategories(r.Context())
	if err != nil {
		h.serverError(w, "list categories", err)
		return
	}
	data := productFormData{adminLayout: h.protectedLayout(r), Product: *product, Categories: cats}
	h.renderOrErr(w, "templates/admin/product_form.html", data)
}

func (h *Handlers) ProductUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	params, formErr := parseProductForm(r)
	if formErr != "" {
		product, _ := h.catalog.GetProductByID(r.Context(), id)
		if product == nil {
			product = &catalog.Product{}
		}
		h.reshowProductForm(w, r, *product, false, formErr)
		return
	}
	if err := h.catalog.UpdateProduct(r.Context(), id, params); err != nil {
		h.serverError(w, "update product", err)
		return
	}
	http.Redirect(w, r, "/admin/products/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

func (h *Handlers) reshowProductForm(w http.ResponseWriter, r *http.Request, product catalog.Product, isNew bool, formErr string) {
	cats, err := h.catalog.ListCategories(r.Context())
	if err != nil {
		h.serverError(w, "list categories", err)
		return
	}
	data := productFormData{adminLayout: h.protectedLayout(r), Product: product, Categories: cats, IsNew: isNew, FormError: formErr}
	h.renderOrErr(w, "templates/admin/product_form.html", data)
}

func parseProductForm(r *http.Request) (catalog.ProductParams, string) {
	slug := strings.TrimSpace(r.FormValue("slug"))
	title := strings.TrimSpace(r.FormValue("title"))
	if slug == "" || title == "" {
		return catalog.ProductParams{}, "Slug and title are required."
	}
	status := catalog.StatusDraft
	if r.FormValue("status") == string(catalog.StatusPublished) {
		status = catalog.StatusPublished
	}
	var categoryID *int64
	if v := r.FormValue("category_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return catalog.ProductParams{}, "Invalid category."
		}
		categoryID = &id
	}
	return catalog.ProductParams{
		CategoryID:  categoryID,
		Slug:        slug,
		Title:       title,
		Description: strings.TrimSpace(r.FormValue("description")),
		Status:      status,
	}, ""
}

func (h *Handlers) ProductDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.catalog.DeleteProduct(r.Context(), id); err != nil {
		h.serverError(w, "delete product", err)
		return
	}
	http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
}

// --- Variants (nested under a product) -----------------------------------

func (h *Handlers) VariantCreate(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	params, formErr := parseVariantForm(r)
	if formErr != "" {
		http.Redirect(w, r, editURL(productID)+"?error="+formErr, http.StatusSeeOther)
		return
	}
	if _, err := h.catalog.CreateVariant(r.Context(), productID, params); err != nil {
		h.serverError(w, "create variant", err)
		return
	}
	http.Redirect(w, r, editURL(productID), http.StatusSeeOther)
}

func (h *Handlers) VariantUpdate(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	variantID, err := strconv.ParseInt(r.PathValue("variantID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	params, formErr := parseVariantForm(r)
	if formErr != "" {
		http.Redirect(w, r, editURL(productID)+"?error="+formErr, http.StatusSeeOther)
		return
	}
	if err := h.catalog.UpdateVariant(r.Context(), variantID, params); err != nil {
		h.serverError(w, "update variant", err)
		return
	}
	http.Redirect(w, r, editURL(productID), http.StatusSeeOther)
}

func (h *Handlers) VariantDelete(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	variantID, err := strconv.ParseInt(r.PathValue("variantID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.catalog.DeleteVariant(r.Context(), variantID); err != nil {
		h.serverError(w, "delete variant", err)
		return
	}
	http.Redirect(w, r, editURL(productID), http.StatusSeeOther)
}

func parseVariantForm(r *http.Request) (catalog.VariantParams, string) {
	sku := strings.TrimSpace(r.FormValue("sku"))
	name := strings.TrimSpace(r.FormValue("name"))
	if sku == "" || name == "" {
		return catalog.VariantParams{}, "SKU and name are required."
	}
	price, err := parseDollarsToCents(r.FormValue("price"))
	if err != nil {
		return catalog.VariantParams{}, "Invalid price."
	}
	stock, err := strconv.Atoi(r.FormValue("stock_qty"))
	if err != nil || stock < 0 {
		return catalog.VariantParams{}, "Invalid stock quantity."
	}
	threshold, err := strconv.Atoi(r.FormValue("low_stock_threshold"))
	if err != nil || threshold < 0 {
		threshold = 5
	}
	return catalog.VariantParams{
		SKU: sku, Name: name, Options: parseOptions(r.FormValue("options")),
		PriceCents: price, StockQty: stock, LowStockThreshold: threshold,
	}, ""
}

// parseOptions turns "size=M, color=Red" into {"size":"M","color":"Red"}.
func parseOptions(s string) map[string]string {
	out := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok || strings.TrimSpace(k) == "" {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

// parseDollarsToCents parses a plain-dollar amount ("29.99") into cents as
// an integer — no float64 anywhere near money.
func parseDollarsToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty price")
	}
	whole, frac, _ := strings.Cut(s, ".")
	dollars, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || dollars < 0 {
		return 0, errors.New("invalid price")
	}
	cents := int64(0)
	if frac != "" {
		frac = (frac + "00")[:2]
		c, err := strconv.ParseInt(frac, 10, 64)
		if err != nil || c < 0 {
			return 0, errors.New("invalid price")
		}
		cents = c
	}
	return dollars*100 + cents, nil
}

// --- Images (nested under a product) --------------------------------------

func (h *Handlers) ImageUpload(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// 10 MiB cap per request; media.Save re-encodes anyway so oversized
	// dimensions get downscaled, this just bounds the upload itself.
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Redirect(w, r, editURL(productID)+"?error=Upload too large.", http.StatusSeeOther)
		return
	}
	alt := strings.TrimSpace(r.FormValue("alt"))

	files := r.MultipartForm.File["images"]
	for _, fh := range files {
		if err := h.saveOneImage(r, productID, fh, alt); err != nil {
			slog.Error("upload image", "err", err)
			http.Redirect(w, r, editURL(productID)+"?error=One or more images could not be processed.", http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, editURL(productID), http.StatusSeeOther)
}

func (h *Handlers) saveOneImage(r *http.Request, productID int64, fh *multipart.FileHeader, alt string) error {
	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	filename, err := media.Save(h.uploadsDir, f)
	if err != nil {
		return err
	}
	_, err = h.catalog.AddImage(r.Context(), productID, "/uploads/"+filename, alt)
	return err
}

func (h *Handlers) ImageDelete(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	imageID, err := strconv.ParseInt(r.PathValue("imageID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.catalog.DeleteImage(r.Context(), imageID); err != nil {
		h.serverError(w, "delete image", err)
		return
	}
	http.Redirect(w, r, editURL(productID), http.StatusSeeOther)
}

func (h *Handlers) ImageMove(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	imageID, err := strconv.ParseInt(r.PathValue("imageID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	direction := -1
	if r.FormValue("direction") == "down" {
		direction = 1
	}
	if err := h.catalog.MoveImage(r.Context(), imageID, direction); err != nil {
		h.serverError(w, "move image", err)
		return
	}
	http.Redirect(w, r, editURL(productID), http.StatusSeeOther)
}

func editURL(productID int64) string {
	return "/admin/products/" + strconv.FormatInt(productID, 10) + "/edit"
}

func (h *Handlers) serverError(w http.ResponseWriter, action string, err error) {
	slog.Error(action, "err", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
