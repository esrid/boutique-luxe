-- +goose Up
CREATE TABLE discount_codes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    code       TEXT NOT NULL UNIQUE,
    type       TEXT NOT NULL CHECK (type IN ('percent', 'fixed')),
    value      INTEGER NOT NULL CHECK (value > 0), -- percent: 1-100, fixed: cents
    active     INTEGER NOT NULL DEFAULT 1,
    expires_at TEXT, -- nullable = never expires
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

ALTER TABLE orders ADD COLUMN discount_code TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE orders DROP COLUMN discount_code;
DROP TABLE discount_codes;
