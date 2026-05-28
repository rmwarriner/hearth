-- +goose Up
CREATE TABLE IF NOT EXISTS journal_entries (
    id             TEXT        NOT NULL PRIMARY KEY,
    household_id   TEXT        NOT NULL REFERENCES households(id),
    posted_at      TIMESTAMPTZ NOT NULL,
    description    TEXT        NOT NULL DEFAULT '',
    reference      TEXT        NOT NULL DEFAULT '',
    source         TEXT        NOT NULL DEFAULT 'manual',
    created_by     TEXT        NOT NULL DEFAULT '',
    is_reversal_of TEXT        REFERENCES journal_entries(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS journal_entries_household_id ON journal_entries(household_id);
CREATE INDEX IF NOT EXISTS journal_entries_posted_at ON journal_entries(posted_at);
CREATE INDEX IF NOT EXISTS journal_entries_description ON journal_entries(description);

CREATE TRIGGER journal_entries_updated_at
    BEFORE UPDATE ON journal_entries
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS journal_entries_updated_at ON journal_entries;
DROP INDEX IF EXISTS journal_entries_description;
DROP INDEX IF EXISTS journal_entries_posted_at;
DROP INDEX IF EXISTS journal_entries_household_id;
DROP TABLE IF EXISTS journal_entries;
