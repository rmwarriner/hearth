-- +goose Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    token_hash   TEXT        NOT NULL PRIMARY KEY,
    family_id    TEXT        NOT NULL,
    member_id    TEXT        NOT NULL REFERENCES members(id),
    household_id TEXT        NOT NULL REFERENCES households(id),
    issued_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS refresh_tokens_family_id ON refresh_tokens(family_id);
CREATE INDEX IF NOT EXISTS refresh_tokens_member_id ON refresh_tokens(member_id);
CREATE INDEX IF NOT EXISTS refresh_tokens_expires_at ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS refresh_tokens_expires_at;
DROP INDEX IF EXISTS refresh_tokens_member_id;
DROP INDEX IF EXISTS refresh_tokens_family_id;
DROP TABLE IF EXISTS refresh_tokens;
