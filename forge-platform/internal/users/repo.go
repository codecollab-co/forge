// Package users persists the platform.users table.
//
// At MVP, rows are created from the SuperTokens OnSignInUp hook (slice 2).
// Future slices add Repository ownership and permissions on top of these rows.
package users

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID            string
	SuperTokensID string
	Provider      string
	ExternalID    string
	Handle        string
	Email         string
	DisplayName   string
	AvatarURL     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

type SignInUpInput struct {
	SuperTokensID string
	Provider      string
	ExternalID    string
	Email         string
	DisplayName   string
	AvatarURL     string
}

// UpsertOnSignInUp creates the user on first sign-in, or refreshes mutable
// profile fields on subsequent sign-ins. Idempotent.
func (r *Repo) UpsertOnSignInUp(ctx context.Context, in SignInUpInput) (*User, error) {
	handle, err := r.allocateHandle(ctx, in.Email, in.DisplayName, in.Provider, in.ExternalID)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `
        INSERT INTO platform.users (
            supertokens_id, provider, external_id, handle, email, display_name, avatar_url
        ) VALUES ($1, $2, $3, $4, NULLIF($5,''), NULLIF($6,''), NULLIF($7,''))
        ON CONFLICT (supertokens_id) DO UPDATE SET
            email        = EXCLUDED.email,
            display_name = EXCLUDED.display_name,
            avatar_url   = EXCLUDED.avatar_url,
            updated_at   = NOW()
        RETURNING id, supertokens_id, provider, external_id, handle,
                  COALESCE(email,''), COALESCE(display_name,''), COALESCE(avatar_url,''),
                  created_at, updated_at
    `, in.SuperTokensID, in.Provider, in.ExternalID, handle, in.Email, in.DisplayName, in.AvatarURL)
	u := &User{}
	if err := row.Scan(
		&u.ID, &u.SuperTokensID, &u.Provider, &u.ExternalID, &u.Handle,
		&u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Repo) ByID(ctx context.Context, id string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, supertokens_id, provider, external_id, handle,
               COALESCE(email,''), COALESCE(display_name,''), COALESCE(avatar_url,''),
               created_at, updated_at
          FROM platform.users WHERE id = $1
    `, id)
	u := &User{}
	if err := row.Scan(
		&u.ID, &u.SuperTokensID, &u.Provider, &u.ExternalID, &u.Handle,
		&u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (r *Repo) BySuperTokensID(ctx context.Context, stID string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, supertokens_id, provider, external_id, handle,
               COALESCE(email,''), COALESCE(display_name,''), COALESCE(avatar_url,''),
               created_at, updated_at
          FROM platform.users WHERE supertokens_id = $1
    `, stID)
	u := &User{}
	if err := row.Scan(
		&u.ID, &u.SuperTokensID, &u.Provider, &u.ExternalID, &u.Handle,
		&u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

// RenameHandle updates the user's handle. Caller must move on-disk repo
// storage (GitRepository.MoveOwner) — this only touches the DB row.
func (r *Repo) RenameHandle(ctx context.Context, userID, newHandle string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE platform.users SET handle = $2, updated_at = NOW() WHERE id = $1`,
		userID, newHandle,
	)
	return err
}

func (r *Repo) HandleAvailable(ctx context.Context, handle string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM platform.users WHERE handle = $1)`, handle,
	).Scan(&exists)
	return !exists, err
}

var handleSafe = regexp.MustCompile(`[^a-z0-9-]+`)

func (r *Repo) allocateHandle(ctx context.Context, email, displayName, provider, externalID string) (string, error) {
	candidate := suggestHandle(email, displayName, provider, externalID)
	for attempt := 0; attempt < 20; attempt++ {
		try := candidate
		if attempt > 0 {
			try = fmt.Sprintf("%s-%d", candidate, attempt+1)
		}
		var exists bool
		if err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM platform.users WHERE handle = $1)`, try,
		).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return try, nil
		}
	}
	return "", errors.New("could not allocate a unique handle")
}

func suggestHandle(email, displayName, provider, externalID string) string {
	source := ""
	if email != "" {
		source = strings.SplitN(email, "@", 2)[0]
	} else if displayName != "" {
		source = displayName
	} else {
		source = provider + "-" + externalID
	}
	source = strings.ToLower(source)
	source = handleSafe.ReplaceAllString(source, "-")
	source = strings.Trim(source, "-")
	if len(source) < 2 {
		source = "user-" + externalID
	}
	if len(source) > 32 {
		source = source[:32]
	}
	return source
}
