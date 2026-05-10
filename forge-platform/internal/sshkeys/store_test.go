package sshkeys_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/ssh"

	"github.com/codecollab-co/forge/forge-platform/internal/sshkeys"
	"github.com/codecollab-co/forge/forge-platform/internal/users"
)

// freshKey generates a brand-new ed25519 keypair per test so re-running the
// suite doesn't trip the unique-fingerprint constraint from a prior insert.
func freshKey(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return string(ssh.MarshalAuthorizedKey(sshPub))
}

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("FORGE_TEST_PG")
	if dsn == "" {
		t.Skip("FORGE_TEST_PG not set; skipping integration tests")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func fixtureUser(t *testing.T, pool *pgxpool.Pool) *users.User {
	t.Helper()
	usersRepo := users.NewRepo(pool)
	uniq := time.Now().Format("150405.000000000")
	u, err := usersRepo.UpsertOnSignInUp(context.Background(), users.SignInUpInput{
		SuperTokensID: "stk-" + uniq,
		Provider:      "test",
		ExternalID:    "ext-" + uniq,
		Email:         "t" + uniq + "@forge.local",
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestAdd_ReturnsKeyWithFingerprint(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := sshkeys.NewStore(pool)
	k, err := store.Add(context.Background(), u.ID, "laptop", freshKey(t))
	if err != nil {
		t.Fatal(err)
	}
	if k.Fingerprint == "" || k.Name != "laptop" {
		t.Fatalf("unexpected: %+v", k)
	}
	if !startsWith(k.Fingerprint, "SHA256:") {
		t.Errorf("fingerprint should be SHA256: prefix, got %q", k.Fingerprint)
	}
}

func TestAdd_RejectsMalformedKey(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := sshkeys.NewStore(pool)
	_, err := store.Add(context.Background(), u.ID, "garbage", "not-a-real-ssh-key")
	if err == nil {
		t.Fatal("expected error for malformed key")
	}
}

func TestAdd_RejectsDuplicateFingerprintAcrossUsers(t *testing.T) {
	pool := newPool(t)
	a := fixtureUser(t, pool)
	b := fixtureUser(t, pool)
	store := sshkeys.NewStore(pool)
	shared := freshKey(t)
	if _, err := store.Add(context.Background(), a.ID, "k", shared); err != nil {
		t.Fatal(err)
	}
	_, err := store.Add(context.Background(), b.ID, "k", shared)
	if err == nil {
		t.Fatal("expected duplicate-fingerprint rejection")
	}
}

func TestByFingerprint_FindsTheUser(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := sshkeys.NewStore(pool)
	k, _ := store.Add(context.Background(), u.ID, "laptop", freshKey(t))
	got, err := store.ByFingerprint(context.Background(), k.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != u.ID {
		t.Fatalf("got %+v, want user %s", got, u.ID)
	}
}

func TestByFingerprint_UnknownReturnsNil(t *testing.T) {
	pool := newPool(t)
	store := sshkeys.NewStore(pool)
	got, _ := store.ByFingerprint(context.Background(), "SHA256:nonexistent")
	if got != nil {
		t.Errorf("expected nil for unknown fp, got %+v", got)
	}
}

func TestRevoke_PreventsLookup(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := sshkeys.NewStore(pool)
	k, _ := store.Add(context.Background(), u.ID, "laptop", freshKey(t))
	if err := store.Revoke(context.Background(), u.ID, k.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := store.ByFingerprint(context.Background(), k.Fingerprint)
	if got != nil {
		t.Fatal("revoked key still resolvable")
	}
}

func TestRevoke_OtherUsersKeyForbidden(t *testing.T) {
	pool := newPool(t)
	a := fixtureUser(t, pool)
	b := fixtureUser(t, pool)
	store := sshkeys.NewStore(pool)
	k, _ := store.Add(context.Background(), a.ID, "k", freshKey(t))
	err := store.Revoke(context.Background(), b.ID, k.ID)
	if !errors.Is(err, sshkeys.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func startsWith(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }
