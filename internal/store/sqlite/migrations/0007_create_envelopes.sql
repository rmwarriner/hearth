-- +goose Up
CREATE TABLE IF NOT EXISTS envelopes (
    id              TEXT NOT NULL PRIMARY KEY,
    household_id    TEXT NOT NULL REFERENCES households(id),
    name            TEXT NOT NULL,
    target_amount   TEXT NOT NULL DEFAULT '0',
    target_currency TEXT NOT NULL DEFAULT 'USD',
    period_type     TEXT NOT NULL DEFAULT 'monthly',
    rollover_policy TEXT NOT NULL DEFAULT 'zero',
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS envelopes_household_id ON envelopes(household_id);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS envelopes_updated_at
AFTER UPDATE ON envelopes
FOR EACH ROW
BEGIN
    UPDATE envelopes SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS envelopes_updated_at;
DROP INDEX IF EXISTS envelopes_household_id;
DROP TABLE IF EXISTS envelopes;
