package cli_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codecollab-co/forge/forge-cli/internal/auth"
	repocmd "github.com/codecollab-co/forge/forge-cli/internal/cmd/repo"
	"github.com/codecollab-co/forge/forge-cli/internal/cli"
)

// stubLoggedIn writes a credentials.json into a fresh FORGE_CONFIG_DIR
// pointing at the supplied API URL, so subsequent commands skip login.
func stubLoggedIn(t *testing.T, apiURL string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("FORGE_CONFIG_DIR", dir)
	body, _ := json.Marshal(auth.Credentials{
		APIURL: apiURL, Token: "tok-test", Handle: "alice",
	})
	if err := os.WriteFile(filepath.Join(dir, "credentials.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRepoList_HitsAPIAndPrintsTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos" {
			t.Errorf("unexpected: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "r1", "owner": "alice", "name": "first", "visibility": "public", "description": "first repo", "clone_url": "/alice/first.git"},
			{"id": "r2", "owner": "alice", "name": "second", "visibility": "private", "description": "", "clone_url": "/alice/second.git"},
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"repo", "list"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	if !strings.Contains(got, "alice/first") || !strings.Contains(got, "first repo") {
		t.Errorf("expected first repo in output, got %q", got)
	}
	if !strings.Contains(got, "alice/second") || !strings.Contains(got, "private") {
		t.Errorf("expected second repo + private label, got %q", got)
	}
}

func TestRepoList_JSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "r1", "owner": "alice", "name": "x"},
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"repo", "list", "--json"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("not valid JSON: %v\nbody=%q", err, stdout.String())
	}
	if len(got) != 1 || got[0]["name"] != "x" {
		t.Errorf("unexpected JSON: %v", got)
	}
}

func TestRepoCreate_PassesFlags(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "r1", "owner": "alice", "name": got["name"], "visibility": got["visibility"],
			"clone_url": "/alice/" + got["name"].(string) + ".git",
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	err := cli.Run([]string{
		"repo", "create", "newone",
		"--description", "shiny",
		"--private",
		"--readme",
	}, &stdout, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if got["name"] != "newone" || got["visibility"] != "private" || got["init_readme"] != true {
		t.Errorf("body = %+v", got)
	}
	if !strings.Contains(stdout.String(), "Created alice/newone (private)") {
		t.Errorf("output = %q", stdout.String())
	}
}

func TestRepoDelete_RequiresYesFlag(t *testing.T) {
	stubLoggedIn(t, "http://unused")
	var stderr bytes.Buffer
	err := cli.Run([]string{"repo", "delete", "alice/foo"}, io.Discard, &stderr)
	if err == nil {
		t.Fatal("expected error without --yes")
	}
	if !strings.Contains(err.Error(), "refusing") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRepoDelete_HappyPathHits204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/alice/foo" || r.Method != http.MethodDelete {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)
	var stdout bytes.Buffer
	if err := cli.Run([]string{"repo", "delete", "alice/foo", "--yes"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Deleted alice/foo") {
		t.Errorf("output = %q", stdout.String())
	}
}

func TestRepoClone_BuildsExpectedGitArgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "r1", "owner": "alice", "name": "foo", "visibility": "public",
			"clone_url": "/alice/foo.git",
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var capturedArgs []string
	repocmd.CloneExec = func(args ...string) error {
		capturedArgs = args
		return nil
	}
	t.Cleanup(func() { repocmd.CloneExec = nil })

	var stdout bytes.Buffer
	if err := cli.Run([]string{"repo", "clone", "alice/foo", "/tmp/foo"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if len(capturedArgs) != 3 || capturedArgs[0] != "clone" || capturedArgs[2] != "/tmp/foo" {
		t.Errorf("git args = %v", capturedArgs)
	}
	if !strings.HasSuffix(capturedArgs[1], "/alice/foo.git") {
		t.Errorf("clone URL = %q", capturedArgs[1])
	}
}

func TestRepoView_RendersAndJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "r1", "owner": "alice", "name": "foo", "visibility": "public",
			"description": "hello", "clone_url": "/alice/foo.git",
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"repo", "view", "alice/foo"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "alice/foo (public)") || !strings.Contains(out, "hello") {
		t.Errorf("unexpected: %q", out)
	}
}
