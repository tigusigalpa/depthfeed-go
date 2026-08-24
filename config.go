package depthfeed

import (
	"net/http"
	"os"
	"time"
)

// DefaultBaseURL is the default DepthFeed REST base URL.
const DefaultBaseURL = "https://api.depthfeed.com/v3"

// DefaultWebSocketURL is the default DepthFeed WebSocket URL.
const DefaultWebSocketURL = "wss://api.depthfeed.com/v3/stream"

// DefaultSSEBaseURL is the base URL for the sports odds SSE stream.
const DefaultSSEBaseURL = "https://api.depthfeed.com/v3"

// DefaultPaperBaseURL is the base URL for the Paper Trading webhook.
// Note: Paper Trading uses /paper/hook/{token}, NOT the /v3 base path.
const DefaultPaperBaseURL = "https://api.depthfeed.com"

// DefaultTimeout is the default HTTP client timeout.
const DefaultTimeout = 30 * time.Second

// DefaultUserAgent is the default User-Agent header value.
const DefaultUserAgent = "depthfeed-go/1.0.0"

// Config holds all client settings. Use NewClient to construct a client;
// the zero value is not usable directly.
type Config struct {
	// APIKey is the DepthFeed key (df_...). Required for all routes
	// except /v3/health and /v3/overview.
	APIKey string
	// BaseURL overrides the default REST base URL. Must end with /v3
	// (or the equivalent test path). Primarily useful for tests.
	BaseURL string
	// WebSocketURL overrides the default WebSocket URL.
	WebSocketURL string
	// SSEBaseURL overrides the default SSE base URL.
	SSEBaseURL string
	// PaperBaseURL overrides the default Paper Trading base URL.
	PaperBaseURL string
	// Timeout is the per-request timeout. Zero uses DefaultTimeout.
	Timeout time.Duration
	// AuthStrategy selects the auth header. Default is AuthBearer.
	AuthStrategy AuthStrategy
	// UserAgent overrides the default User-Agent.
	UserAgent string
	// HTTPClient is an injectable *http.Client. If nil, a client with
	// Timeout is created.
	HTTPClient *http.Client
	// Retry is the opt-in retry policy. Retry.Enabled must be true to
	// retry; the zero value disables retries.
	Retry RetryConfig
	// Logger receives diagnostic messages. It must redact secrets; the
	// SDK never passes the API key to it.
	Logger Logger
}

// RetryConfig is the opt-in, configurable retry policy. Retries are
// disabled by default. When Enabled, retryable error codes and statuses
// (429, 503 BARS_BUSY, UPSTREAM_UNAVAILABLE, AUTH_UNAVAILABLE, transient
// network errors, and selected 5xx) are retried with capped exponential
// backoff and jitter. Invalid requests, auth/plan/history-window
// failures, and non-idempotent calls are never retried automatically.
type RetryConfig struct {
	// Enabled enables retry. Default false.
	Enabled bool
	// MaxAttempts is the total number of attempts (including the first).
	// Default 4.
	MaxAttempts int
	// InitialDelay is the first backoff delay. Default 250ms.
	InitialDelay time.Duration
	// MaxDelay caps the backoff. Default 5s.
	MaxDelay time.Duration
	// OnRetry is called before each retry sleep with the attempt number
	// (1-based for the upcoming retry), the delay, and the last error.
	OnRetry func(attempt int, delay time.Duration, err error)
	// Clock is an injectable clock for testing. Defaults to SystemClock.
	Clock Clock
	// Sleeper is an injectable sleeper for testing. Defaults to
	// SystemSleeper.
	Sleeper Sleeper
}

// Logger is a minimal structured-ish logging interface for diagnostics.
// Implementations must redact secrets; the SDK never passes the API key.
type Logger interface {
	Logf(format string, args ...any)
}

// nopLogger discards all output.
type nopLogger struct{}

func (nopLogger) Logf(string, ...any) {}

// Clock returns the current time. Injectable for testing.
type Clock interface {
	Now() time.Time
}

// SystemClock is the real-time Clock.
type SystemClock struct{}

// Now returns time.Now().
func (SystemClock) Now() time.Time { return time.Now() }

// Sleeper sleeps for a duration. It is injectable for testing.
// The built-in sleeper is interruptible through the request context.
type Sleeper interface {
	Sleep(d time.Duration)
}

// SystemSleeper is the real Sleeper.
type SystemSleeper struct{}

// Sleep calls time.Sleep.
func (SystemSleeper) Sleep(d time.Duration) { time.Sleep(d) }

// Option configures a Client. Applied in order.
type Option func(*Config)

// WithBaseURL overrides the default REST base URL.
func WithBaseURL(u string) Option {
	return func(c *Config) {
		if u != "" {
			c.BaseURL = u
		}
	}
}

// WithWebSocketURL overrides the default WebSocket URL.
func WithWebSocketURL(u string) Option {
	return func(c *Config) {
		if u != "" {
			c.WebSocketURL = u
		}
	}
}

// WithHTTPClient sets a custom *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Config) {
		if h != nil {
			c.HTTPClient = h
		}
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.Timeout = d
		}
	}
}

// WithAuthStrategy selects the auth header (bearer or header).
func WithAuthStrategy(s AuthStrategy) Option {
	return func(c *Config) { c.AuthStrategy = s }
}

// WithUserAgent overrides the default User-Agent.
func WithUserAgent(ua string) Option {
	return func(c *Config) {
		if ua != "" {
			c.UserAgent = ua
		}
	}
}

// WithRetry enables retry with the given policy.
func WithRetry(r RetryConfig) Option {
	return func(c *Config) { c.Retry = r }
}

// WithLogger sets a diagnostic logger.
func WithLogger(l Logger) Option {
	return func(c *Config) {
		if l != nil {
			c.Logger = l
		}
	}
}

// osLookupEnv wraps os.LookupEnv for testability.
var osLookupEnv = func(key string) string {
	v, _ := os.LookupEnv(key)
	return v
}
