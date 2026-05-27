-- +goose Up
CREATE TABLE IF NOT EXISTS envelope_allocations (
    id           TEXT NOT NULL PRIMARY KEY,
    envelope_id  TEXT NOT NULL REFERENCES envelopes(id),
    period_start TEXT NOT NULL,
    amount       TEXT NOT NULL,
    currency     TEXT NOT NULL,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS envelope_allocations_envelope_id  ON envelope_allocations(envelope_id);
CREATE INDEX IF NOT EXISTS envelope_allocations_period_start ON envelope_allocations(period_start);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS envelope_allocations_updated_at
AFTER UPDATE ON envelope_allocations
FOR EACH ROW
BEGIN
    UPDATE envelope_allocations SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS envelope_allocations_updated_at;
DROP INDEX IF EXISTS envelope_allocations_period_start;
DROP INDEX IF EXISTS envelope_allocations_envelope_id;
DROP TABLE IF EXISTS envelope_allocations;
