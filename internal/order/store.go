package order

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// dbtx is satisfied by both *sql.DB and *sql.Tx. Write methods that must
// share checkout's transaction (Create, MarkPaid) take it explicitly;
// read-only methods use the store's own *sql.DB.
type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type CreateParams struct {
	OrderNumber   string
	Email         string
	SubtotalCents int64
	ShippingCents int64
	DiscountCents int64
	DiscountCode  string
	TotalCents    int64
	Shipping      Address
	Items         []Item
}

// Create inserts the order and its line items via tx — part of checkout's
// single transaction, so order creation and the stock decrement that
// precedes it commit or roll back together.
func (s *Store) Create(ctx context.Context, tx dbtx, p CreateParams) (int64, error) {
	res, err := tx.ExecContext(ctx, `
		INSERT INTO orders (order_number, email, status, subtotal_cents, shipping_cents, discount_cents, discount_code, total_cents,
			shipping_name, shipping_address, shipping_city, shipping_postal_code, shipping_country)
		VALUES (?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.OrderNumber, p.Email, p.SubtotalCents, p.ShippingCents, p.DiscountCents, p.DiscountCode, p.TotalCents,
		p.Shipping.Name, p.Shipping.Line1, p.Shipping.City, p.Shipping.PostalCode, p.Shipping.Country)
	if err != nil {
		return 0, fmt.Errorf("create order: %w", err)
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("create order: %w", err)
	}

	for _, item := range p.Items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO order_items (order_id, variant_id, product_title, variant_name, unit_price_cents, qty)
			VALUES (?, ?, ?, ?, ?, ?)`,
			orderID, item.VariantID, item.ProductTitle, item.VariantName, item.UnitPriceCents, item.Qty); err != nil {
			return 0, fmt.Errorf("create order item: %w", err)
		}
	}
	return orderID, nil
}

func (s *Store) MarkPaid(ctx context.Context, tx dbtx, orderID int64, paymentRef string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE orders SET status = 'paid', payment_reference = ?, updated_at = datetime('now') WHERE id = ?`,
		paymentRef, orderID)
	if err != nil {
		return fmt.Errorf("mark order paid: %w", err)
	}
	return nil
}

// GetByNumberAndEmail is how a guest looks up their own order: the pairing
// of order number (unguessable-ish) and the email they checked out with,
// no account needed. Case-insensitive on email since customers won't
// always retype it identically.
func (s *Store) GetByNumberAndEmail(ctx context.Context, orderNumber, email string) (*Order, error) {
	var o Order
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, order_number, email, status, subtotal_cents, shipping_cents, discount_cents, discount_code, total_cents,
		       shipping_name, shipping_address, shipping_city, shipping_postal_code, shipping_country,
		       payment_reference, tracking_number, created_at
		FROM orders WHERE order_number = ? AND lower(email) = lower(?)`,
		orderNumber, email,
	).Scan(&o.ID, &o.OrderNumber, &o.Email, &o.Status, &o.SubtotalCents, &o.ShippingCents, &o.DiscountCents, &o.DiscountCode, &o.TotalCents,
		&o.Shipping.Name, &o.Shipping.Line1, &o.Shipping.City, &o.Shipping.PostalCode, &o.Shipping.Country,
		&o.PaymentReference, &o.TrackingNumber, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	o.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

	items, err := s.itemsFor(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return &o, nil
}

func (s *Store) itemsFor(ctx context.Context, orderID int64) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT variant_id, product_title, variant_name, unit_price_cents, qty FROM order_items WHERE order_id = ? ORDER BY id`,
		orderID)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.VariantID, &it.ProductTitle, &it.VariantName, &it.UnitPriceCents, &it.Qty); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// NewOrderNumber returns a short, unguessable order reference like
// "BQ-3F9A1C2E0B4D". The orders.order_number UNIQUE constraint is the
// actual collision guard; 48 bits of randomness just makes a collision
// astronomically unlikely at this app's scale.
func NewOrderNumber() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "BQ-" + strings.ToUpper(hex.EncodeToString(b)), nil
}
