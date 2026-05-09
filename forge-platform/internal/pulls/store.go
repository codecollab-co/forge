// Package pulls persists Pull Requests and their comments.
package pulls

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("pull request not found")
	ErrNotOpen   = errors.New("pull request is not open")
	ErrConflict  = errors.New("pull request conflicts with the base")
)

type State string

const (
	StateOpen   State = "open"
	StateMerged State = "merged"
	StateClosed State = "closed"
)

type PullRequest struct {
	ID             string
	RepoID         string
	Number         int
	AuthorID       *string
	AuthorHandle   *string
	Title          string
	Body           string
	HeadBranch     string
	BaseBranch     string
	State          State
	MergeCommitOID *string
	MergedAt       *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Comment struct {
	ID           string
	PRID         string
	AuthorID     *string
	AuthorHandle *string
	AuthorKind   string
	Body         string
	CreatedAt    time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type CreateInput struct {
	RepoID     string
	AuthorID   string
	Title      string
	Body       string
	HeadBranch string
	BaseBranch string
}

// Create allocates the next per-repo PR number atomically and inserts the row.
func (s *Store) Create(ctx context.Context, in CreateInput) (*PullRequest, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var number int
	if err := tx.QueryRow(ctx, `
        UPDATE platform.repositories
           SET next_pr_number = next_pr_number + 1, updated_at = NOW()
         WHERE id = $1
     RETURNING next_pr_number - 1
    `, in.RepoID).Scan(&number); err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := tx.QueryRow(ctx, `
        INSERT INTO platform.pull_requests
            (repo_id, number, author_id, title, body, head_branch, base_branch)
        VALUES ($1, $2, $3, $4, NULLIF($5,''), $6, $7)
        RETURNING id, repo_id, number, author_id, title, COALESCE(body,''),
                  head_branch, base_branch, state, merge_commit_oid, merged_at,
                  created_at, updated_at
    `, in.RepoID, number, in.AuthorID, in.Title, in.Body, in.HeadBranch, in.BaseBranch).
		Scan(&pr.ID, &pr.RepoID, &pr.Number, &pr.AuthorID, &pr.Title, &pr.Body,
			&pr.HeadBranch, &pr.BaseBranch, &pr.State, &pr.MergeCommitOID, &pr.MergedAt,
			&pr.CreatedAt, &pr.UpdatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &pr, nil
}

func (s *Store) GetByRepoAndNumber(ctx context.Context, repoID string, number int) (*PullRequest, error) {
	var pr PullRequest
	err := s.pool.QueryRow(ctx, `
        SELECT pr.id, pr.repo_id, pr.number, pr.author_id, pr.title, COALESCE(pr.body,''),
               pr.head_branch, pr.base_branch, pr.state, pr.merge_commit_oid, pr.merged_at,
               pr.created_at, pr.updated_at, u.handle
          FROM platform.pull_requests pr
          LEFT JOIN platform.users u ON u.id = pr.author_id
         WHERE pr.repo_id = $1 AND pr.number = $2
    `, repoID, number).Scan(
		&pr.ID, &pr.RepoID, &pr.Number, &pr.AuthorID, &pr.Title, &pr.Body,
		&pr.HeadBranch, &pr.BaseBranch, &pr.State, &pr.MergeCommitOID, &pr.MergedAt,
		&pr.CreatedAt, &pr.UpdatedAt, &pr.AuthorHandle,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &pr, nil
}

func (s *Store) ListByRepo(ctx context.Context, repoID string, state State) ([]*PullRequest, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT pr.id, pr.repo_id, pr.number, pr.author_id, pr.title, COALESCE(pr.body,''),
               pr.head_branch, pr.base_branch, pr.state, pr.merge_commit_oid, pr.merged_at,
               pr.created_at, pr.updated_at, u.handle
          FROM platform.pull_requests pr
          LEFT JOIN platform.users u ON u.id = pr.author_id
         WHERE pr.repo_id = $1 AND ($2 = '' OR pr.state = $2)
         ORDER BY pr.number DESC
    `, repoID, string(state))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*PullRequest
	for rows.Next() {
		pr := &PullRequest{}
		if err := rows.Scan(
			&pr.ID, &pr.RepoID, &pr.Number, &pr.AuthorID, &pr.Title, &pr.Body,
			&pr.HeadBranch, &pr.BaseBranch, &pr.State, &pr.MergeCommitOID, &pr.MergedAt,
			&pr.CreatedAt, &pr.UpdatedAt, &pr.AuthorHandle,
		); err != nil {
			return nil, err
		}
		out = append(out, pr)
	}
	return out, rows.Err()
}

// MarkMerged transitions an open PR to merged with the given merge commit OID.
// Returns ErrNotOpen if the PR is not in the open state.
func (s *Store) MarkMerged(ctx context.Context, prID, mergedByID, mergeCommitOID string) error {
	cmd, err := s.pool.Exec(ctx, `
        UPDATE platform.pull_requests
           SET state = 'merged', merge_commit_oid = $2, merged_at = NOW(),
               merged_by = $3, updated_at = NOW()
         WHERE id = $1 AND state = 'open'
    `, prID, mergeCommitOID, mergedByID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotOpen
	}
	return nil
}

func (s *Store) AddComment(ctx context.Context, prID, authorID, body string) (*Comment, error) {
	c := &Comment{}
	err := s.pool.QueryRow(ctx, `
        INSERT INTO platform.pr_comments (pr_id, author_id, author_kind, body)
        VALUES ($1, $2, 'user', $3)
        RETURNING id, pr_id, author_id, author_kind, body, created_at
    `, prID, authorID, body).Scan(
		&c.ID, &c.PRID, &c.AuthorID, &c.AuthorKind, &c.Body, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// AddAgentComment appends a review comment authored by the system Reviewer
// Agent. author_id is NULL; UI keys off author_kind='agent' for badging.
func (s *Store) AddAgentComment(ctx context.Context, prID, body string) (*Comment, error) {
	c := &Comment{}
	err := s.pool.QueryRow(ctx, `
        INSERT INTO platform.pr_comments (pr_id, author_id, author_kind, body)
        VALUES ($1, NULL, 'agent', $2)
        RETURNING id, pr_id, author_id, author_kind, body, created_at
    `, prID, body).Scan(
		&c.ID, &c.PRID, &c.AuthorID, &c.AuthorKind, &c.Body, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetByID returns a PR by its UUID and looks up the owning repo.
func (s *Store) GetByID(ctx context.Context, prID string) (*PullRequest, string, string, error) {
	var pr PullRequest
	var ownerHandle, repoName string
	err := s.pool.QueryRow(ctx, `
        SELECT pr.id, pr.repo_id, pr.number, pr.author_id, pr.title, COALESCE(pr.body,''),
               pr.head_branch, pr.base_branch, pr.state, pr.merge_commit_oid, pr.merged_at,
               pr.created_at, pr.updated_at, u.handle,
               ou.handle, r.name
          FROM platform.pull_requests pr
          JOIN platform.repositories r ON r.id = pr.repo_id
          JOIN platform.users ou ON ou.id = r.owner_id
          LEFT JOIN platform.users u ON u.id = pr.author_id
         WHERE pr.id = $1
    `, prID).Scan(
		&pr.ID, &pr.RepoID, &pr.Number, &pr.AuthorID, &pr.Title, &pr.Body,
		&pr.HeadBranch, &pr.BaseBranch, &pr.State, &pr.MergeCommitOID, &pr.MergedAt,
		&pr.CreatedAt, &pr.UpdatedAt, &pr.AuthorHandle,
		&ownerHandle, &repoName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", "", ErrNotFound
		}
		return nil, "", "", err
	}
	return &pr, ownerHandle, repoName, nil
}

func (s *Store) ListComments(ctx context.Context, prID string) ([]*Comment, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT c.id, c.pr_id, c.author_id, c.author_kind, c.body, c.created_at, u.handle
          FROM platform.pr_comments c
          LEFT JOIN platform.users u ON u.id = c.author_id
         WHERE c.pr_id = $1
         ORDER BY c.created_at ASC
    `, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Comment
	for rows.Next() {
		c := &Comment{}
		if err := rows.Scan(&c.ID, &c.PRID, &c.AuthorID, &c.AuthorKind, &c.Body, &c.CreatedAt, &c.AuthorHandle); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
