-- +goose Up
CREATE TABLE IF NOT EXISTS members (
    id            TEXT        NOT NULL PRIMARY KEY,
    household_id  TEXT        NOT NULL REFERENCES households(id),
    display_name  TEXT        NOT NULL,
    email         TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'member',
    password_hash TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (household_id, email)
);

CREATE INDEX IF NOT EXISTS members_household_id ON members(household_id);

CREATE TRIGGER members_updated_at
    BEFORE UPDATE ON members
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS members_updated_at ON members;
DROP INDEX IF EXISTS members_household_id;
DROP TABLE IF EXISTS members;
