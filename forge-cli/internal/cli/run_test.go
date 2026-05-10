package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codecollab-co/forge/forge-cli/internal/cli"
)

func TestRunList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/runs" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "00000001-aaaa", "state": "running",
				"issue_number": 5, "issue_title": "Bug", "repo_owner": "alice", "repo_name": "foo"},
		})
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"run", "list"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "00000001") || !strings.Contains(stdout.String(), "alice/foo#5") {
		t.Errorf("unexpected: %q", stdout.String())
	}
}

func TestRunCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/runs/abc/cancel" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cancel_requested":true}`))
	}))
	defer srv.Close()
	stubLoggedIn(t, srv.URL)
	var stdout bytes.Buffer
	if err := cli.Run([]string{"run", "cancel", "abc"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Cancellation requested") {
		t.Errorf("output = %q", stdout.String())
	}
}

func TestRunWatch_StreamsThenStopsOnTerminal(t *testing.T) {
	// SSE server (separate origin from the platform — mirrors real deploy).
	sse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/runs/r1/stream" {
			t.Errorf("stream path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		f, _ := w.(http.Flusher)
		fmt.Fprintf(w, "id: 1\nevent: tool.use\ndata: {\"name\":\"read_file\"}\n\n")
		f.Flush()
		fmt.Fprintf(w, "id: 2\nevent: run.terminal\ndata: {\"state\":\"succeeded\"}\n\n")
		f.Flush()
	}))
	defer sse.Close()

	// Platform server returns the run with a stream_url pointing at the SSE server.
	plat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/runs/r1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "r1", "state": "running",
			"stream_url": sse.URL + "/runs/r1/stream",
		})
	}))
	defer plat.Close()
	stubLoggedIn(t, plat.URL)

	var stdout bytes.Buffer
	if err := cli.Run([]string{"run", "watch", "r1"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "tool.use") || !strings.Contains(out, "run.terminal") {
		t.Errorf("expected both events in output, got %q", out)
	}
	if !strings.Contains(out, "Watching r1") {
		t.Errorf("expected header line, got %q", out)
	}
}
