-- platform.users — populated by the SuperTokens OnSignInUp hook (slice 2).
-- One row per (provider, external_id) pair.

CREATE TABLE IF NOT EXISTS platform.users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supertokens_id  TEXT NOT NULL UNIQUE,
    provider        TEXT NOT NULL,
    external_id     TEXT NOT NULL,
    handle          TEXT NOT NULL UNIQUE,
    email           TEXT,
    display_name    TEXT,
    avatar_url      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, external_id)
);

CREATE INDEX IF NOT EXISTS users_email_idx ON platform.users (email);
