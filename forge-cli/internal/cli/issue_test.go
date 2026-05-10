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

func TestIssueList_OpenByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/issues") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("expected state=open, got %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "i1", "number": 1, "title": "first", "state": "open", "author": "alice"},
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"issue", "list", "alice/foo"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "#1") || !strings.Contains(stdout.String(), "first") {
		t.Errorf("unexpected: %q", stdout.String())
	}
}

func TestIssueList_AllSkipsStateQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query for --state all, got %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)
	if err := cli.Run([]string{"issue", "list", "alice/foo", "--state", "all"}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
}

func TestIssueCreate_PassesTitleAndBody(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "i1", "number": 7, "title": got["title"], "state": "open", "author": "alice",
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"issue", "create", "alice/foo", "--title", "Bug X", "--body", "Steps:\n1. ..."}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got["title"] != "Bug X" || !strings.Contains(got["body"].(string), "Steps:") {
		t.Errorf("body = %+v", got)
	}
	if !strings.Contains(stdout.String(), "Opened #7") {
		t.Errorf("output = %q", stdout.String())
	}
}

func TestIssueCreate_MissingTitleErrors(t *testing.T) {
	stubLoggedIn(t, "http://unused")
	err := cli.Run([]string{"issue", "create", "alice/foo"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("expected title-required error, got %v", err)
	}
}

func TestIssueClose(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/issues/4/close") {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "i4", "number": 4, "state": "closed",
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"issue", "close", "alice/foo", "4"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "#4 is now closed") {
		t.Errorf("output = %q", stdout.String())
	}
}

func TestIssueAssignAgent_PrintsRunIDAndWatchHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/assign-agent") {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "run-abc", "state": "queued",
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"issue", "assign-agent", "alice/foo", "12"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "run-abc") || !strings.Contains(stdout.String(), "forge run watch") {
		t.Errorf("output = %q", stdout.String())
	}
}

func TestIssueView_RendersBodyAndComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issue": map[string]any{
				"id": "i1", "number": 3, "title": "Refactor", "state": "open",
				"author": "alice", "body": "Body of issue.",
			},
			"comments": []map[string]any{
				{"id": "c1", "author": "bob", "body": "+1"},
			},
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"issue", "view", "alice/foo", "3"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "#3 Refactor") || !strings.Contains(out, "Body of issue") || !strings.Contains(out, "@bob") {
		t.Errorf("unexpected: %q", out)
	}
}
