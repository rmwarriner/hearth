-- +goose Up
CREATE TABLE IF NOT EXISTS postings (
    id               TEXT        NOT NULL PRIMARY KEY,
    journal_entry_id TEXT        NOT NULL REFERENCES journal_entries(id),
    account_id       TEXT        NOT NULL REFERENCES accounts(id),
    amount           TEXT        NOT NULL,
    currency         TEXT        NOT NULL,
    memo             TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS postings_journal_entry_id ON postings(journal_entry_id);
CREATE INDEX IF NOT EXISTS postings_account_id ON postings(account_id);
CREATE INDEX IF NOT EXISTS postings_account_currency ON postings(account_id, currency);

CREATE TRIGGER postings_updated_at
    BEFORE UPDATE ON postings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS postings_updated_at ON postings;
DROP INDEX IF EXISTS postings_account_currency;
DROP INDEX IF EXISTS postings_account_id;
DROP INDEX IF EXISTS postings_journal_entry_id;
DROP TABLE IF EXISTS postings;
