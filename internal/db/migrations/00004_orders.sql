-- +goose Up
CREATE TABLE orders (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    order_number         TEXT NOT NULL UNIQUE,
    email                TEXT NOT NULL,
    status               TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending','paid','fulfilled','shipped','delivered','cancelled')),
    subtotal_cents       INTEGER NOT NULL,
    shipping_cents       INTEGER NOT NULL DEFAULT 0,
    discount_cents       INTEGER NOT NULL DEFAULT 0,
    total_cents          INTEGER NOT NULL,
    shipping_name        TEXT NOT NULL,
    shipping_address     TEXT NOT NULL,
    shipping_city        TEXT NOT NULL,
    shipping_postal_code TEXT NOT NULL,
    shipping_country     TEXT NOT NULL,
    payment_reference    TEXT NOT NULL DEFAULT '',
    tracking_number      TEXT NOT NULL DEFAULT '',
    created_at           TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at           TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_orders_email ON orders(email);

CREATE TABLE order_items (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id           INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    variant_id         INTEGER NOT NULL REFERENCES product_variants(id),
    product_title      TEXT NOT NULL,
    variant_name       TEXT NOT NULL,
    unit_price_cents   INTEGER NOT NULL,
    qty                INTEGER NOT NULL
);
CREATE INDEX idx_order_items_order ON order_items(order_id);

-- +goose Down
DROP TABLE order_items;
DROP TABLE orders;
