package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, slug, name FROM categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListProducts returns published products matching filter, plus the total
// match count (ignoring pagination) for building pager UI.
func (s *Store) ListProducts(ctx context.Context, f ProductFilter) ([]ProductSummary, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 24
	}

	const base = `
		FROM (
			SELECT p.id, p.slug, p.title, c.slug AS category_slug,
			       (SELECT pi.url FROM product_images pi WHERE pi.product_id = p.id ORDER BY pi.position LIMIT 1) AS thumbnail,
			       (SELECT MIN(v.price_cents) FROM product_variants v WHERE v.product_id = p.id) AS min_price_cents,
			       (SELECT COALESCE(SUM(v.stock_qty), 0) FROM product_variants v WHERE v.product_id = p.id) AS total_stock
			FROM products p
			LEFT JOIN categories c ON c.id = p.category_id
			WHERE p.status = 'published'
		) x
		WHERE (:category = '' OR x.category_slug = :category)
		  AND (:min_price = 0 OR x.min_price_cents >= :min_price)
		  AND (:max_price = 0 OR x.min_price_cents <= :max_price)
	`
	args := []any{
		sql.Named("category", f.CategorySlug),
		sql.Named("min_price", f.MinPriceCents),
		sql.Named("max_price", f.MaxPriceCents),
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) `+base, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	orderBy := "x.id DESC"
	switch f.Sort {
	case SortPriceAsc:
		orderBy = "x.min_price_cents ASC"
	case SortPriceDesc:
		orderBy = "x.min_price_cents DESC"
	}

	query := `SELECT x.id, x.slug, x.title, x.category_slug, x.thumbnail, x.min_price_cents, x.total_stock ` +
		base + ` ORDER BY ` + orderBy + ` LIMIT :limit OFFSET :offset`
	args = append(args,
		sql.Named("limit", f.PageSize),
		sql.Named("offset", (f.Page-1)*f.PageSize),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var out []ProductSummary
	for rows.Next() {
		var p ProductSummary
		var categorySlug, thumbnail sql.NullString
		var minPrice sql.NullInt64
		var totalStock int
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &categorySlug, &thumbnail, &minPrice, &totalStock); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		p.CategorySlug = categorySlug.String
		p.Thumbnail = thumbnail.String
		p.MinPriceCents = minPrice.Int64
		p.InStock = totalStock > 0
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// GetProductBySlug loads a published product with its images and variants.
// Returns ErrNotFound if it doesn't exist or isn't published.
func (s *Store) GetProductBySlug(ctx context.Context, slug string) (*Product, error) {
	var p Product
	var categoryID sql.NullInt64
	var status string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, category_id, slug, title, description, status
		FROM products WHERE slug = ? AND status = 'published'`, slug,
	).Scan(&p.ID, &categoryID, &p.Slug, &p.Title, &p.Description, &status)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	if categoryID.Valid {
		id := categoryID.Int64
		p.CategoryID = &id
	}
	p.Status = ProductStatus(status)

	images, err := s.imagesFor(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Images = images

	variants, err := s.variantsFor(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Variants = variants

	return &p, nil
}

func (s *Store) imagesFor(ctx context.Context, productID int64) ([]Image, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, product_id, url, alt, position FROM product_images WHERE product_id = ? ORDER BY position`, productID)
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	defer rows.Close()

	var out []Image
	for rows.Next() {
		var im Image
		if err := rows.Scan(&im.ID, &im.ProductID, &im.URL, &im.Alt, &im.Position); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}
		out = append(out, im)
	}
	return out, rows.Err()
}

func (s *Store) variantsFor(ctx context.Context, productID int64) ([]Variant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, product_id, sku, name, options_json, price_cents, stock_qty, low_stock_threshold
		 FROM product_variants WHERE product_id = ? ORDER BY id`, productID)
	if err != nil {
		return nil, fmt.Errorf("list variants: %w", err)
	}
	defer rows.Close()

	var out []Variant
	for rows.Next() {
		var v Variant
		var optionsJSON string
		if err := rows.Scan(&v.ID, &v.ProductID, &v.SKU, &v.Name, &optionsJSON, &v.PriceCents, &v.StockQty, &v.LowStockThreshold); err != nil {
			return nil, fmt.Errorf("scan variant: %w", err)
		}
		if err := json.Unmarshal([]byte(optionsJSON), &v.Options); err != nil {
			return nil, fmt.Errorf("decode variant options: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
