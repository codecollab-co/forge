// Package runs is the Go-side accessor for the agent.runs table.
//
// Per ADR-0007 the agent.* schema is owned by forge-agent (Python). The
// only exception (documented in migration 06) is that platform-api may
// INSERT a Run row in response to an "Assign Agent" click, so the API call
// can return a run_id synchronously. All UPDATES remain orchestrator-only.
package runs

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("run not found")

type State string

const (
	StateQueued    State = "queued"
	StateRunning   State = "running"
	StateSucceeded State = "succeeded"
	StateFailed    State = "failed"
	StateCancelled State = "cancelled"
)

type Run struct {
	ID               string
	RepoID           string
	IssueID          string
	RequestedBy      string
	State            State
	CancelRequested  bool
	SandboxID        *string
	PRID             *string
	ErrorCategory    *string
	ErrorMessage     *string
	LastHeartbeatAt  *time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Event struct {
	ID        int64
	RunID     string
	Type      string
	Payload   []byte // raw JSON
	CreatedAt time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Create inserts a new Run in the queued state. Cross-schema-write
// exception per migration 06.
func (s *Store) Create(ctx context.Context, repoID, issueID, requestedBy string) (*Run, error) {
	r := &Run{}
	err := s.pool.QueryRow(ctx, `
        INSERT INTO agent.runs (repo_id, issue_id, requested_by, state)
        VALUES ($1, $2, $3, 'queued')
        RETURNING id, repo_id, issue_id, requested_by, state, cancel_requested,
                  sandbox_id, pr_id, error_category, error_message,
                  last_heartbeat_at, started_at, finished_at, created_at, updated_at
    `, repoID, issueID, requestedBy).Scan(
		&r.ID, &r.RepoID, &r.IssueID, &r.RequestedBy, &r.State, &r.CancelRequested,
		&r.SandboxID, &r.PRID, &r.ErrorCategory, &r.ErrorMessage,
		&r.LastHeartbeatAt, &r.StartedAt, &r.FinishedAt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Store) Get(ctx context.Context, id string) (*Run, error) {
	r := &Run{}
	err := s.pool.QueryRow(ctx, `
        SELECT id, repo_id, issue_id, requested_by, state, cancel_requested,
               sandbox_id, pr_id, error_category, error_message,
               last_heartbeat_at, started_at, finished_at, created_at, updated_at
          FROM agent.runs WHERE id = $1
    `, id).Scan(
		&r.ID, &r.RepoID, &r.IssueID, &r.RequestedBy, &r.State, &r.CancelRequested,
		&r.SandboxID, &r.PRID, &r.ErrorCategory, &r.ErrorMessage,
		&r.LastHeartbeatAt, &r.StartedAt, &r.FinishedAt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return r, nil
}

// LatestForIssue returns the most recently created Run on an Issue, or nil.
func (s *Store) LatestForIssue(ctx context.Context, issueID string) (*Run, error) {
	r := &Run{}
	err := s.pool.QueryRow(ctx, `
        SELECT id, repo_id, issue_id, requested_by, state, cancel_requested,
               sandbox_id, pr_id, error_category, error_message,
               last_heartbeat_at, started_at, finished_at, created_at, updated_at
          FROM agent.runs WHERE issue_id = $1
         ORDER BY created_at DESC LIMIT 1
    `, issueID).Scan(
		&r.ID, &r.RepoID, &r.IssueID, &r.RequestedBy, &r.State, &r.CancelRequested,
		&r.SandboxID, &r.PRID, &r.ErrorCategory, &r.ErrorMessage,
		&r.LastHeartbeatAt, &r.StartedAt, &r.FinishedAt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}

func (s *Store) RequestCancel(ctx context.Context, id string) error {
	cmd, err := s.pool.Exec(ctx, `
        UPDATE agent.runs SET cancel_requested = TRUE, updated_at = NOW()
         WHERE id = $1 AND state IN ('queued','running')
    `, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListForUser returns the user's most recent Runs across all their repos.
type RunListItem struct {
	ID          string
	State       string
	IssueNumber int
	IssueTitle  string
	RepoOwner   string
	RepoName    string
	PRNumber    *int
	CreatedAt   time.Time
}

func (s *Store) ListForUser(ctx context.Context, userID string, limit int) ([]*RunListItem, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT r.id, r.state, i.number, i.title,
               u.handle, repo.name,
               pr.number, r.created_at
          FROM agent.runs r
          JOIN platform.repositories repo ON repo.id = r.repo_id
          JOIN platform.users u ON u.id = repo.owner_id
          JOIN platform.issues i ON i.id = r.issue_id
          LEFT JOIN platform.pull_requests pr ON pr.id = r.pr_id
         WHERE r.requested_by = $1
         ORDER BY r.created_at DESC
         LIMIT $2
    `, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RunListItem
	for rows.Next() {
		it := &RunListItem{}
		if err := rows.Scan(&it.ID, &it.State, &it.IssueNumber, &it.IssueTitle,
			&it.RepoOwner, &it.RepoName, &it.PRNumber, &it.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// FailStuck marks Runs as failed if they've been running with no heartbeat
// for more than `staleAfter`. Returns the number of Runs affected. Called
// by the janitor goroutine.
func (s *Store) FailStuck(ctx context.Context, staleAfter time.Duration) (int64, error) {
	cmd, err := s.pool.Exec(ctx, `
        UPDATE agent.runs
           SET state = 'failed',
               error_category = COALESCE(error_category, 'orchestrator-stalled'),
               error_message = COALESCE(error_message, 'no heartbeat'),
               finished_at = COALESCE(finished_at, NOW()),
               updated_at = NOW()
         WHERE state = 'running'
           AND (last_heartbeat_at IS NULL OR last_heartbeat_at < NOW() - $1::interval)
    `, staleAfter.String())
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}

// RecentEvents returns the last N events on a Run. Only used for the basic
// /runs/:id GET response; full live streaming arrives in slice 9.
func (s *Store) RecentEvents(ctx context.Context, runID string, limit int) ([]*Event, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT id, run_id, type, payload, created_at
          FROM agent.run_events
         WHERE run_id = $1
         ORDER BY id DESC LIMIT $2
    `, runID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Event
	for rows.Next() {
		e := &Event{}
		if err := rows.Scan(&e.ID, &e.RunID, &e.Type, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
