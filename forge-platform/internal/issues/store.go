// Package issues persists Issues and their comments.
//
// Slice 6 supports human assignees only. The assignee field is intentionally
// shaped so slice 7 can add Agent assignment without breaking callers.
package issues

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("issue not found")

type State string

const (
	StateOpen   State = "open"
	StateClosed State = "closed"
)

type AssigneeKind string

const (
	AssigneeNone  AssigneeKind = ""
	AssigneeUser  AssigneeKind = "user"
	AssigneeAgent AssigneeKind = "agent"
)

type Issue struct {
	ID                  string
	RepoID              string
	Number              int
	AuthorID            *string
	AuthorHandle        *string
	Title               string
	Body                string
	State               State
	AssigneeUserID      *string
	AssigneeUserHandle  *string
	ClosedAt            *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (i *Issue) AssigneeKind() AssigneeKind {
	if i.AssigneeUserID != nil {
		return AssigneeUser
	}
	return AssigneeNone
}

type Comment struct {
	ID           string
	IssueID      string
	AuthorID     *string
	AuthorHandle *string
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
	RepoID         string
	AuthorID       string
	Title          string
	Body           string
	AssigneeUserID string // optional
}

func (s *Store) Create(ctx context.Context, in CreateInput) (*Issue, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var number int
	if err := tx.QueryRow(ctx, `
        UPDATE platform.repositories
           SET next_issue_number = next_issue_number + 1, updated_at = NOW()
         WHERE id = $1
     RETURNING next_issue_number - 1
    `, in.RepoID).Scan(&number); err != nil {
		return nil, err
	}

	var assignee any
	if in.AssigneeUserID != "" {
		assignee = in.AssigneeUserID
	}

	iss := &Issue{}
	if err := tx.QueryRow(ctx, `
        INSERT INTO platform.issues
            (repo_id, number, author_id, title, body, assignee_user_id)
        VALUES ($1, $2, $3, $4, NULLIF($5,''), $6)
        RETURNING id, repo_id, number, author_id, title, COALESCE(body,''),
                  state, assignee_user_id, closed_at, created_at, updated_at
    `, in.RepoID, number, in.AuthorID, in.Title, in.Body, assignee).
		Scan(&iss.ID, &iss.RepoID, &iss.Number, &iss.AuthorID, &iss.Title, &iss.Body,
			&iss.State, &iss.AssigneeUserID, &iss.ClosedAt, &iss.CreatedAt, &iss.UpdatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return iss, nil
}

const selectIssueWithJoins = `
    SELECT i.id, i.repo_id, i.number, i.author_id, i.title, COALESCE(i.body,''),
           i.state, i.assignee_user_id, i.closed_at, i.created_at, i.updated_at,
           a.handle, au.handle
      FROM platform.issues i
      LEFT JOIN platform.users a  ON a.id  = i.author_id
      LEFT JOIN platform.users au ON au.id = i.assignee_user_id
`

func scanIssue(row pgx.Row) (*Issue, error) {
	iss := &Issue{}
	err := row.Scan(
		&iss.ID, &iss.RepoID, &iss.Number, &iss.AuthorID, &iss.Title, &iss.Body,
		&iss.State, &iss.AssigneeUserID, &iss.ClosedAt, &iss.CreatedAt, &iss.UpdatedAt,
		&iss.AuthorHandle, &iss.AssigneeUserHandle,
	)
	return iss, err
}

func (s *Store) GetByRepoAndNumber(ctx context.Context, repoID string, number int) (*Issue, error) {
	row := s.pool.QueryRow(ctx, selectIssueWithJoins+`WHERE i.repo_id = $1 AND i.number = $2`, repoID, number)
	iss, err := scanIssue(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return iss, nil
}

func (s *Store) ListByRepo(ctx context.Context, repoID string, state State) ([]*Issue, error) {
	rows, err := s.pool.Query(ctx,
		selectIssueWithJoins+`WHERE i.repo_id = $1 AND ($2 = '' OR i.state = $2) ORDER BY i.number DESC`,
		repoID, string(state),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Issue
	for rows.Next() {
		iss, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, iss)
	}
	return out, rows.Err()
}

func (s *Store) Close(ctx context.Context, issueID, byUserID string) error {
	cmd, err := s.pool.Exec(ctx, `
        UPDATE platform.issues
           SET state = 'closed', closed_at = NOW(), closed_by = $2, updated_at = NOW()
         WHERE id = $1 AND state = 'open'
    `, issueID, byUserID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Reopen(ctx context.Context, issueID string) error {
	cmd, err := s.pool.Exec(ctx, `
        UPDATE platform.issues
           SET state = 'open', closed_at = NULL, closed_by = NULL, updated_at = NOW()
         WHERE id = $1 AND state = 'closed'
    `, issueID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AddComment(ctx context.Context, issueID, authorID, body string) (*Comment, error) {
	c := &Comment{}
	err := s.pool.QueryRow(ctx, `
        INSERT INTO platform.issue_comments (issue_id, author_id, body)
        VALUES ($1, $2, $3)
        RETURNING id, issue_id, author_id, body, created_at
    `, issueID, authorID, body).Scan(&c.ID, &c.IssueID, &c.AuthorID, &c.Body, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) ListComments(ctx context.Context, issueID string) ([]*Comment, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT c.id, c.issue_id, c.author_id, c.body, c.created_at, u.handle
          FROM platform.issue_comments c
          LEFT JOIN platform.users u ON u.id = c.author_id
         WHERE c.issue_id = $1
         ORDER BY c.created_at ASC
    `, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Comment
	for rows.Next() {
		c := &Comment{}
		if err := rows.Scan(&c.ID, &c.IssueID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.AuthorHandle); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
