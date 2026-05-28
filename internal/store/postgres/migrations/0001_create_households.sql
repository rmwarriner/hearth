-- +goose Up
CREATE TABLE IF NOT EXISTS households (
    id               TEXT        NOT NULL PRIMARY KEY,
    name             TEXT        NOT NULL,
    fiscal_year_start INTEGER    NOT NULL DEFAULT 1,
    base_currency    TEXT        NOT NULL DEFAULT 'USD',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER households_updated_at
    BEFORE UPDATE ON households
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS households_updated_at ON households;
DROP TABLE IF EXISTS households;
DROP FUNCTION IF EXISTS set_updated_at();
