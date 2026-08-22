// Package sse implements a client for the DepthFeed sports odds
// Server-Sent Events stream (GET /v3/sports/odds/stream with
// Accept: text/event-stream). It parses SSE fields correctly (event, id,
// multiline data, comments, and reconnect hints), respects context
// cancellation, and provides an opt-in reconnection policy. It does not
// buffer an unbounded stream in memory.
//
// The package is self-contained and does not import the root depthfeed
// package.
//
// Docs: https://depthfeed.com/docs#ep-sports
package sse

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Event is a parsed SSE event.
type Event struct {
	// Event is the event: field value (empty if not set).
	Event string
	// Data is the joined data: field value (multiline data is joined with
	// "\n" per the SSE spec).
	Data string
	// ID is the id: field value (empty if not set).
	ID string
	// Retry is the retry: hint in milliseconds (0 if not set).
	Retry int
}

// HasData reports whether the event carries any data.
func (e *Event) HasData() bool { return e.Data != "" }

// Config holds SSE client settings.
type Config struct {
	// URL is the full SSE endpoint URL.
	URL string
	// APIKey is the DepthFeed key (df_...).
	APIKey string
	// AuthStrategy is "bearer" (default) or "header".
	AuthStrategy string
	// UserAgent overrides the default User-Agent.
	UserAgent string
	// HTTPClient is an injectable *http.Client. If nil, a client with no
	// timeout (long-lived stream) is used.
	HTTPClient *http.Client
	// Reconnect enables automatic reconnect. Default false.
	Reconnect bool
	// InitialBackoff is the first reconnect delay. Default 1s.
	InitialBackoff time.Duration
	// MaxBackoff caps the reconnect delay. Default 30s.
	MaxBackoff time.Duration
	// MaxReconnectAttempts caps reconnect attempts (0 = unlimited).
	MaxReconnectAttempts int
}

// Option configures a Config.
type Option func(*Config)

// WithHTTPClient sets a custom *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Config) {
		if h != nil {
			c.HTTPClient = h
		}
	}
}

// WithReconnect enables automatic reconnect.
func WithReconnect(enabled bool) Option { return func(c *Config) { c.Reconnect = enabled } }

// WithBackoff sets reconnect backoff bounds.
func WithBackoff(initial, max time.Duration) Option {
	return func(c *Config) {
		if initial > 0 {
			c.InitialBackoff = initial
		}
		if max > 0 {
			c.MaxBackoff = max
		}
	}
}

// WithMaxReconnectAttempts caps reconnect attempts.
func WithMaxReconnectAttempts(n int) Option { return func(c *Config) { c.MaxReconnectAttempts = n } }

// Client is a DepthFeed sports odds SSE client.
type Client struct {
	cfg    Config
	mu     sync.Mutex
	closed bool
	cancel context.CancelFunc
	lastID string
}

// NewClient creates a new SSE client. The stream is NOT opened; call Stream.
func NewClient(cfg Config, opts ...Option) *Client {
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.AuthStrategy == "" {
		cfg.AuthStrategy = "bearer"
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 1 * time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{} // no timeout: long-lived stream
	}
	return &Client{cfg: cfg}
}

// Stream opens the SSE connection and calls onEvent for each parsed event.
// It blocks until the context is canceled, the server closes the stream,
// or an error occurs. With Reconnect enabled, it reconnects on EOF/error.
// The caller should run this in a goroutine.
func (c *Client) Stream(ctx context.Context, onEvent func(Event)) error {
	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()
	defer cancel()

	for attempt := 0; ; attempt++ {
		if c.isClosed() {
			return nil
		}
		err := c.streamOnce(ctx, onEvent)
		if c.isClosed() || ctx.Err() != nil {
			return nil
		}
		if err == nil {
			return nil
		}
		if !c.cfg.Reconnect {
			return err
		}
		if c.cfg.MaxReconnectAttempts > 0 && attempt >= c.cfg.MaxReconnectAttempts {
			return fmt.Errorf("sse: reconnect attempts exhausted: %w", err)
		}
		delay := c.backoff(attempt)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
}

// streamOnce opens one connection and streams events until EOF/error.
func (c *Client) streamOnce(ctx context.Context, onEvent func(Event)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL, nil)
	if err != nil {
		return fmt.Errorf("sse: build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	if c.cfg.AuthStrategy == "header" {
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	if c.lastID != "" {
		req.Header.Set("Last-Event-ID", c.lastID)
	}

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("sse: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sse: unexpected status %d", resp.StatusCode)
	}

	return parseSSE(resp.Body, onEvent, &c.lastID)
}

// parseSSE reads SSE events from r, calling onEvent for each complete
// event (blank line dispatch). Comments (lines starting with ':') are
// ignored. Multiline data: fields are joined with "\n".
func parseSSE(r io.Reader, onEvent func(Event), lastID *string) error {
	scanner := bufio.NewScanner(r)
	// Allow long lines.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var ev Event
	var dataLines []string

	dispatch := func() {
		if len(dataLines) > 0 {
			ev.Data = strings.Join(dataLines, "\n")
		}
		if ev.HasData() || ev.Event != "" || ev.ID != "" {
			onEvent(ev)
		}
		if ev.ID != "" {
			*lastID = ev.ID
		}
		ev = Event{}
		dataLines = dataLines[:0]
	}

	for scanner.Scan() {
		line := scanner.Text()
		// Strip a single trailing \r (CRLF).
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			dispatch()
			continue
		}
		if line[0] == ':' {
			continue // comment
		}
		field, value, _ := strings.Cut(line, ":")
		if strings.HasPrefix(value, " ") {
			value = value[1:]
		}
		switch field {
		case "event":
			ev.Event = value
		case "data":
			dataLines = append(dataLines, value)
		case "id":
			ev.ID = value
		case "retry":
			if n := parseIntSafe(value); n > 0 {
				ev.Retry = n
			}
		}
	}
	dispatch()
	if err := scanner.Err(); err != nil {
		return err
	}
	return io.EOF
}

func (c *Client) backoff(attempt int) time.Duration {
	d := c.cfg.InitialBackoff << uint(attempt)
	if d <= 0 || d > c.cfg.MaxBackoff {
		d = c.cfg.MaxBackoff
	}
	return d
}

func (c *Client) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// Close cancels the active stream. Safe to call multiple times.
func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func parseIntSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
