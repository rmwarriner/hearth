-- +goose Up
CREATE TABLE IF NOT EXISTS accounts (
    id             TEXT        NOT NULL PRIMARY KEY,
    household_id   TEXT        NOT NULL REFERENCES households(id),
    name           TEXT        NOT NULL,
    type           TEXT        NOT NULL,
    subtype        TEXT        NOT NULL DEFAULT '',
    currency       TEXT        NOT NULL DEFAULT 'USD',
    parent_id      TEXT        REFERENCES accounts(id),
    is_placeholder BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS accounts_household_id ON accounts(household_id);
CREATE INDEX IF NOT EXISTS accounts_parent_id ON accounts(parent_id);

CREATE TRIGGER accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS accounts_updated_at ON accounts;
DROP INDEX IF EXISTS accounts_parent_id;
DROP INDEX IF EXISTS accounts_household_id;
DROP TABLE IF EXISTS accounts;
