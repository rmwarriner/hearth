-- +goose Up
CREATE TABLE IF NOT EXISTS envelope_allocations (
    id           TEXT        NOT NULL PRIMARY KEY,
    envelope_id  TEXT        NOT NULL REFERENCES envelopes(id),
    period_start TEXT        NOT NULL,
    amount       TEXT        NOT NULL,
    currency     TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS envelope_allocations_envelope_id ON envelope_allocations(envelope_id);
CREATE INDEX IF NOT EXISTS envelope_allocations_period_start ON envelope_allocations(period_start);

CREATE TRIGGER envelope_allocations_updated_at
    BEFORE UPDATE ON envelope_allocations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS envelope_allocations_updated_at ON envelope_allocations;
DROP INDEX IF EXISTS envelope_allocations_period_start;
DROP INDEX IF EXISTS envelope_allocations_envelope_id;
DROP TABLE IF EXISTS envelope_allocations;
