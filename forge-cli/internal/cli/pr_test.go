package cli_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codecollab-co/forge/forge-cli/internal/cli"
)

func TestPRList_OpenByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/pulls") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("state filter = %q", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "p1", "number": 3, "title": "Fix nav", "state": "open",
				"author": "alice", "head_branch": "fix-nav", "base_branch": "main"},
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"pr", "list", "alice/foo"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "#3") || !strings.Contains(out, "fix-nav → main") {
		t.Errorf("unexpected: %q", out)
	}
}

func TestPRView_RendersAndOptionalDiff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pull_request": map[string]any{
				"id": "p1", "number": 5, "title": "PR title", "state": "open",
				"author": "alice", "head_branch": "h", "base_branch": "main",
				"body": "details",
			},
			"diff": "--- a/x.go\n+++ b/x.go\n@@\n-old\n+new\n",
			"comments": []map[string]any{
				{"id": "c1", "author": "forge-reviewer", "author_kind": "agent", "body": "looks fine"},
			},
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"pr", "view", "alice/foo", "5"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "#5 PR title") || !strings.Contains(out, "details") {
		t.Errorf("missing fields: %q", out)
	}
	if !strings.Contains(out, "(agent)") {
		t.Errorf("expected agent badge: %q", out)
	}
	if strings.Contains(out, "+new") {
		t.Errorf("diff should not appear without --diff: %q", out)
	}

	stdout.Reset()
	if err := cli.Run([]string{"pr", "view", "alice/foo", "5", "--diff"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "+new") {
		t.Errorf("expected diff with --diff, got %q", stdout.String())
	}
}

func TestPRCreate_RequiresFlags(t *testing.T) {
	stubLoggedIn(t, "http://unused")
	err := cli.Run([]string{"pr", "create", "alice/foo", "--title", "T"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "head") {
		t.Fatalf("expected head/base required error, got %v", err)
	}
}

func TestPRCreate_Happy(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "p1", "number": 9, "title": got["title"],
			"head_branch": got["head_branch"], "base_branch": got["base_branch"],
			"state": "open",
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	err := cli.Run([]string{
		"pr", "create", "alice/foo",
		"--title", "Refactor x", "--body", "rationale",
		"--head", "feat-x", "--base", "main",
	}, &stdout, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if got["head_branch"] != "feat-x" || got["base_branch"] != "main" {
		t.Errorf("body = %+v", got)
	}
	if !strings.Contains(stdout.String(), "Opened #9") {
		t.Errorf("output = %q", stdout.String())
	}
}

func TestPRMerge_DefaultDeletesBranch(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/merge") {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"merge_commit_oid": "abcdef1234567890aaaa", "state": "merged",
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"pr", "merge", "alice/foo", "1"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got["delete_branch"] != true {
		t.Errorf("expected delete_branch=true by default, got %+v", got)
	}
	if !strings.Contains(stdout.String(), "Merged #1 as abcdef123456") {
		t.Errorf("output = %q", stdout.String())
	}
}

func TestPRMerge_KeepBranchFlag(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"merge_commit_oid": "0000000000000000aaaa", "state": "merged",
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)
	if err := cli.Run([]string{"pr", "merge", "alice/foo", "1", "--keep-branch"}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got["delete_branch"] != false {
		t.Errorf("expected delete_branch=false with --keep-branch, got %+v", got)
	}
}
