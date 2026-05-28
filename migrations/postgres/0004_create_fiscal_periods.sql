-- +goose Up
CREATE TABLE IF NOT EXISTS fiscal_periods (
    id           TEXT        NOT NULL PRIMARY KEY,
    household_id TEXT        NOT NULL REFERENCES households(id),
    name         TEXT        NOT NULL,
    start_date   TEXT        NOT NULL,
    end_date     TEXT        NOT NULL,
    locked_at    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS fiscal_periods_household_id ON fiscal_periods(household_id);
CREATE INDEX IF NOT EXISTS fiscal_periods_dates ON fiscal_periods(start_date, end_date);

CREATE TRIGGER fiscal_periods_updated_at
    BEFORE UPDATE ON fiscal_periods
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS fiscal_periods_updated_at ON fiscal_periods;
DROP INDEX IF EXISTS fiscal_periods_dates;
DROP INDEX IF EXISTS fiscal_periods_household_id;
DROP TABLE IF EXISTS fiscal_periods;
