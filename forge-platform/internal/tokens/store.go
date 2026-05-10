// Package tokens manages Personal Access Tokens — slice 12.
//
// Replaces the slice-4 user_git_secrets table. The interface is intentionally
// shaped to match what the Git transport needs for HTTP Basic auth:
//   user, err := tokens.Verify(ctx, handle, secret)
//   if err == nil && user != nil { /* allow */ }
package tokens

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/codecollab-co/forge/forge-platform/internal/users"
)

var (
	ErrNotFound       = errors.New("token not found")
	ErrDuplicateName  = errors.New("token name already exists for this user")
)

// Token mirrors the storage row minus the secret. The hash is intentionally
// excluded from the public struct — Verify() is the only caller that touches it.
type Token struct {
	ID         string
	UserID     string
	Name       string
	Scopes     []string
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time

	// SecretHash is exported only so JSON marshallers DON'T include it
	// elsewhere (we tag it with json:"-" callers; it's always empty in
	// the value the public API hands back). The store fills it only on
	// the Verify() internal path.
	SecretHash string
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Mint generates a fresh secret, hashes it with bcrypt, and inserts a token
// row. Returns the plaintext (caller shows it to the user once) plus the
// stored Token (without the hash). expiresIn=0 means no expiry.
func (s *Store) Mint(ctx context.Context, userID, name string, scopes []string, expiresIn time.Duration) (string, *Token, error) {
	plain, err := randomToken()
	if err != nil {
		return "", nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, err
	}
	if scopes == nil {
		scopes = []string{"git:read", "git:write"}
	}
	var expiresAt *time.Time
	if expiresIn != 0 {
		t := time.Now().Add(expiresIn)
		expiresAt = &t
	}

	tok := &Token{}
	err = s.pool.QueryRow(ctx, `
        INSERT INTO platform.personal_access_tokens (user_id, name, secret_hash, scopes, expires_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, user_id, name, scopes, expires_at, last_used_at, created_at
    `, userID, strings.TrimSpace(name), string(hash), scopes, expiresAt).
		Scan(&tok.ID, &tok.UserID, &tok.Name, &tok.Scopes, &tok.ExpiresAt, &tok.LastUsedAt, &tok.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return "", nil, ErrDuplicateName
		}
		return "", nil, err
	}
	return plain, tok, nil
}

// Verify looks the user up by handle, compares the supplied secret against
// every active (non-expired, non-revoked) token's bcrypt hash. Returns the
// user on first match, or nil when no token matches. Updates last_used_at on
// the matching token (best-effort; non-fatal).
//
// Constant-time bcrypt comparison guards against timing attacks per token,
// but the linear scan over a user's tokens leaks count via timing. With
// realistic token counts (<10) this is acceptable.
func (s *Store) Verify(ctx context.Context, handle, plain string) (*users.User, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT t.id, t.secret_hash,
               u.id, u.supertokens_id, u.provider, u.external_id, u.handle,
               COALESCE(u.email,''), COALESCE(u.display_name,''), COALESCE(u.avatar_url,''),
               u.created_at, u.updated_at
          FROM platform.personal_access_tokens t
          JOIN platform.users u ON u.id = t.user_id
         WHERE u.handle = $1
           AND (t.expires_at IS NULL OR t.expires_at > NOW())
    `, handle)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tokenID, hash string
		u := &users.User{}
		if err := rows.Scan(
			&tokenID, &hash,
			&u.ID, &u.SuperTokensID, &u.Provider, &u.ExternalID, &u.Handle,
			&u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil {
			rows.Close()
			_, _ = s.pool.Exec(ctx,
				`UPDATE platform.personal_access_tokens SET last_used_at = NOW() WHERE id = $1`,
				tokenID)
			return u, nil
		}
	}
	return nil, nil
}

// List returns the user's tokens with hash and secret omitted.
func (s *Store) List(ctx context.Context, userID string) ([]*Token, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT id, user_id, name, scopes, expires_at, last_used_at, created_at
          FROM platform.personal_access_tokens
         WHERE user_id = $1
         ORDER BY created_at DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Token
	for rows.Next() {
		t := &Token{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Scopes, &t.ExpiresAt, &t.LastUsedAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Revoke deletes the token. Returns ErrNotFound if the token doesn't exist
// or doesn't belong to the supplied user (so cross-user revocation can't
// be used to enumerate other users' token IDs).
func (s *Store) Revoke(ctx context.Context, userID, tokenID string) error {
	cmd, err := s.pool.Exec(ctx,
		`DELETE FROM platform.personal_access_tokens WHERE id = $1 AND user_id = $2`,
		tokenID, userID)
	if err != nil {
		// Map invalid-uuid errors to ErrNotFound for a tidy API.
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

// ByID is used by the API handler to look up a token before revoking, for
// returning its name in 200 responses.
func (s *Store) ByID(ctx context.Context, userID, tokenID string) (*Token, error) {
	t := &Token{}
	err := s.pool.QueryRow(ctx, `
        SELECT id, user_id, name, scopes, expires_at, last_used_at, created_at
          FROM platform.personal_access_tokens
         WHERE id = $1 AND user_id = $2
    `, tokenID, userID).Scan(&t.ID, &t.UserID, &t.Name, &t.Scopes, &t.ExpiresAt, &t.LastUsedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "fgp_" + base64.RawURLEncoding.EncodeToString(buf), nil
}
