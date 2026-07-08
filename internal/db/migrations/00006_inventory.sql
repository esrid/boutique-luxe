-- +goose Up
CREATE TABLE stock_movements (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    variant_id INTEGER NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    delta      INTEGER NOT NULL,
    reason     TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_stock_movements_variant ON stock_movements(variant_id);

-- +goose Down
DROP TABLE stock_movements;
