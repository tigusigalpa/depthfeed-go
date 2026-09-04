package depthfeed

import (
	"context"
	"net/http"
	"net/url"

	"github.com/tigusigalpa/depthfeed-go/stream"
)

// Client is the entry point of the DepthFeed Go SDK. It holds HTTP
// configuration and exposes one service struct per API resource family.
// A Client is safe for concurrent use by multiple goroutines.
//
// Create one with NewClient (or NewClientFromEnv) and customize it with
// Option values:
//
//	client, err := depthfeed.NewClient(depthfeed.Config{
//	    APIKey: os.Getenv("DEPTHFEED_API_KEY"),
//	    Retry:  depthfeed.RetryConfig{Enabled: true},
//	})
type Client struct {
	apiKey       string
	baseURL      string
	webSocketURL string
	sseBaseURL   string
	paperBaseURL string
	authStrategy AuthStrategy
	userAgent    string
	httpClient   *http.Client
	retry        RetryConfig
	logger       Logger

	// Identity provides access to health, whoami, and overview.
	Identity *IdentityService
	// Markets provides access to Polymarket market discovery.
	Markets *MarketsService
	// Snapshots provides access to Polymarket order-book snapshots.
	Snapshots *SnapshotsService
	// Ticks provides access to Polymarket and Kalshi tick tapes.
	Ticks *TicksService
	// Binance provides access to Binance spot & futures books and trades.
	Binance *BinanceService
	// Bars provides access to /v3/bars OHLCV price bars.
	Bars *BarsService
	// Kalshi provides access to Kalshi markets, books, snapshots, candles.
	Kalshi *KalshiService
	// Limitless provides access to Limitless markets, books, snapshots.
	Limitless *LimitlessService
	// BinancePrediction provides access to Binance prediction markets.
	BinancePrediction *BinancePredictionService
	// Screener provides access to the cross-venue screener.
	Screener *ScreenerService
	// Batch provides access to POST /v3/batch.
	Batch *BatchService
	// Sports provides access to the sports REST surface.
	Sports *SportsService
	// SportsWebhooks provides access to sports injury webhook management.
	SportsWebhooks *SportsWebhooksService
	// Paper provides access to the Paper Trading signal webhook.
	Paper *PaperService
}

// NewClient creates a new DepthFeed client from the given Config. Unset
// fields fall back to safe defaults. A client may be created without an API
// key to call public routes; authenticated operations return ErrMissingAPIKey
// locally until a key is supplied.
func NewClient(cfg Config, opts ...Option) (*Client, error) {
	for _, opt := range opts {
		opt(&cfg)
	}
	c := &Client{
		apiKey:       cfg.APIKey,
		baseURL:      cfg.BaseURL,
		webSocketURL: cfg.WebSocketURL,
		sseBaseURL:   cfg.SSEBaseURL,
		paperBaseURL: cfg.PaperBaseURL,
		authStrategy: cfg.AuthStrategy,
		userAgent:    cfg.UserAgent,
		httpClient:   cfg.HTTPClient,
		retry:        cfg.Retry,
		logger:       cfg.Logger,
	}
	if c.baseURL == "" {
		c.baseURL = DefaultBaseURL
	}
	if c.webSocketURL == "" {
		c.webSocketURL = DefaultWebSocketURL
	}
	if c.sseBaseURL == "" {
		c.sseBaseURL = DefaultSSEBaseURL
	}
	if c.paperBaseURL == "" {
		c.paperBaseURL = DefaultPaperBaseURL
	}
	if c.authStrategy == "" {
		c.authStrategy = AuthBearer
	}
	if c.userAgent == "" {
		c.userAgent = DefaultUserAgent
	}
	if c.httpClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = DefaultTimeout
		}
		c.httpClient = &http.Client{Timeout: timeout}
	}
	if c.logger == nil {
		c.logger = nopLogger{}
	}
	c.initServices()
	return c, nil
}

