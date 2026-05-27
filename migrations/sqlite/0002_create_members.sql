-- +goose Up
CREATE TABLE IF NOT EXISTS members (
    id            TEXT NOT NULL PRIMARY KEY,
    household_id  TEXT NOT NULL REFERENCES households(id),
    display_name  TEXT NOT NULL,
    email         TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'member',
    password_hash TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS members_household_id ON members(household_id);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS members_updated_at
AFTER UPDATE ON members
FOR EACH ROW
BEGIN
    UPDATE members SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS members_updated_at;
DROP INDEX IF EXISTS members_household_id;
DROP TABLE IF EXISTS members;
