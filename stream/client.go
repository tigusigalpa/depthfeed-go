package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// AuthMode selects how the API key is sent at connection time.
type AuthMode string

// Documented auth modes. Server-side clients should use a header mode;
// query-string is for browsers that cannot set WebSocket headers.
const (
	// AuthBearer sends Authorization: Bearer df_... (recommended for servers).
	AuthBearer AuthMode = "bearer"
	// AuthHeader sends X-API-Key: df_...
	AuthHeader AuthMode = "header"
	// AuthQuery sends ?api_key=df_... (browsers only; never default for servers).
	AuthQuery AuthMode = "query"
)

// Config holds stream client settings.
type Config struct {
	// APIKey is the DepthFeed key (df_...).
	APIKey string
	// URL is the WebSocket URL (wss://api.depthfeed.com/v3/stream).
	URL string
	// AuthMode selects the auth method. Default AuthBearer.
	AuthMode AuthMode
	// UserAgent overrides the default User-Agent.
	UserAgent string
	// Dialer is an injectable *websocket.Dialer. If nil, a default dialer
	// with a 30s handshake timeout is used.
	Dialer *websocket.Dialer
	// Reconnect enables automatic reconnect with jittered exponential
	// backoff (250ms doubling to ~5s) on unexpected disconnect. Default
	// true. Set false to disable.
	Reconnect bool
	// InitialBackoff is the first reconnect delay. Default 250ms.
	InitialBackoff time.Duration
	// MaxBackoff caps the reconnect delay. Default 5s.
	MaxBackoff time.Duration
	// MaxReconnectAttempts caps the number of reconnect attempts (0 = unlimited).
	MaxReconnectAttempts int
	// HandshakeTimeout is the WebSocket handshake timeout. Default 30s.
	HandshakeTimeout time.Duration
}

// Option configures a stream Config.
type Option func(*Config)

// WithAuthMode sets the auth mode.
func WithAuthMode(m AuthMode) Option { return func(c *Config) { c.AuthMode = m } }

// WithUserAgent sets the User-Agent.
func WithUserAgent(ua string) Option {
	return func(c *Config) {
		if ua != "" {
			c.UserAgent = ua
		}
	}
}

// WithDialer sets a custom *websocket.Dialer.
func WithDialer(d *websocket.Dialer) Option {
	return func(c *Config) {
		if d != nil {
			c.Dialer = d
		}
	}
}

// WithReconnect enables/disables automatic reconnect.
func WithReconnect(enabled bool) Option { return func(c *Config) { c.Reconnect = enabled } }

// WithBackoff sets the reconnect backoff bounds.
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

// Event is a lifecycle event delivered to an EventHandler.
type Event struct {
	// Kind is the event kind.
	Kind EventKind
	// Err is the associated error (for error/disconnect events).
	Err error
	// Attempt is the reconnect attempt number (1-based) for reconnect events.
	Attempt int
}

// EventKind is a stream lifecycle event kind.
type EventKind int

// Documented event kinds.
const (
	EventConnected EventKind = iota
	EventDisconnected
	EventReconnecting
	EventReconnected
	EventClosed
	EventError
)

// EventHandler receives lifecycle events. It must not block the stream loop.
type EventHandler func(Event)

// Client is a DepthFeed live stream WebSocket client. It authenticates at
// connection time, manages subscriptions, decodes typed frames, and
// reconnects with jittered exponential backoff on unexpected disconnect,
// re-sending the active subscription set. It never panics or exits the
// host process. Do not reconnect after explicit Close.
//
// A Client is single-connection; for multiple connections create multiple
// clients. The client is NOT safe for concurrent Subscribe/Unsubscribe
// calls — use a single reader goroutine.
type Client struct {
	cfg Config

	mu     sync.Mutex
	conn   *websocket.Conn
	closed bool
	// active is the set of channels to re-send on reconnect.
	active map[string]struct{}

	events EventHandler
}

// NewClient creates a new stream client. The connection is NOT opened;
// call Connect to open it.
func NewClient(cfg Config, opts ...Option) *Client {
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.AuthMode == "" {
		cfg.AuthMode = AuthBearer
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 250 * time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 5 * time.Second
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 30 * time.Second
	}
	if cfg.Dialer == nil {
		cfg.Dialer = &websocket.Dialer{HandshakeTimeout: cfg.HandshakeTimeout}
	}
	return &Client{
		cfg:    cfg,
		active: make(map[string]struct{}),
	}
}

// OnEvent registers a lifecycle event handler.
func (c *Client) OnEvent(h EventHandler) { c.events = h }

func (c *Client) emit(e Event) {
	if c.events != nil {
		c.events(e)
	}
}

// Connect opens the WebSocket connection. It does NOT subscribe; call
// Subscribe afterwards. The context bounds the dial.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("stream: client closed")
	}
	c.mu.Unlock()
	return c.dial(ctx)
}

