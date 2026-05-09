-- platform.pull_requests + pr_comments — slice 5.
-- PR numbers are per-repository, allocated by atomically incrementing
-- repositories.next_pr_number.

ALTER TABLE platform.repositories
    ADD COLUMN IF NOT EXISTS next_pr_number INT NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS platform.pull_requests (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id          UUID NOT NULL REFERENCES platform.repositories(id) ON DELETE CASCADE,
    number           INT NOT NULL,
    author_id        UUID REFERENCES platform.users(id),
    title            TEXT NOT NULL,
    body             TEXT,
    head_branch      TEXT NOT NULL,
    base_branch      TEXT NOT NULL,
    state            TEXT NOT NULL DEFAULT 'open'
                      CHECK (state IN ('open', 'merged', 'closed')),
    merge_commit_oid TEXT,
    merged_at        TIMESTAMPTZ,
    merged_by        UUID REFERENCES platform.users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (repo_id, number)
);

CREATE INDEX IF NOT EXISTS pull_requests_repo_state_idx
    ON platform.pull_requests (repo_id, state);

CREATE TABLE IF NOT EXISTS platform.pr_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pr_id       UUID NOT NULL REFERENCES platform.pull_requests(id) ON DELETE CASCADE,
    author_id   UUID REFERENCES platform.users(id),
    -- 'user' or 'agent' (Reviewer Agent comments arrive in slice 10)
    author_kind TEXT NOT NULL DEFAULT 'user'
                 CHECK (author_kind IN ('user', 'agent')),
    body        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS pr_comments_pr_id_idx ON platform.pr_comments (pr_id);