// NewClientFromEnv reads DEPTHFEED_API_KEY from the environment and
// returns a new client. Other env vars (DEPTHFEED_BASE_URL,
// DEPTHFEED_AUTH_STRATEGY) are honored when set.
func NewClientFromEnv(opts ...Option) (*Client, error) {
	cfg := Config{
		APIKey:       osLookupEnv("DEPTHFEED_API_KEY"),
		BaseURL:      osLookupEnv("DEPTHFEED_BASE_URL"),
		WebSocketURL: osLookupEnv("DEPTHFEED_WS_URL"),
	}
	if s := osLookupEnv("DEPTHFEED_AUTH_STRATEGY"); s != "" {
		cfg.AuthStrategy = AuthStrategy(s)
	}
	if s := osLookupEnv("DEPTHFEED_USER_AGENT"); s != "" {
		cfg.UserAgent = s
	}
	return NewClient(cfg, opts...)
}

// initServices wires all service structs to the client.
func (c *Client) initServices() {
	c.Identity = &IdentityService{client: c}
	c.Markets = &MarketsService{client: c}
	c.Snapshots = &SnapshotsService{client: c}
	c.Ticks = &TicksService{client: c}
	c.Binance = &BinanceService{client: c}
	c.Bars = &BarsService{client: c}
	c.Kalshi = &KalshiService{client: c}
	c.Limitless = &LimitlessService{client: c}
	c.BinancePrediction = &BinancePredictionService{client: c}
	c.Screener = &ScreenerService{client: c}
	c.Batch = &BatchService{client: c}
	c.Sports = &SportsService{client: c}
	c.SportsWebhooks = &SportsWebhooksService{client: c}
	c.Paper = &PaperService{client: c}
}

// --- low-level helpers used by services ---

// get performs an authenticated GET and decodes data + pagination.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any, pg **Pagination) (*ResponseMetadata, error) {
	return c.doRequest(ctx, http.MethodGet, path, query, nil, out, pg, true, true)
}

// getNoAuth performs a public GET (no API key) — for /health and /overview.
func (c *Client) getNoAuth(ctx context.Context, path string, query url.Values, out any) (*ResponseMetadata, error) {
	return c.doRequest(ctx, http.MethodGet, path, query, nil, out, nil, false, true)
}

// getSingle performs an authenticated GET expecting a single-object data
// payload (no pagination block).
func (c *Client) getSingle(ctx context.Context, path string, query url.Values, out any) (*ResponseMetadata, error) {
	return c.doRequest(ctx, http.MethodGet, path, query, nil, out, nil, true, true)
}

// post performs an authenticated POST with a JSON body. No automatic retry
// (non-idempotent by default).
func (c *Client) post(ctx context.Context, path string, body any, out any) (*ResponseMetadata, error) {
	return c.doRequest(ctx, http.MethodPost, path, nil, body, out, nil, true, false)
}

// postRetryable performs an authenticated POST eligible for retry (only
// for endpoints documented as retryable, e.g. /v3/batch which contains
// only GET subrequests).
func (c *Client) postRetryable(ctx context.Context, path string, body any, out any) (*ResponseMetadata, error) {
	return c.doRequest(ctx, http.MethodPost, path, nil, body, out, nil, true, true)
}

// postNoAuth performs a POST without an API key (Paper Trading uses a
// token in the path). Caller controls retry via allowRetry=false.
func (c *Client) postNoAuth(ctx context.Context, baseURL, path string, body any, out any) (*ResponseMetadata, error) {
	return c.doRequestTo(ctx, baseURL, http.MethodPost, path, nil, body, out, nil, false, false)
}

// delete performs an authenticated DELETE. No automatic retry.
func (c *Client) delete(ctx context.Context, path string, out any) (*ResponseMetadata, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil, out, nil, true, false)
}

// paperURL returns the Paper Trading base URL.
func (c *Client) paperURL() string { return c.paperBaseURL }

// NewStreamClient builds a live-stream WebSocket client wired to this
// client's API key and WebSocket URL. The connection is not opened; call
// Connect on the returned client. The stream subpackage is self-contained
// and decodes frame data as json.RawMessage so you can decode it into
// depthfeed types (e.g. depthfeed.Snapshot) in your own code.
func (c *Client) NewStreamClient(opts ...stream.Option) *stream.Client {
	authMode := stream.AuthBearer
	if c.authStrategy == AuthHeader {
		authMode = stream.AuthHeader
	}
	return stream.NewClient(stream.Config{
		APIKey:    c.apiKey,
		URL:       c.webSocketURL,
		AuthMode:  authMode,
		UserAgent: c.userAgent,
		Reconnect: true,
	}, opts...)
}
