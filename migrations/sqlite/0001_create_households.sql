-- +goose Up
CREATE TABLE IF NOT EXISTS households (
    id               TEXT NOT NULL PRIMARY KEY,
    name             TEXT NOT NULL,
    fiscal_year_start INTEGER NOT NULL DEFAULT 1,
    base_currency    TEXT NOT NULL DEFAULT 'USD',
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS households_updated_at
AFTER UPDATE ON households
FOR EACH ROW
BEGIN
    UPDATE households SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS households_updated_at;
DROP TABLE IF EXISTS households;
