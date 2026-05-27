-- +goose Up
CREATE TABLE IF NOT EXISTS sync_events (
    id           TEXT NOT NULL PRIMARY KEY,
    device_id    TEXT NOT NULL,
    sequence     INTEGER NOT NULL,
    payload_type TEXT NOT NULL,
    payload      TEXT NOT NULL,
    synced_at    TEXT,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS sync_events_device_sequence ON sync_events(device_id, sequence);
CREATE INDEX IF NOT EXISTS sync_events_synced_at       ON sync_events(synced_at);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS sync_events_updated_at
AFTER UPDATE ON sync_events
FOR EACH ROW
BEGIN
    UPDATE sync_events SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS sync_events_updated_at;
DROP INDEX IF EXISTS sync_events_synced_at;
DROP INDEX IF EXISTS sync_events_device_sequence;
DROP TABLE IF EXISTS sync_events;
