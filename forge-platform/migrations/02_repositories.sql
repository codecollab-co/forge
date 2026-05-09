-- platform.repositories — slice 3.
-- A Repository is owned by exactly one User at MVP (Orgs deferred per ADR-0003).

CREATE TABLE IF NOT EXISTS platform.repositories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES platform.users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    visibility  TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'private')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_id, name)
);
