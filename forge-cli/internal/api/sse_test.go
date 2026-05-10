package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/codecollab-co/forge/forge-cli/internal/api"
)

func TestStreamRun_ParsesEventsInOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "id: 1\nevent: tool.use\ndata: {\"name\":\"read_file\"}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, ": ping\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "id: 2\nevent: run.terminal\ndata: {\"state\":\"succeeded\"}\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch, err := api.New(srv.URL, "tok").StreamRun(ctx, srv.URL+"/runs/x/stream")
	if err != nil {
		t.Fatal(err)
	}

	var events []api.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(events), events)
	}
	if events[0].Type != "tool.use" || events[0].ID != "1" || events[0].Payload != `{"name":"read_file"}` {
		t.Errorf("event 0 = %+v", events[0])
	}
	if events[1].Type != "run.terminal" || events[1].ID != "2" {
		t.Errorf("event 1 = %+v", events[1])
	}
}

func TestStreamRun_HandlesMultiLineData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f, _ := w.(http.Flusher)
		fmt.Fprintf(w, "event: x\ndata: line1\ndata: line2\n\n")
		f.Flush()
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ch, err := api.New(srv.URL, "").StreamRun(ctx, srv.URL+"/x")
	if err != nil {
		t.Fatal(err)
	}
	ev := <-ch
	if ev.Payload != "line1\nline2" {
		t.Errorf("payload = %q", ev.Payload)
	}
}
