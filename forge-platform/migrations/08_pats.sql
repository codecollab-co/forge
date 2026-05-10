-- platform.personal_access_tokens — slice 12.
-- Replaces the slice-4 user_git_secrets table. Each token has a name and
-- (later) a list of scopes; secret is bcrypt-hashed.

CREATE TABLE IF NOT EXISTS platform.personal_access_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES platform.users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    secret_hash   TEXT NOT NULL,
    scopes        TEXT[] NOT NULL DEFAULT ARRAY['git:read','git:write']::TEXT[],
    expires_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS pats_user_idx ON platform.personal_access_tokens (user_id);

-- Drop the slice-4 hack. Anyone with a stored git-secret will need to mint
-- a PAT via /me/tokens — there's a single beta user (test@forge.local) so
-- the migration cost is zero.
DROP TABLE IF EXISTS platform.user_git_secrets;
