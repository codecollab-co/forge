package users

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// GenerateGitSecret mints a new git-push secret for the user, persists its
// bcrypt hash, and returns the plaintext exactly once.
func (r *Repo) GenerateGitSecret(ctx context.Context, userID string) (string, error) {
	plain, err := randomToken()
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	_, err = r.pool.Exec(ctx, `
        INSERT INTO platform.user_git_secrets (user_id, secret_hash)
        VALUES ($1, $2)
        ON CONFLICT (user_id) DO UPDATE SET
            secret_hash = EXCLUDED.secret_hash,
            created_at  = NOW(),
            last_used_at = NULL
    `, userID, string(hash))
	if err != nil {
		return "", err
	}
	return plain, nil
}

// VerifyGitSecret looks up the user by handle, compares the supplied secret
// against the stored bcrypt hash, and on success returns the user. Constant-
// time bcrypt comparison guards against timing attacks.
func (r *Repo) VerifyGitSecret(ctx context.Context, handle, secret string) (*User, error) {
	var (
		user *User
		hash string
	)
	row := r.pool.QueryRow(ctx, `
        SELECT u.id, u.supertokens_id, u.provider, u.external_id, u.handle,
               COALESCE(u.email,''), COALESCE(u.display_name,''), COALESCE(u.avatar_url,''),
               u.created_at, u.updated_at,
               s.secret_hash
          FROM platform.users u
          JOIN platform.user_git_secrets s ON s.user_id = u.id
         WHERE u.handle = $1
    `, handle)
	user = &User{}
	if err := row.Scan(
		&user.ID, &user.SuperTokensID, &user.Provider, &user.ExternalID, &user.Handle,
		&user.Email, &user.DisplayName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt,
		&hash,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)); err != nil {
		return nil, nil
	}
	_, _ = r.pool.Exec(ctx,
		`UPDATE platform.user_git_secrets SET last_used_at = NOW() WHERE user_id = $1`,
		user.ID)
	return user, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "fg_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

// GitSecretInfo returns whether the user has minted a secret and the time it
// was last used (nil if never), without exposing the hash.
type GitSecretInfo struct {
	Exists     bool
	CreatedAt  *time.Time
	LastUsedAt *time.Time
}

func (r *Repo) GitSecretInfo(ctx context.Context, userID string) (GitSecretInfo, error) {
	var info GitSecretInfo
	row := r.pool.QueryRow(ctx, `
        SELECT created_at, last_used_at
          FROM platform.user_git_secrets WHERE user_id = $1
    `, userID)
	var created time.Time
	var lastUsed *time.Time
	if err := row.Scan(&created, &lastUsed); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return info, nil
		}
		return info, err
	}
	info.Exists = true
	info.CreatedAt = &created
	info.LastUsedAt = lastUsed
	return info, nil
}
