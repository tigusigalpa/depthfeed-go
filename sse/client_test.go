package sse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseSSE(t *testing.T) {
	// A multi-event SSE stream with comments, multiline data, id, retry.
	body := strings.NewReader(
		": comment line\n" +
			"event: odds\n" +
			"data: {\"line\":\"a\"}\n" +
			"id: 1\n" +
			"\n" +
			"event: odds\n" +
			"data: {\"line\":\"b\",\n" +
			"data:  \"more\":true}\n" +
			"id: 2\n" +
			"\n" +
			"retry: 1000\n" +
			"\n",
	)
	var events []Event
	var lastID string
	var retryDelay time.Duration
	err := parseSSE(body, func(e Event) { events = append(events, e) }, &lastID, func(delay time.Duration) {
		retryDelay = delay
	})
	if err == nil {
		t.Fatalf("expected EOF, got nil")
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Event != "odds" || events[0].ID != "1" {
		t.Errorf("event 0: %+v", events[0])
	}
	if events[0].Data != `{"line":"a"}` {
		t.Errorf("event 0 data = %q", events[0].Data)
	}
	if events[1].Data != `{"line":"b",`+"\n"+` "more":true}` {
		t.Errorf("event 1 data = %q", events[1].Data)
	}
	if lastID != "2" {
		t.Errorf("lastID = %q, want 2", lastID)
	}
	if retryDelay != time.Second {
		t.Errorf("retry delay = %v, want 1s", retryDelay)
	}
}

func TestParseSSEEmptyIDResetsLastEventID(t *testing.T) {
	lastID := "previous"
	err := parseSSE(strings.NewReader("id:\n\n"), nil, &lastID, nil)
	if err != io.EOF {
		t.Fatalf("parseSSE error = %v, want EOF", err)
	}
	if lastID != "" {
		t.Errorf("lastID = %q, want empty", lastID)
	}
}

func TestServerRetryHintControlsReconnectDelay(t *testing.T) {
	c := NewClient(Config{InitialBackoff: time.Second, MaxBackoff: 10 * time.Second})
	c.setRetryDelay(1500 * time.Millisecond)
	if got := c.backoff(4); got != 1500*time.Millisecond {
		t.Errorf("backoff = %v, want 1.5s server retry hint", got)
	}
}

func TestSSEStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer df_") {
			t.Errorf("expected Bearer auth, got %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("event: odds\ndata: {\"x\":1}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("event: odds\ndata: {\"x\":2}\n\n"))
		flusher.Flush()
	}))
	defer srv.Close()

	c := NewClient(Config{
		APIKey: "df_test_example_not_a_real_key",
		URL:    srv.URL + "/v3/sports/odds/stream",
	})
	var events []Event
	done := make(chan error, 1)
	go func() {
		done <- c.Stream(context.Background(), func(e Event) {
			events = append(events, e)
			if len(events) == 2 {
				c.Close()
			}
		})
	}()
	select {
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for events")
	case <-done:
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
}

func TestSSEAuthHeaderMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got == "" {
			t.Errorf("expected X-API-Key header")
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("should not send Authorization with header mode, got %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("data: {}\n\n"))
	}))
	defer srv.Close()

	c := NewClient(Config{
		APIKey:       "df_test",
		URL:          srv.URL,
		AuthStrategy: "header",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = c.Stream(ctx, func(e Event) { cancel() })
}
