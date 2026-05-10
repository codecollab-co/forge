package auth_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/codecollab-co/forge/forge-cli/internal/auth"
)

func TestStoreLoad_ReturnsErrNotLoggedInWhenEmpty(t *testing.T) {
	s := auth.NewStore(t.TempDir())
	_, err := s.Load()
	if !errors.Is(err, auth.ErrNotLoggedIn) {
		t.Fatalf("expected ErrNotLoggedIn, got %v", err)
	}
}

func TestStoreSaveThenLoad_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	s := auth.NewStore(dir)
	in := auth.Credentials{
		APIURL: "http://localhost:8080",
		Token:  "abc.def.ghi",
		Handle: "alice",
	}
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != in {
		t.Errorf("round-trip mismatch: %+v vs %+v", got, in)
	}
	// Different store instance, same dir — persistence across processes.
	if other, _ := auth.NewStore(dir).Load(); other != in {
		t.Errorf("re-opened store didn't see saved credentials: %+v", other)
	}
}

func TestStoreSave_WritesFileWith0600Mode(t *testing.T) {
	dir := t.TempDir()
	s := auth.NewStore(dir)
	if err := s.Save(auth.Credentials{APIURL: "x", Token: "y", Handle: "z"}); err != nil {
		t.Fatal(err)
	}
	info, err := statFile(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.mode&0o777 != 0o600 {
		t.Errorf("credentials.json mode = %o, want 0600", info.mode&0o777)
	}
}

func TestStoreClear_LeavesNothingToLoad(t *testing.T) {
	s := auth.NewStore(t.TempDir())
	_ = s.Save(auth.Credentials{APIURL: "x", Token: "y", Handle: "z"})
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); !errors.Is(err, auth.ErrNotLoggedIn) {
		t.Errorf("after Clear, Load should return ErrNotLoggedIn, got %v", err)
	}
}
