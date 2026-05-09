// Package repos persists Repository rows. The on-disk Git side lives in
// internal/git; this package is the platform.repositories table only.
package repos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyExists = errors.New("repository already exists")
	ErrNotFound      = errors.New("repository not found")
)

type Repository struct {
	ID          string
	OwnerID     string
	OwnerHandle string
	Name        string
	Description string
	Visibility  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type CreateInput struct {
	OwnerID     string
	Name        string
	Description string
	Visibility  string
}

func (s *Store) Create(ctx context.Context, in CreateInput) (*Repository, error) {
	if in.Visibility == "" {
		in.Visibility = "public"
	}
	row := s.pool.QueryRow(ctx, `
        INSERT INTO platform.repositories (owner_id, name, description, visibility)
        VALUES ($1, $2, NULLIF($3,''), $4)
        ON CONFLICT (owner_id, name) DO NOTHING
        RETURNING id, owner_id, name, COALESCE(description,''), visibility, created_at, updated_at
    `, in.OwnerID, in.Name, in.Description, in.Visibility)
	r := &Repository{}
	if err := row.Scan(&r.ID, &r.OwnerID, &r.Name, &r.Description, &r.Visibility, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	return r, nil
}

func (s *Store) ListByOwnerID(ctx context.Context, ownerID string) ([]*Repository, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT r.id, r.owner_id, u.handle, r.name, COALESCE(r.description,''), r.visibility, r.created_at, r.updated_at
          FROM platform.repositories r
          JOIN platform.users u ON u.id = r.owner_id
         WHERE r.owner_id = $1
         ORDER BY r.created_at DESC
    `, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Repository
	for rows.Next() {
		r := &Repository{}
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.OwnerHandle, &r.Name, &r.Description, &r.Visibility, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetByOwnerHandleAndName(ctx context.Context, ownerHandle, name string) (*Repository, error) {
	row := s.pool.QueryRow(ctx, `
        SELECT r.id, r.owner_id, u.handle, r.name, COALESCE(r.description,''), r.visibility, r.created_at, r.updated_at
          FROM platform.repositories r
          JOIN platform.users u ON u.id = r.owner_id
         WHERE u.handle = $1 AND r.name = $2
    `, ownerHandle, name)
	r := &Repository{}
	if err := row.Scan(&r.ID, &r.OwnerID, &r.OwnerHandle, &r.Name, &r.Description, &r.Visibility, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return r, nil
}
