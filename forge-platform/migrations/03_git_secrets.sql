-- platform.user_git_secrets — slice 4.
-- Per-user secret used as the HTTP Basic password for `git push` over HTTPS.
-- Stored hashed (bcrypt). The plaintext is shown to the user exactly once.
-- Slice 12 replaces this with named, revocable Personal Access Tokens.

CREATE TABLE IF NOT EXISTS platform.user_git_secrets (
    user_id     UUID PRIMARY KEY REFERENCES platform.users(id) ON DELETE CASCADE,
    secret_hash TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);
