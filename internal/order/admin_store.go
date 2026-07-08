package order

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// GetByID loads an order regardless of who it belongs to — for the admin
// detail page (no email pairing check, unlike GetByNumberAndEmail).
func (s *Store) GetByID(ctx context.Context, id int64) (*Order, error) {
	var o Order
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, order_number, email, status, subtotal_cents, shipping_cents, discount_cents, discount_code, total_cents,
		       shipping_name, shipping_address, shipping_city, shipping_postal_code, shipping_country,
		       payment_reference, tracking_number, created_at
		FROM orders WHERE id = ?`, id,
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

// ListOrders returns orders newest-first, optionally filtered by status
// ("" means all).
func (s *Store) ListOrders(ctx context.Context, status Status) ([]OrderSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, order_number, email, status, total_cents, created_at
		FROM orders
		WHERE (:status = '' OR status = :status)
		ORDER BY id DESC`,
		sql.Named("status", string(status)))
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()
	return scanOrderSummaries(rows)
}

func (s *Store) RecentOrders(ctx context.Context, limit int) ([]OrderSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, order_number, email, status, total_cents, created_at FROM orders ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("recent orders: %w", err)
	}
	defer rows.Close()
	return scanOrderSummaries(rows)
}

func scanOrderSummaries(rows *sql.Rows) ([]OrderSummary, error) {
	var out []OrderSummary
	for rows.Next() {
		var o OrderSummary
		var createdAt string
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.Email, &o.Status, &o.TotalCents, &createdAt); err != nil {
			return nil, fmt.Errorf("scan order summary: %w", err)
		}
		o.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpdateStatus applies a status transition, rejecting ones not allowed by
// the fulfillment pipeline (see CanTransition).
func (s *Store) UpdateStatus(ctx context.Context, id int64, newStatus Status) error {
	var current Status
	if err := s.db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ?`, id).Scan(&current); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("get order status: %w", err)
	}
	if !CanTransition(current, newStatus) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current, newStatus)
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status = ?, updated_at = datetime('now') WHERE id = ?`, string(newStatus), id)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return nil
}

func (s *Store) UpdateTracking(ctx context.Context, id int64, tracking string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE orders SET tracking_number = ?, updated_at = datetime('now') WHERE id = ?`, tracking, id)
	if err != nil {
		return fmt.Errorf("update tracking: %w", err)
	}
	return nil
}

// RevenueSince sums paid-or-later orders (excludes pending/cancelled)
// created at or after since.
func (s *Store) RevenueSince(ctx context.Context, since time.Time) (int64, error) {
	var total sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT SUM(total_cents) FROM orders
		WHERE status NOT IN ('pending', 'cancelled') AND created_at >= ?`,
		since.UTC().Format("2006-01-02 15:04:05")).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("revenue since: %w", err)
	}
	return total.Int64, nil
}

func (s *Store) CountOrders(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count orders: %w", err)
	}
	return n, nil
}
