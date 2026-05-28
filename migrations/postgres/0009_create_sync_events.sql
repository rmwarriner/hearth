-- +goose Up
CREATE TABLE IF NOT EXISTS sync_events (
    id           TEXT        NOT NULL PRIMARY KEY,
    device_id    TEXT        NOT NULL,
    sequence     BIGINT      NOT NULL,
    payload_type TEXT        NOT NULL,
    payload      TEXT        NOT NULL,
    synced_at    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS sync_events_device_sequence ON sync_events(device_id, sequence);
CREATE INDEX IF NOT EXISTS sync_events_synced_at ON sync_events(synced_at) WHERE synced_at IS NULL;

CREATE TRIGGER sync_events_updated_at
    BEFORE UPDATE ON sync_events
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS sync_events_updated_at ON sync_events;
DROP INDEX IF EXISTS sync_events_synced_at;
DROP INDEX IF EXISTS sync_events_device_sequence;
DROP TABLE IF EXISTS sync_events;
