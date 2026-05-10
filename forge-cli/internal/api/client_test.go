package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codecollab-co/forge/forge-cli/internal/api"
)

func TestRequestDeviceCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/device/code" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"device_code":      "dev-xyz",
			"user_code":        "ABCD-EFGH",
			"verification_uri": "http://web/device",
			"expires_in":       600,
			"interval":         5,
		})
	}))
	defer srv.Close()

	c := api.New(srv.URL, "")
	got, err := c.RequestDeviceCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.DeviceCode != "dev-xyz" || got.UserCode != "ABCD-EFGH" || got.VerificationURI != "http://web/device" {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestPollDeviceToken_PendingThenApproved(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/device/token" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		calls++
		if calls < 2 {
			http.Error(w, `{"error":"authorization_pending"}`, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok-1",
			"handle":       "alice",
		})
	}))
	defer srv.Close()

	c := api.New(srv.URL, "")
	// First call: pending.
	_, err := c.PollDeviceToken(context.Background(), "dev-xyz")
	if !errors.Is(err, api.ErrAuthorizationPending) {
		t.Fatalf("expected ErrAuthorizationPending, got %v", err)
	}
	// Second call: success.
	tok, err := c.PollDeviceToken(context.Background(), "dev-xyz")
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "tok-1" || tok.Handle != "alice" {
		t.Errorf("unexpected token: %+v", tok)
	}
}

func TestListRepos(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "r1", "owner": "alice", "name": "first", "visibility": "public", "clone_url": "/alice/first.git"},
			{"id": "r2", "owner": "alice", "name": "second", "visibility": "private", "clone_url": "/alice/second.git"},
		})
	}))
	defer srv.Close()
	repos, err := api.New(srv.URL, "tok").ListRepos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 || repos[0].Name != "first" || repos[1].Visibility != "private" {
		t.Fatalf("unexpected: %+v", repos)
	}
}

func TestGetRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/alice/foo" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "r1", "owner": "alice", "name": "foo", "visibility": "public",
			"description": "hi", "clone_url": "/alice/foo.git",
		})
	}))
	defer srv.Close()
	r, err := api.New(srv.URL, "tok").GetRepo(context.Background(), "alice", "foo")
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "foo" || r.Description != "hi" {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestCreateRepo_SendsBody(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "r1", "owner": "alice", "name": got["name"], "visibility": got["visibility"],
			"clone_url": "/alice/" + got["name"].(string) + ".git",
		})
	}))
	defer srv.Close()
	r, err := api.New(srv.URL, "tok").CreateRepo(context.Background(), api.CreateRepoInput{
		Name: "newone", Description: "d", Visibility: "private", InitReadme: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "newone" || got["init_readme"] != true || got["visibility"] != "private" {
		t.Errorf("unexpected: %+v body=%+v", r, got)
	}
}

func TestDeleteRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/alice/foo" || r.Method != http.MethodDelete {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	if err := api.New(srv.URL, "tok").DeleteRepo(context.Background(), "alice", "foo"); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/alice/foo" || r.Method != http.MethodPatch {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "r1", "owner": "alice", "name": "foo", "visibility": "private",
			"description": "new", "clone_url": "/alice/foo.git",
		})
	}))
	defer srv.Close()
	desc, vis := "new", "private"
	r, err := api.New(srv.URL, "tok").UpdateRepo(context.Background(), "alice", "foo",
		api.UpdateRepoInput{Description: &desc, Visibility: &vis})
	if err != nil {
		t.Fatal(err)
	}
	if r.Visibility != "private" || r.Description != "new" {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestMe_SendsBearerHeader(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "u-1", "handle": "alice", "email": "a@b", "display_name": "Alice",
		})
	}))
	defer srv.Close()

	c := api.New(srv.URL, "tok-99")
	me, err := c.Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if me.Handle != "alice" {
		t.Errorf("unexpected me: %+v", me)
	}
	if !strings.HasPrefix(seen, "Bearer ") || !strings.Contains(seen, "tok-99") {
		t.Errorf("Authorization header = %q, want Bearer tok-99", seen)
	}
}
