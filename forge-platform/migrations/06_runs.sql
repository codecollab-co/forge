-- agent.runs + agent.run_events — slice 7.
--
-- Cross-schema-write exception: forge-platform may INSERT a Run row in
-- response to a user clicking "Assign to Agent" (so the API call can return
-- a run_id synchronously). All UPDATES to agent.* remain orchestrator-only.

CREATE TABLE IF NOT EXISTS agent.runs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id             UUID NOT NULL REFERENCES platform.repositories(id) ON DELETE CASCADE,
    issue_id            UUID NOT NULL REFERENCES platform.issues(id) ON DELETE CASCADE,
    requested_by        UUID NOT NULL REFERENCES platform.users(id),
    state               TEXT NOT NULL DEFAULT 'queued'
                          CHECK (state IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    cancel_requested    BOOLEAN NOT NULL DEFAULT FALSE,
    sandbox_id          TEXT,
    pr_id               UUID REFERENCES platform.pull_requests(id) ON DELETE SET NULL,
    error_category      TEXT,
    error_message       TEXT,
    last_heartbeat_at   TIMESTAMPTZ,
    started_at          TIMESTAMPTZ,
    finished_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS runs_state_heartbeat_idx
    ON agent.runs (state, last_heartbeat_at)
    WHERE state = 'running';

CREATE INDEX IF NOT EXISTS runs_issue_idx
    ON agent.runs (issue_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent.run_events (
    id          BIGSERIAL PRIMARY KEY,
    run_id      UUID NOT NULL REFERENCES agent.runs(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS run_events_run_idx ON agent.run_events (run_id, id);

-- The PR a Run produced (nullable, set by the orchestrator after the PR opens).
ALTER TABLE platform.pull_requests
    ADD COLUMN IF NOT EXISTS created_by_run_id UUID REFERENCES agent.runs(id) ON DELETE SET NULL;

-- Agent assignee on an Issue. Companion to assignee_user_id introduced in
-- slice 6. CHECK enforces at most one assignee shape is set.
ALTER TABLE platform.issues
    ADD COLUMN IF NOT EXISTS assignee_agent_run_id UUID REFERENCES agent.runs(id) ON DELETE SET NULL;

ALTER TABLE platform.issues
    DROP CONSTRAINT IF EXISTS issues_one_assignee_chk;

ALTER TABLE platform.issues
    ADD CONSTRAINT issues_one_assignee_chk CHECK (
        (assignee_user_id IS NULL) OR (assignee_agent_run_id IS NULL)
    );
