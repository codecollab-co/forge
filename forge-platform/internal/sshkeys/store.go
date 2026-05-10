// Package sshkeys persists SSH public keys for git-over-SSH auth.
//
// Add / List / Revoke are session-authed (web UI). ByFingerprint is the
// hot path used by the SSH server during the public-key auth callback.
package sshkeys

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/ssh"

	"github.com/codecollab-co/forge/forge-platform/internal/users"
)

var (
	ErrNotFound        = errors.New("ssh key not found")
	ErrDuplicate       = errors.New("ssh key already registered")
	ErrInvalidKey      = errors.New("could not parse ssh public key")
)

type SSHKey struct {
	ID          string
	UserID      string
	Name        string
	Fingerprint string
	PublicKey   string
	LastUsedAt  *time.Time
	CreatedAt   time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Add parses the supplied authorized_keys line, computes its SHA256
// fingerprint, and persists it.
func (s *Store) Add(ctx context.Context, userID, name, authorizedKey string) (*SSHKey, error) {
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(authorizedKey)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	fp := ssh.FingerprintSHA256(pub)
	k := &SSHKey{}
	err = s.pool.QueryRow(ctx, `
        INSERT INTO platform.ssh_keys (user_id, name, fingerprint, public_key)
        VALUES ($1, $2, $3, $4)
        RETURNING id, user_id, name, fingerprint, public_key, last_used_at, created_at
    `, userID, strings.TrimSpace(name), fp, strings.TrimSpace(authorizedKey)).
		Scan(&k.ID, &k.UserID, &k.Name, &k.Fingerprint, &k.PublicKey, &k.LastUsedAt, &k.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, err
	}
	return k, nil
}

// ByFingerprint returns the user that owns the key with the given
// SHA256 fingerprint, or nil if no match. Updates last_used_at on
// match (best-effort, non-fatal).
func (s *Store) ByFingerprint(ctx context.Context, fp string) (*users.User, error) {
	u := &users.User{}
	err := s.pool.QueryRow(ctx, `
        SELECT u.id, u.supertokens_id, u.provider, u.external_id, u.handle,
               COALESCE(u.email,''), COALESCE(u.display_name,''), COALESCE(u.avatar_url,''),
               u.created_at, u.updated_at
          FROM platform.ssh_keys k
          JOIN platform.users u ON u.id = k.user_id
         WHERE k.fingerprint = $1
    `, fp).Scan(
		&u.ID, &u.SuperTokensID, &u.Provider, &u.ExternalID, &u.Handle,
		&u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	_, _ = s.pool.Exec(ctx,
		`UPDATE platform.ssh_keys SET last_used_at = NOW() WHERE fingerprint = $1`, fp)
	return u, nil
}

func (s *Store) List(ctx context.Context, userID string) ([]*SSHKey, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT id, user_id, name, fingerprint, public_key, last_used_at, created_at
          FROM platform.ssh_keys
         WHERE user_id = $1
         ORDER BY created_at DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*SSHKey
	for rows.Next() {
		k := &SSHKey{}
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.Fingerprint, &k.PublicKey, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) Revoke(ctx context.Context, userID, keyID string) error {
	cmd, err := s.pool.Exec(ctx,
		`DELETE FROM platform.ssh_keys WHERE id = $1 AND user_id = $2`,
		keyID, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return ErrNotFound
		}
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
