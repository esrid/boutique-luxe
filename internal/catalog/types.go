// Package catalog owns products, variants, categories, and product images —
// the read/write surface both the storefront and the admin catalog pages
// are built on.
package catalog

import "errors"

var ErrNotFound = errors.New("catalog: not found")

type Category struct {
	ID   int64
	Slug string
	Name string
}

type ProductStatus string

const (
	StatusDraft     ProductStatus = "draft"
	StatusPublished ProductStatus = "published"
)

// ProductSummary is what a listing page needs: no full image/variant list.
type ProductSummary struct {
	ID            int64
	Slug          string
	Title         string
	CategorySlug  string
	Thumbnail     string
	MinPriceCents int64
	InStock       bool
}

type Product struct {
	ID          int64
	CategoryID  *int64
	Slug        string
	Title       string
	Description string
	Status      ProductStatus
	Images      []Image
	Variants    []Variant
}

type Image struct {
	ID        int64
	ProductID int64
	URL       string
	Alt       string
	Position  int
}

type Variant struct {
	ID                int64
	ProductID         int64
	SKU               string
	Name              string
	Options           map[string]string
	PriceCents        int64
	StockQty          int
	LowStockThreshold int
}

func (v Variant) LowStock() bool {
	return v.StockQty <= v.LowStockThreshold
}

type ProductFilter struct {
	CategorySlug  string
	MinPriceCents int64 // 0 = no floor
	MaxPriceCents int64 // 0 = no ceiling
	Sort          Sort
	Page          int // 1-based
	PageSize      int
}

type Sort string

const (
	SortNewest    Sort = "newest"
	SortPriceAsc  Sort = "price_asc"
	SortPriceDesc Sort = "price_desc"
)
