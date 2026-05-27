-- +goose Up
CREATE TABLE IF NOT EXISTS accounts (
    id             TEXT NOT NULL PRIMARY KEY,
    household_id   TEXT NOT NULL REFERENCES households(id),
    name           TEXT NOT NULL,
    type           TEXT NOT NULL,
    subtype        TEXT NOT NULL DEFAULT '',
    currency       TEXT NOT NULL DEFAULT 'USD',
    parent_id      TEXT REFERENCES accounts(id),
    is_placeholder INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS accounts_household_id ON accounts(household_id);
CREATE INDEX IF NOT EXISTS accounts_parent_id    ON accounts(parent_id);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS accounts_updated_at
AFTER UPDATE ON accounts
FOR EACH ROW
BEGIN
    UPDATE accounts SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS accounts_updated_at;
DROP INDEX IF EXISTS accounts_parent_id;
DROP INDEX IF EXISTS accounts_household_id;
DROP TABLE IF EXISTS accounts;
