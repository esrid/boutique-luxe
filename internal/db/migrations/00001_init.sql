-- +goose Up
-- Bootstrap migration: proves the goose wiring works. Domain tables land in
-- later migrations (slice 2+).
CREATE TABLE IF NOT EXISTS schema_bootstrap (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO schema_bootstrap (id) VALUES (1);

-- +goose Down
DROP TABLE schema_bootstrap;
