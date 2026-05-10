package tokens_test

// Tests live next to a tokens.Store implementation. They run only when
// FORGE_TEST_PG is set, pointing at a Postgres with the platform schema
// already migrated. CI provides this; locally, run:
//
//   FORGE_TEST_PG="postgres://forge:forge@localhost:5432/forge?sslmode=disable" go test ./internal/tokens/...

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codecollab-co/forge/forge-platform/internal/tokens"
	"github.com/codecollab-co/forge/forge-platform/internal/users"
)

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("FORGE_TEST_PG")
	if dsn == "" {
		t.Skip("FORGE_TEST_PG not set; skipping integration tests")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// fixtureUser inserts a synthetic user we can attach tokens to. Each call
// gets a unique identifier so parallel tests don't collide.
func fixtureUser(t *testing.T, pool *pgxpool.Pool) *users.User {
	t.Helper()
	usersRepo := users.NewRepo(pool)
	uniq := time.Now().Format("150405.000000000")
	u, err := usersRepo.UpsertOnSignInUp(context.Background(), users.SignInUpInput{
		SuperTokensID: "stk-" + uniq,
		Provider:      "test",
		ExternalID:    "ext-" + uniq,
		Email:         "t" + uniq + "@forge.local",
		DisplayName:   "Test",
	})
	if err != nil {
		t.Fatalf("fixture user: %v", err)
	}
	return u
}

func TestMint_ReturnsPlaintextOnce(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := tokens.NewStore(pool)
	plain, tok, err := store.Mint(context.Background(), u.ID, "laptop", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if plain == "" || tok.ID == "" || tok.Name != "laptop" {
		t.Fatalf("plain=%q tok=%+v", plain, tok)
	}
	// Listing the token must NOT expose plaintext or hash.
	list, err := store.List(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "laptop" {
		t.Fatalf("list = %+v", list)
	}
	// Token struct has no Plaintext field — compile-time guarantee. Spot
	// check that the hash isn't accidentally returned.
	if list[0].SecretHash != "" {
		t.Errorf("List() leaked SecretHash: %q", list[0].SecretHash)
	}
}

func TestMint_RejectsDuplicateNamePerUser(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := tokens.NewStore(pool)
	if _, _, err := store.Mint(context.Background(), u.ID, "ci", nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Mint(context.Background(), u.ID, "ci", nil, 0); err == nil {
		t.Fatal("expected duplicate-name rejection")
	}
}

func TestVerify_HappyPath(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := tokens.NewStore(pool)
	plain, _, err := store.Mint(context.Background(), u.ID, "k1", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Verify(context.Background(), u.Handle, plain)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != u.ID {
		t.Fatalf("Verify returned %+v, want user %s", got, u.ID)
	}
}

func TestVerify_WrongSecretReturnsNil(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := tokens.NewStore(pool)
	if _, _, err := store.Mint(context.Background(), u.ID, "k1", nil, 0); err != nil {
		t.Fatal(err)
	}
	got, err := store.Verify(context.Background(), u.Handle, "bogus")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("Verify with wrong secret returned %+v, want nil", got)
	}
}

func TestRevoke_PreventsSubsequentVerify(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := tokens.NewStore(pool)
	plain, tok, err := store.Mint(context.Background(), u.ID, "k1", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Revoke(context.Background(), u.ID, tok.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Verify(context.Background(), u.Handle, plain)
	if got != nil {
		t.Fatal("revoked token still verifies")
	}
}

func TestRevoke_OtherUsersTokenForbidden(t *testing.T) {
	pool := newPool(t)
	a := fixtureUser(t, pool)
	b := fixtureUser(t, pool)
	store := tokens.NewStore(pool)
	_, tok, err := store.Mint(context.Background(), a.ID, "k1", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Revoke(context.Background(), b.ID, tok.ID)
	if !errors.Is(err, tokens.ErrNotFound) {
		t.Fatalf("expected ErrNotFound when revoking another user's token, got %v", err)
	}
}

func TestVerify_ExpiredTokenRejected(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := tokens.NewStore(pool)
	// negative TTL → already expired
	plain, _, err := store.Mint(context.Background(), u.ID, "k1", nil, -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := store.Verify(context.Background(), u.Handle, plain)
	if got != nil {
		t.Fatal("expired token verified successfully")
	}
}

func TestVerify_LastUsedTimestampUpdates(t *testing.T) {
	pool := newPool(t)
	u := fixtureUser(t, pool)
	store := tokens.NewStore(pool)
	plain, _, err := store.Mint(context.Background(), u.ID, "k1", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Verify(context.Background(), u.Handle, plain); err != nil {
		t.Fatal(err)
	}
	list, _ := store.List(context.Background(), u.ID)
	if len(list) != 1 || list[0].LastUsedAt == nil {
		t.Fatalf("LastUsedAt not set: %+v", list)
	}
}