// dial opens the connection with the configured auth mode.
func (c *Client) dial(ctx context.Context) error {
	dialer := c.cfg.Dialer
	url := c.cfg.URL
	headers := http.Header{}
	headers.Set("User-Agent", c.cfg.UserAgent)
	switch c.cfg.AuthMode {
	case AuthHeader:
		headers.Set("X-API-Key", c.cfg.APIKey)
	case AuthQuery:
		// Append api_key query param (browsers only).
		sep := "?"
		if hasQuery(url) {
			sep = "&"
		}
		url = url + sep + "api_key=" + c.cfg.APIKey
	default: // AuthBearer
		headers.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	conn, _, err := dialer.DialContext(ctx, url, headers)
	if err != nil {
		return fmt.Errorf("stream: dial: %w", err)
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	c.emit(Event{Kind: EventConnected})
	return nil
}

// Subscribe subscribes to the given channels atomically. It also records
// them so they are re-sent after a reconnect.
func (c *Client) Subscribe(channels []string) error {
	if len(channels) == 0 {
		return nil
	}
	for _, ch := range channels {
		if !ValidateChannel(ch) {
			return fmt.Errorf("stream: invalid channel %q", ch)
		}
	}
	msg, err := SubscribeMessage(channels)
	if err != nil {
		return err
	}
	if err := c.write(msg); err != nil {
		return err
	}
	c.mu.Lock()
	for _, ch := range channels {
		c.active[ch] = struct{}{}
	}
	c.mu.Unlock()
	return nil
}

// Unsubscribe unsubscribes from the given channels atomically and removes
// them from the active set.
func (c *Client) Unsubscribe(channels []string) error {
	if len(channels) == 0 {
		return nil
	}
	msg, err := UnsubscribeMessage(channels)
	if err != nil {
		return err
	}
	if err := c.write(msg); err != nil {
		return err
	}
	c.mu.Lock()
	for _, ch := range channels {
		delete(c.active, ch)
	}
	c.mu.Unlock()
	return nil
}

// Ping sends a ping op.
func (c *Client) Ping() error {
	msg, err := PingMessage()
	if err != nil {
		return err
	}
	return c.write(msg)
}

// write sends a message under the lock.
func (c *Client) write(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return errors.New("stream: not connected")
	}
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

// Receive reads and decodes the next frame. It blocks until a frame is
// available, the connection closes, or an error occurs. On an unexpected
// close, if Reconnect is enabled, it reconnects (re-sending subscriptions)
// and continues; otherwise it returns the error.
func (c *Client) Receive(ctx context.Context) (*Frame, error) {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return nil, errors.New("stream: not connected")
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			if c.isClosed() {
				return nil, errors.New("stream: client closed")
			}
			// Inspect close code.
			if ce, ok := err.(*websocket.CloseError); ok {
				if isFatalClose(ce.Code) {
					c.emit(Event{Kind: EventError, Err: fmt.Errorf("stream: close %d: %s", ce.Code, ce.Text)})
					return nil, fmt.Errorf("stream: fatal close %d: %s", ce.Code, ce.Text)
				}
			}
			if !c.cfg.Reconnect {
				c.emit(Event{Kind: EventDisconnected, Err: err})
				return nil, fmt.Errorf("stream: read: %w", err)
			}
			// Reconnect with backoff.
			if rerr := c.reconnectLoop(ctx); rerr != nil {
				return nil, rerr
			}
			continue
		}
		var frame Frame
		if err := json.Unmarshal(data, &frame); err != nil {
			// Allow raw access for forwards compatibility.
			return &Frame{Op: OpError, Message: string(data)}, nil
		}
		return &frame, nil
	}
}

// ReceiveRaw reads the next raw message without decoding (for forwards
// compatibility with unknown frame shapes).
func (c *Client) ReceiveRaw(ctx context.Context) (json.RawMessage, error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return nil, errors.New("stream: not connected")
	}
	_, data, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// reconnectLoop reconnects with jittered exponential backoff, re-sending
// the active subscription set after a successful reconnect.
func (c *Client) reconnectLoop(ctx context.Context) error {
	backoff := c.cfg.InitialBackoff
	if backoff <= 0 {
		backoff = 250 * time.Millisecond
	}
	maxBackoff := c.cfg.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 5 * time.Second
	}
	for attempt := 1; ; attempt++ {
		if c.isClosed() {
			return errors.New("stream: client closed")
		}
		if c.cfg.MaxReconnectAttempts > 0 && attempt > c.cfg.MaxReconnectAttempts {
			return fmt.Errorf("stream: reconnect attempts exhausted (%d)", c.cfg.MaxReconnectAttempts)
		}
		// Jittered exponential backoff: backoff * 2^(attempt-1), capped, + jitter up to backoff.
		delay := backoff << uint(attempt-1)
		if delay <= 0 || delay > maxBackoff {
			delay = maxBackoff
		}
		jitter := time.Duration(rand.Int63n(int64(backoff) + 1))
		delay += jitter
		c.emit(Event{Kind: EventReconnecting, Attempt: attempt})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		if err := c.dial(ctx); err != nil {
			continue
		}
		// Re-send active subscriptions.
		c.mu.Lock()
		channels := make([]string, 0, len(c.active))
		for ch := range c.active {
			channels = append(channels, ch)
		}
		c.mu.Unlock()
		if len(channels) > 0 {
			msg, _ := SubscribeMessage(channels)
			_ = c.write(msg)
		}
		c.emit(Event{Kind: EventReconnected, Attempt: attempt})
		return nil
	}
}

// isFatalClose reports whether a close code should NOT trigger reconnect.
func isFatalClose(code int) bool {
	switch CloseCode(code) {
	case CloseAuthFailed, ClosePlanLimit:
		return true
	}
	return false
}

// isClosed reports whether the client was explicitly closed.
func (c *Client) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// Close closes the connection. After Close, no reconnect is attempted and
// further calls return an error.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	conn := c.conn
	c.mu.Unlock()
	c.emit(Event{Kind: EventClosed})
	if conn != nil {
		// Send a clean close frame.
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return conn.Close()
	}
	return nil
}

// ActiveChannels returns a copy of the active subscription set.
func (c *Client) ActiveChannels() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.active))
	for ch := range c.active {
		out = append(out, ch)
	}
	return out
}

func hasQuery(u string) bool {
	for i := 0; i < len(u); i++ {
		if u[i] == '?' {
			return true
		}
	}
	return false
}
