-- +goose Up
CREATE TABLE IF NOT EXISTS envelopes (
    id              TEXT        NOT NULL PRIMARY KEY,
    household_id    TEXT        NOT NULL REFERENCES households(id),
    name            TEXT        NOT NULL,
    target_amount   TEXT        NOT NULL DEFAULT '0',
    target_currency TEXT        NOT NULL DEFAULT 'USD',
    period_type     TEXT        NOT NULL DEFAULT 'monthly',
    rollover_policy TEXT        NOT NULL DEFAULT 'zero',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS envelopes_household_id ON envelopes(household_id);

CREATE TRIGGER envelopes_updated_at
    BEFORE UPDATE ON envelopes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS envelopes_updated_at ON envelopes;
DROP INDEX IF EXISTS envelopes_household_id;
DROP TABLE IF EXISTS envelopes;
