-- platform.ssh_keys — slice 11.
-- One row per public key registered to a user. The fingerprint
-- (SHA256 of the marshalled key) is the stable lookup index used by
-- the SSH server during authentication.

CREATE TABLE IF NOT EXISTS platform.ssh_keys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES platform.users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    fingerprint   TEXT NOT NULL UNIQUE,   -- e.g. "SHA256:<base64>"
    public_key    TEXT NOT NULL,           -- the full authorized_keys line
    last_used_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS ssh_keys_user_idx ON platform.ssh_keys (user_id);
