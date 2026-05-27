-- +goose Up
CREATE TABLE IF NOT EXISTS journal_entries (
    id              TEXT NOT NULL PRIMARY KEY,
    household_id    TEXT NOT NULL REFERENCES households(id),
    posted_at       TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    reference       TEXT NOT NULL DEFAULT '',
    source          TEXT NOT NULL DEFAULT 'manual',
    created_by      TEXT NOT NULL DEFAULT '',
    is_reversal_of  TEXT REFERENCES journal_entries(id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS journal_entries_household_id ON journal_entries(household_id);
CREATE INDEX IF NOT EXISTS journal_entries_posted_at    ON journal_entries(posted_at);
CREATE INDEX IF NOT EXISTS journal_entries_description  ON journal_entries(description);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS journal_entries_updated_at
AFTER UPDATE ON journal_entries
FOR EACH ROW
BEGIN
    UPDATE journal_entries SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS journal_entries_updated_at;
DROP INDEX IF EXISTS journal_entries_description;
DROP INDEX IF EXISTS journal_entries_posted_at;
DROP INDEX IF EXISTS journal_entries_household_id;
DROP TABLE IF EXISTS journal_entries;
