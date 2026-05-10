package api

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// StreamEvent is one SSE record from the agent's /runs/:id/stream endpoint.
type StreamEvent struct {
	ID      string
	Type    string
	Payload string // raw JSON payload from the data: line
}

// StreamRun connects to streamURL and emits every SSE record onto the
// returned channel. Closes the channel when the server closes the stream
// or ctx is cancelled.
func (c *Client) StreamRun(ctx context.Context, streamURL string) (<-chan StreamEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	httpc := &http.Client{Timeout: 0} // SSE is long-lived
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	ch := make(chan StreamEvent, 16)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		parseSSE(resp.Body, ch, ctx)
	}()
	return ch, nil
}

// parseSSE reads SSE records (id:, event:, data: lines, separated by blank
// lines) and emits them to ch. Multi-line `data:` is joined with `\n` per RFC.
func parseSSE(r io.Reader, ch chan<- StreamEvent, ctx context.Context) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var cur StreamEvent
	var dataLines []string
	flush := func() {
		if len(dataLines) > 0 {
			cur.Payload = strings.Join(dataLines, "\n")
		}
		if cur.Type == "" && cur.Payload == "" {
			cur, dataLines = StreamEvent{}, nil
			return
		}
		select {
		case ch <- cur:
		case <-ctx.Done():
		}
		cur, dataLines = StreamEvent{}, nil
	}
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := sc.Text()
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, ":") {
			// Comment / keep-alive ping; ignore.
			continue
		}
		if k, v, ok := splitField(line); ok {
			switch k {
			case "id":
				cur.ID = v
			case "event":
				cur.Type = v
			case "data":
				dataLines = append(dataLines, v)
			}
		}
	}
	// Flush trailing event if the stream ended without a blank line.
	flush()
}

func splitField(line string) (key, value string, ok bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return line, "", true
	}
	value = line[idx+1:]
	if strings.HasPrefix(value, " ") {
		value = value[1:]
	}
	return line[:idx], value, true
}
