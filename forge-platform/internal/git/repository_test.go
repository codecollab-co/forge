package git

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

func TestInit_CreatesBareRepo(t *testing.T) {
	skipIfNoGit(t)
	r, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Init(context.Background(), "alice", "hello-world"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !r.Exists("alice", "hello-world") {
		t.Fatal("Exists() = false after Init")
	}
}

func TestInit_RejectsDuplicate(t *testing.T) {
	skipIfNoGit(t)
	r, _ := New(t.TempDir())
	ctx := context.Background()
	if err := r.Init(ctx, "alice", "repo"); err != nil {
		t.Fatal(err)
	}
	err := r.Init(ctx, "alice", "repo")
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestInit_RejectsInvalidNames(t *testing.T) {
	skipIfNoGit(t)
	r, _ := New(t.TempDir())
	for _, c := range []struct{ owner, name string }{
		{"", "repo"},
		{"alice", ""},
		{"Alice", "repo"},
		{"alice", "Repo"},
		{"alice", "-leading"},
		{"alice", "trailing-"},
		{"alice", "with space"},
		{"../etc", "repo"},
	} {
		err := r.Init(context.Background(), c.owner, c.name)
		if err == nil {
			t.Errorf("expected error for owner=%q name=%q", c.owner, c.name)
		}
	}
}

func TestListRefs_EmptyRepo(t *testing.T) {
	skipIfNoGit(t)
	r, _ := New(t.TempDir())
	if err := r.Init(context.Background(), "alice", "empty"); err != nil {
		t.Fatal(err)
	}
	refs, err := r.ListRefs(context.Background(), "alice", "empty")
	if err != nil {
		t.Fatalf("ListRefs: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected no refs on empty repo, got %d", len(refs))
	}
}

func TestListRefs_MissingRepo(t *testing.T) {
	skipIfNoGit(t)
	r, _ := New(t.TempDir())
	_, err := r.ListRefs(context.Background(), "alice", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
