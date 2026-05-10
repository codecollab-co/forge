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
	"github.com/codecollab-co/forge/forge-cli/internal/cli"
	browsecmd "github.com/codecollab-co/forge/forge-cli/internal/cmd/browse"
)

// stubLoggedInWithWebsite writes credentials including a websiteURL.
func stubLoggedInWithWebsite(t *testing.T, apiURL, websiteURL string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("FORGE_CONFIG_DIR", dir)
	body, _ := json.Marshal(auth.Credentials{
		APIURL: apiURL, WebsiteURL: websiteURL, Token: "tok-test", Handle: "alice",
	})
	if err := os.WriteFile(filepath.Join(dir, "credentials.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBrowse_NoBrowserPrintsURL(t *testing.T) {
	stubLoggedInWithWebsite(t, "http://api", "http://web")
	var stdout bytes.Buffer
	if err := cli.Run([]string{"browse", "alice/foo", "--no-browser"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "http://web/alice/foo" {
		t.Errorf("URL = %q", got)
	}
}

func TestBrowse_IssueAndPRPaths(t *testing.T) {
	stubLoggedInWithWebsite(t, "http://api", "http://web")
	var stdout bytes.Buffer
	if err := cli.Run([]string{"browse", "alice/foo", "--issue", "12", "--no-browser"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "/alice/foo/issues/12") {
		t.Errorf("issue URL = %q", stdout.String())
	}
	stdout.Reset()
	if err := cli.Run([]string{"browse", "alice/foo", "--pr", "5", "--no-browser"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "/alice/foo/pulls/5") {
		t.Errorf("PR URL = %q", stdout.String())
	}
}

func TestBrowse_OpenerCalledWhenNotNoBrowser(t *testing.T) {
	stubLoggedInWithWebsite(t, "http://api", "http://web")
	var opened string
	browsecmd.Opener = func(url string) error {
		opened = url
		return nil
	}
	t.Cleanup(func() {
		browsecmd.Opener = func(url string) error { return nil }
	})
	if err := cli.Run([]string{"browse", "alice/foo"}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if opened != "http://web/alice/foo" {
		t.Errorf("opener got %q", opened)
	}
}

func TestAPI_GetEchoesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Errorf("missing Authorization header")
		}
		_, _ = w.Write([]byte(`[{"id":"r1","name":"foo"}]`))
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)
	var stdout bytes.Buffer
	if err := cli.Run([]string{"api", "/repos"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"name":"foo"`) {
		t.Errorf("output = %q", stdout.String())
	}
}

func TestAPI_PostWithFields(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)
	if err := cli.Run([]string{
		"api", "/repos", "-X", "POST",
		"-F", "name=demo", "-F", "init_readme=true",
	}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "demo" || got["init_readme"] != true {
		t.Errorf("body = %+v", got)
	}
}

func TestAPI_NonZeroOn4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"nope"}`, http.StatusBadRequest)
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)
	err := cli.Run([]string{"api", "/anything"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("expected HTTP 400 error, got %v", err)
	}
}
