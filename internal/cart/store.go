package cart

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, tokenHash string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO carts (token_hash) VALUES (?)`, tokenHash)
	if err != nil {
		return 0, fmt.Errorf("create cart: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) FindByTokenHash(ctx context.Context, tokenHash string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM carts WHERE token_hash = ?`, tokenHash).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) Count(ctx context.Context, cartID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(qty), 0) FROM cart_items WHERE cart_id = ?`, cartID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count cart items: %w", err)
	}
	return n, nil
}

// AddItem adds qty of a variant to the cart, capped at available stock. If
// the variant is already in the cart, the line quantity is increased (also
// capped) rather than duplicated.
func (s *Store) AddItem(ctx context.Context, cartID, variantID int64, qty int) error {
	if qty < 1 {
		return fmt.Errorf("add item: qty must be positive, got %d", qty)
	}

	var stock int
	if err := s.db.QueryRowContext(ctx, `SELECT stock_qty FROM product_variants WHERE id = ?`, variantID).Scan(&stock); err != nil {
		return fmt.Errorf("look up variant stock: %w", err)
	}

	var existingQty int
	err := s.db.QueryRowContext(ctx, `SELECT qty FROM cart_items WHERE cart_id = ? AND variant_id = ?`, cartID, variantID).Scan(&existingQty)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("look up cart item: %w", err)
	}

	newQty := min(existingQty+qty, stock)
	if newQty < 1 {
		return nil // out of stock — nothing to add
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cart_items (cart_id, variant_id, qty) VALUES (?, ?, ?)
		ON CONFLICT(cart_id, variant_id) DO UPDATE SET qty = excluded.qty`,
		cartID, variantID, newQty)
	if err != nil {
		return fmt.Errorf("upsert cart item: %w", err)
	}
	return nil
}

// UpdateItemQty sets a line's quantity, capped at available stock. qty <= 0
// removes the line.
func (s *Store) UpdateItemQty(ctx context.Context, cartID, variantID int64, qty int) error {
	if qty < 1 {
		return s.RemoveItem(ctx, cartID, variantID)
	}
	var stock int
	if err := s.db.QueryRowContext(ctx, `SELECT stock_qty FROM product_variants WHERE id = ?`, variantID).Scan(&stock); err != nil {
		return fmt.Errorf("look up variant stock: %w", err)
	}
	qty = min(qty, stock)
	_, err := s.db.ExecContext(ctx, `UPDATE cart_items SET qty = ? WHERE cart_id = ? AND variant_id = ?`, qty, cartID, variantID)
	if err != nil {
		return fmt.Errorf("update cart item: %w", err)
	}
	return nil
}

func (s *Store) RemoveItem(ctx context.Context, cartID, variantID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cart_items WHERE cart_id = ? AND variant_id = ?`, cartID, variantID)
	if err != nil {
		return fmt.Errorf("remove cart item: %w", err)
	}
	return nil
}

func (s *Store) Load(ctx context.Context, cartID int64) (*Cart, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT v.id, p.slug, p.title, v.name, v.price_cents, ci.qty, v.stock_qty
		FROM cart_items ci
		JOIN product_variants v ON v.id = ci.variant_id
		JOIN products p ON p.id = v.product_id
		WHERE ci.cart_id = ?
		ORDER BY ci.id`, cartID)
	if err != nil {
		return nil, fmt.Errorf("load cart: %w", err)
	}
	defer rows.Close()

	c := &Cart{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.VariantID, &it.ProductSlug, &it.ProductTitle, &it.VariantName, &it.UnitPriceCents, &it.Qty, &it.StockQty); err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		it.LineTotalCents = it.UnitPriceCents * int64(it.Qty)
		c.SubtotalCents += it.LineTotalCents
		c.Items = append(c.Items, it)
	}
	return c, rows.Err()
}
