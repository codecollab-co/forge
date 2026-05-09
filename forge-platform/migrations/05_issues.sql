-- platform.issues + issue_comments — slice 6.
-- Issue numbers are per-repository, allocated by atomically incrementing
-- repositories.next_issue_number.
--
-- Assignee at this slice is a single nullable user FK. Slice 7 will add an
-- assignee_agent column + a CHECK constraint enforcing that at most one
-- assignee column is set.

ALTER TABLE platform.repositories
    ADD COLUMN IF NOT EXISTS next_issue_number INT NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS platform.issues (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id           UUID NOT NULL REFERENCES platform.repositories(id) ON DELETE CASCADE,
    number            INT NOT NULL,
    author_id         UUID REFERENCES platform.users(id),
    title             TEXT NOT NULL,
    body              TEXT,
    state             TEXT NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'closed')),
    assignee_user_id  UUID REFERENCES platform.users(id),
    closed_at         TIMESTAMPTZ,
    closed_by         UUID REFERENCES platform.users(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (repo_id, number)
);

CREATE INDEX IF NOT EXISTS issues_repo_state_idx ON platform.issues (repo_id, state);

CREATE TABLE IF NOT EXISTS platform.issue_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id    UUID NOT NULL REFERENCES platform.issues(id) ON DELETE CASCADE,
    author_id   UUID REFERENCES platform.users(id),
    body        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS issue_comments_issue_id_idx ON platform.issue_comments (issue_id);
