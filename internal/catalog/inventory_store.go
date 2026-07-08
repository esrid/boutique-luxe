package catalog

import (
	"context"
	"database/sql"
	"fmt"
)

type InventoryRow struct {
	VariantID         int64
	SKU               string
	VariantName       string
	ProductID         int64
	ProductTitle      string
	StockQty          int
	LowStockThreshold int
}

func (r InventoryRow) LowStock() bool { return r.StockQty <= r.LowStockThreshold }

func (s *Store) ListInventory(ctx context.Context) ([]InventoryRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT v.id, v.sku, v.name, p.id, p.title, v.stock_qty, v.low_stock_threshold
		FROM product_variants v
		JOIN products p ON p.id = v.product_id
		ORDER BY p.title, v.name`)
	if err != nil {
		return nil, fmt.Errorf("list inventory: %w", err)
	}
	defer rows.Close()
	return scanInventoryRows(rows)
}

// LowStockVariants returns variants at or below their low-stock threshold,
// for the dashboard alert list.
func (s *Store) LowStockVariants(ctx context.Context, limit int) ([]InventoryRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT v.id, v.sku, v.name, p.id, p.title, v.stock_qty, v.low_stock_threshold
		FROM product_variants v
		JOIN products p ON p.id = v.product_id
		WHERE v.stock_qty <= v.low_stock_threshold
		ORDER BY v.stock_qty ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("low stock variants: %w", err)
	}
	defer rows.Close()
	return scanInventoryRows(rows)
}

func scanInventoryRows(rows *sql.Rows) ([]InventoryRow, error) {
	var out []InventoryRow
	for rows.Next() {
		var r InventoryRow
		if err := rows.Scan(&r.VariantID, &r.SKU, &r.VariantName, &r.ProductID, &r.ProductTitle, &r.StockQty, &r.LowStockThreshold); err != nil {
			return nil, fmt.Errorf("scan inventory row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

var ErrWouldGoNegative = fmt.Errorf("catalog: adjustment would take stock below zero")

// AdjustStock applies delta (positive or negative) to a variant's stock and
// logs the change to stock_movements — the audit trail for manual
// corrections (damaged goods, recounts, etc; sales go through checkout's
// own decrement, not this path).
func (s *Store) AdjustStock(ctx context.Context, variantID int64, delta int, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var current int
	if err := tx.QueryRowContext(ctx, `SELECT stock_qty FROM product_variants WHERE id = ?`, variantID).Scan(&current); err != nil {
		return fmt.Errorf("get current stock: %w", err)
	}
	if current+delta < 0 {
		return ErrWouldGoNegative
	}

	if _, err := tx.ExecContext(ctx, `UPDATE product_variants SET stock_qty = stock_qty + ? WHERE id = ?`, delta, variantID); err != nil {
		return fmt.Errorf("adjust stock: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO stock_movements (variant_id, delta, reason) VALUES (?, ?, ?)`, variantID, delta, reason); err != nil {
		return fmt.Errorf("log stock movement: %w", err)
	}
	return tx.Commit()
}
