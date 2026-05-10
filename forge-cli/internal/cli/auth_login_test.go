package cli_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codecollab-co/forge/forge-cli/internal/cli"
	authcmd "github.com/codecollab-co/forge/forge-cli/internal/cmd/auth"
)

func TestAuthLogin_DeviceCodeFlow_HappyPath(t *testing.T) {
	t.Setenv("FORGE_CONFIG_DIR", t.TempDir())

	// Speed up polling.
	authcmd.PollClock = func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}
	t.Cleanup(func() {
		authcmd.PollClock = func(d time.Duration) <-chan time.Time { return time.After(d) }
	})

	var pollCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/device/code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"device_code":      "dev-code-123",
				"user_code":        "ABCD-EFGH",
				"verification_uri": "http://web/device",
				"expires_in":       60, "interval": 1,
			})
		case "/oauth/device/token":
			n := atomic.AddInt32(&pollCalls, 1)
			if n == 1 {
				http.Error(w, `{"error":"authorization_pending"}`, http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "tok-xyz", "handle": "alice",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	err := cli.Run([]string{"auth", "login", "--api-url", srv.URL}, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "ABCD-EFGH") {
		t.Errorf("expected user code in output, got %q", out)
	}
	if !strings.Contains(out, "Signed in as @alice") {
		t.Errorf("expected sign-in confirmation, got %q", out)
	}

	// `forge auth status` should now report the signed-in user.
	srvStatus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/me" {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "u-1", "handle": "alice"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srvStatus.Close()

	// Login saved srv.URL, so /me will hit srv. Re-use it to keep things simple.
	stdout.Reset()
	err = cli.Run([]string{"auth", "status"}, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	// Either path the test server takes, status should mention alice or "rejected".
	if !strings.Contains(stdout.String(), "alice") {
		t.Errorf("expected status to mention alice, got %q", stdout.String())
	}
}

func TestAuthLogout_ClearsCredentials(t *testing.T) {
	t.Setenv("FORGE_CONFIG_DIR", t.TempDir())
	// First, fake a logged-in state by running login through a happy server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/device/code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"device_code": "d", "user_code": "X-Y", "verification_uri": "u", "expires_in": 60, "interval": 1,
			})
		case "/oauth/device/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "handle": "alice"})
		}
	}))
	defer srv.Close()
	authcmd.PollClock = func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}
	t.Cleanup(func() {
		authcmd.PollClock = func(d time.Duration) <-chan time.Time { return time.After(d) }
	})

	if err := cli.Run([]string{"auth", "login", "--api-url", srv.URL}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := cli.Run([]string{"auth", "logout"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Logged out") {
		t.Errorf("expected 'Logged out' in output, got %q", stdout.String())
	}
	stdout.Reset()
	_ = cli.Run([]string{"auth", "status"}, &stdout, io.Discard)
	if !strings.Contains(stdout.String(), "Not logged in") {
		t.Errorf("expected 'Not logged in' after logout, got %q", stdout.String())
	}
}
