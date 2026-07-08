package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// The methods in this file are the admin write path: any status (not just
// published), full CRUD on products/variants/categories/images. The
// storefront-facing read methods live in store.go.

type AdminProductRow struct {
	ID           int64
	Slug         string
	Title        string
	Status       ProductStatus
	CategoryName string
	VariantCount int
}

func (s *Store) ListAllProducts(ctx context.Context) ([]AdminProductRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.slug, p.title, p.status, COALESCE(c.name, ''),
		       (SELECT COUNT(*) FROM product_variants v WHERE v.product_id = p.id)
		FROM products p
		LEFT JOIN categories c ON c.id = p.category_id
		ORDER BY p.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list all products: %w", err)
	}
	defer rows.Close()

	var out []AdminProductRow
	for rows.Next() {
		var row AdminProductRow
		var status string
		if err := rows.Scan(&row.ID, &row.Slug, &row.Title, &status, &row.CategoryName, &row.VariantCount); err != nil {
			return nil, fmt.Errorf("scan product row: %w", err)
		}
		row.Status = ProductStatus(status)
		out = append(out, row)
	}
	return out, rows.Err()
}

// GetProductByID loads a product regardless of status, with its images and
// variants — for the admin edit page.
func (s *Store) GetProductByID(ctx context.Context, id int64) (*Product, error) {
	var p Product
	var categoryID sql.NullInt64
	var status string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, category_id, slug, title, description, status
		FROM products WHERE id = ?`, id,
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

	if p.Images, err = s.imagesFor(ctx, p.ID); err != nil {
		return nil, err
	}
	if p.Variants, err = s.variantsFor(ctx, p.ID); err != nil {
		return nil, err
	}
	return &p, nil
}

type ProductParams struct {
	CategoryID  *int64
	Slug        string
	Title       string
	Description string
	Status      ProductStatus
}

func (s *Store) CreateProduct(ctx context.Context, p ProductParams) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO products (category_id, slug, title, description, status)
		VALUES (?, ?, ?, ?, ?)`,
		p.CategoryID, p.Slug, p.Title, p.Description, string(p.Status))
	if err != nil {
		return 0, fmt.Errorf("create product: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) UpdateProduct(ctx context.Context, id int64, p ProductParams) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE products SET category_id = ?, slug = ?, title = ?, description = ?, status = ?, updated_at = datetime('now')
		WHERE id = ?`,
		p.CategoryID, p.Slug, p.Title, p.Description, string(p.Status), id)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	return nil
}

func (s *Store) DeleteProduct(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	return nil
}

type VariantParams struct {
	SKU               string
	Name              string
	Options           map[string]string
	PriceCents        int64
	StockQty          int
	LowStockThreshold int
}

func (s *Store) CreateVariant(ctx context.Context, productID int64, p VariantParams) (int64, error) {
	optsJSON, err := json.Marshal(p.Options)
	if err != nil {
		return 0, fmt.Errorf("encode variant options: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO product_variants (product_id, sku, name, options_json, price_cents, stock_qty, low_stock_threshold)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		productID, p.SKU, p.Name, string(optsJSON), p.PriceCents, p.StockQty, p.LowStockThreshold)
	if err != nil {
		return 0, fmt.Errorf("create variant: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) UpdateVariant(ctx context.Context, id int64, p VariantParams) error {
	optsJSON, err := json.Marshal(p.Options)
	if err != nil {
		return fmt.Errorf("encode variant options: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE product_variants SET sku = ?, name = ?, options_json = ?, price_cents = ?, stock_qty = ?, low_stock_threshold = ?
		WHERE id = ?`,
		p.SKU, p.Name, string(optsJSON), p.PriceCents, p.StockQty, p.LowStockThreshold, id)
	if err != nil {
		return fmt.Errorf("update variant: %w", err)
	}
	return nil
}

func (s *Store) DeleteVariant(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM product_variants WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete variant: %w", err)
	}
	return nil
}

func (s *Store) AddImage(ctx context.Context, productID int64, url, alt string) (int64, error) {
	var nextPos int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(position), -1) + 1 FROM product_images WHERE product_id = ?`, productID,
	).Scan(&nextPos); err != nil {
		return 0, fmt.Errorf("compute image position: %w", err)
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO product_images (product_id, url, alt, position) VALUES (?, ?, ?, ?)`,
		productID, url, alt, nextPos)
	if err != nil {
		return 0, fmt.Errorf("add image: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) DeleteImage(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM product_images WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete image: %w", err)
	}
	return nil
}

// MoveImage swaps an image's position with its immediate neighbor
// (direction -1 = up/earlier, +1 = down/later), giving no-JS reorder
// controls (up/down buttons) instead of drag-and-drop.
func (s *Store) MoveImage(ctx context.Context, id int64, direction int) error {
	var productID int64
	var position int
	if err := s.db.QueryRowContext(ctx,
		`SELECT product_id, position FROM product_images WHERE id = ?`, id,
	).Scan(&productID, &position); err != nil {
		return fmt.Errorf("find image: %w", err)
	}

	var neighborID int64
	var neighborPos int
	order := "DESC"
	cmp := "<"
	if direction > 0 {
		order = "ASC"
		cmp = ">"
	}
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT id, position FROM product_images
		WHERE product_id = ? AND position %s ?
		ORDER BY position %s LIMIT 1`, cmp, order),
		productID, position,
	).Scan(&neighborID, &neighborPos)
	if err == sql.ErrNoRows {
		return nil // already at the edge — nothing to swap with
	}
	if err != nil {
		return fmt.Errorf("find neighbor image: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// A direct two-way swap would momentarily give two rows the same
	// position, violating UNIQUE(product_id, position) — SQLite checks
	// UNIQUE constraints immediately, not deferred. Route through a scratch
	// position (-1, never a real one) instead.
	if _, err := tx.ExecContext(ctx, `UPDATE product_images SET position = -1 WHERE id = ?`, id); err != nil {
		return fmt.Errorf("swap image position: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE product_images SET position = ? WHERE id = ?`, position, neighborID); err != nil {
		return fmt.Errorf("swap image position: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE product_images SET position = ? WHERE id = ?`, neighborPos, id); err != nil {
		return fmt.Errorf("swap image position: %w", err)
	}
	return tx.Commit()
}

func (s *Store) CreateCategory(ctx context.Context, slug, name string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO categories (slug, name) VALUES (?, ?)`, slug, name)
	if err != nil {
		return 0, fmt.Errorf("create category: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) UpdateCategory(ctx context.Context, id int64, slug, name string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE categories SET slug = ?, name = ? WHERE id = ?`, slug, name, id)
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}
	return nil
}

func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

func (s *Store) GetCategory(ctx context.Context, id int64) (Category, error) {
	var c Category
	err := s.db.QueryRowContext(ctx, `SELECT id, slug, name FROM categories WHERE id = ?`, id).Scan(&c.ID, &c.Slug, &c.Name)
	if err == sql.ErrNoRows {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, fmt.Errorf("get category: %w", err)
	}
	return c, nil
}
