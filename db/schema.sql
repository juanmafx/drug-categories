-- SQLite schema for the optional durable category store.
-- Apply with: sqlite3 data/categories.db < db/schema.sql

BEGIN;

CREATE TABLE IF NOT EXISTS categories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL COLLATE NOCASE
                CHECK (length(name) BETWEEN 1 AND 120 AND name = trim(name)),
    description TEXT NOT NULL DEFAULT ''
                CHECK (length(description) <= 1000),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT categories_name_unique UNIQUE (name)
);

INSERT OR IGNORE INTO categories (name, description)
VALUES ('Analgesics', 'Pain relief drugs');

COMMIT;
