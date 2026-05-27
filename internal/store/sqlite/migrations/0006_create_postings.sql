-- +goose Up
CREATE TABLE IF NOT EXISTS postings (
    id               TEXT NOT NULL PRIMARY KEY,
    journal_entry_id TEXT NOT NULL REFERENCES journal_entries(id),
    account_id       TEXT NOT NULL REFERENCES accounts(id),
    amount           TEXT NOT NULL,
    currency         TEXT NOT NULL,
    memo             TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS postings_journal_entry_id ON postings(journal_entry_id);
CREATE INDEX IF NOT EXISTS postings_account_id       ON postings(account_id);
CREATE INDEX IF NOT EXISTS postings_account_currency ON postings(account_id, currency);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS postings_updated_at
AFTER UPDATE ON postings
FOR EACH ROW
BEGIN
    UPDATE postings SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS postings_updated_at;
DROP INDEX IF EXISTS postings_account_currency;
DROP INDEX IF EXISTS postings_account_id;
DROP INDEX IF EXISTS postings_journal_entry_id;
DROP TABLE IF EXISTS postings;
