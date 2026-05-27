-- +goose Up
CREATE TABLE IF NOT EXISTS fiscal_periods (
    id           TEXT NOT NULL PRIMARY KEY,
    household_id TEXT NOT NULL REFERENCES households(id),
    name         TEXT NOT NULL,
    start_date   TEXT NOT NULL,
    end_date     TEXT NOT NULL,
    locked_at    TEXT,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS fiscal_periods_household_id ON fiscal_periods(household_id);
CREATE INDEX IF NOT EXISTS fiscal_periods_dates        ON fiscal_periods(start_date, end_date);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS fiscal_periods_updated_at
AFTER UPDATE ON fiscal_periods
FOR EACH ROW
BEGIN
    UPDATE fiscal_periods SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS fiscal_periods_updated_at;
DROP INDEX IF EXISTS fiscal_periods_dates;
DROP INDEX IF EXISTS fiscal_periods_household_id;
DROP TABLE IF EXISTS fiscal_periods;
