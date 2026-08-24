package depthfeed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tigusigalpa/depthfeed-go/internal/testutil"
)

func newTestClient(t *testing.T, baseURL string, opts ...Option) *Client {
	t.Helper()
	c, err := NewClient(Config{
		APIKey:  "df_test_example_not_a_real_key",
		BaseURL: baseURL,
	}, opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestNewClientMissingKey(t *testing.T) {
	// A client can be constructed with an empty key (public routes), but
	// authenticated routes will fail server-side. We only error from env
	// when the env var is absent.
	c, err := NewClient(Config{BaseURL: "http://x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("nil client")
	}
}

func TestHealthNoAuth(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.HandleData("GET", "/health", map[string]any{"status": "ok", "service": "depthfeed", "time": "2026-01-01T00:00:00Z"})
	c := newTestClient(t, srv.Start())

	out, meta, err := c.Identity.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if out.Status != "ok" {
		t.Errorf("status = %q, want ok", out.Status)
	}
	if meta.RequestID != "req_test_123" {
		t.Errorf("request id = %q", meta.RequestID)
	}
	// Verify no auth header was sent.
	reqs := srv.Requests()
	if len(reqs) != 1 {
		t.Fatalf("recorded %d requests", len(reqs))
	}
	if h := reqs[0].Headers.Get("Authorization"); h != "" {
		t.Errorf("health should not send Authorization, got %q", h)
	}
	if h := reqs[0].Headers.Get("X-API-Key"); h != "" {
		t.Errorf("health should not send X-API-Key, got %q", h)
	}
}

func TestWhoAmI(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.HandleData("GET", "/whoami", map[string]any{
		"user_id": "acct_1", "plan": "pro", "rps": 25, "rpm": 1000,
		"history_days": 30, "coins": "all", "sports": false,
		"bars": map[string]int{"max_symbols_per_request": 8},
	})
	c := newTestClient(t, srv.Start())

	out, _, err := c.Identity.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("WhoAmI: %v", err)
	}
	if out.Plan != "pro" || out.RPS != 25 || out.Bars.MaxSymbolsPerRequest != 8 {
		t.Errorf("unexpected whoami: %+v", out)
	}
	// Verify bearer auth header was sent.
	reqs := srv.Requests()
	if got := reqs[0].Headers.Get("Authorization"); !strings.HasPrefix(got, "Bearer df_") {
		t.Errorf("expected Bearer auth, got %q", got)
	}
}

func TestAuthHeaderStrategy(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.HandleData("GET", "/whoami", map[string]any{"plan": "free"})
	c := newTestClient(t, srv.Start(), WithAuthStrategy(AuthHeader))

	_, _, err := c.Identity.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("WhoAmI: %v", err)
	}
	reqs := srv.Requests()
	if got := reqs[0].Headers.Get("X-API-Key"); got == "" {
		t.Errorf("expected X-API-Key header")
	}
	if got := reqs[0].Headers.Get("Authorization"); got != "" {
		t.Errorf("should not send Authorization with header strategy, got %q", got)
	}
}

func TestErrorEnvelopeTyped(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.HandleError("GET", "/btc/markets", 400, "INVALID_COIN", "bad coin", nil)
	c := newTestClient(t, srv.Start())

	_, _, _, err := c.Markets.List(context.Background(), "btc", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := AsAPIError(err)
	if apiErr == nil {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != ErrCodeInvalidCoin {
		t.Errorf("code = %q, want INVALID_COIN", apiErr.Code)
	}
	if apiErr.HTTPStatus != 400 {
		t.Errorf("status = %d, want 400", apiErr.HTTPStatus)
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("errors.Is(ErrBadRequest) = false")
	}
}

func TestPlanGatedError(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.HandleError("GET", "/btc/markets", 402, "COIN_NOT_IN_PLAN", "eth not in plan", nil)
	c := newTestClient(t, srv.Start())

	_, _, _, err := c.Markets.List(context.Background(), "btc", nil)
	apiErr := AsAPIError(err)
	if apiErr == nil || !apiErr.IsPlanGated() {
		t.Errorf("expected plan-gated error")
	}
	if !errors.Is(err, ErrPaymentRequired) {
		t.Errorf("errors.Is(ErrPaymentRequired) = false")
	}
}

func TestRateLimitRetryAfter(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.HandleError("GET", "/btc/markets", 429, "RATE_LIMIT_BURST", "slow down", nil)
	c := newTestClient(t, srv.Start())

	_, _, _, err := c.Markets.List(context.Background(), "btc", nil)
	apiErr := AsAPIError(err)
	if apiErr == nil || !apiErr.IsRateLimited() {
		t.Errorf("expected rate-limited error")
	}
	if apiErr.RetryAfter != "1" {
		t.Errorf("RetryAfter = %q, want 1", apiErr.RetryAfter)
	}
}

func TestBarsBusyRetryAfterDetails(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.HandleError("GET", "/bars", 503, "BARS_BUSY", "busy", map[string]any{"retry_after": 2})
	c := newTestClient(t, srv.Start())

	_, _, _, err := c.Bars.Get(context.Background(), &BarsOptions{Symbols: "BTCUSDT"})
	apiErr := AsAPIError(err)
	if apiErr == nil {
		t.Fatal("expected error")
	}
	if !apiErr.IsRetryable() {
		t.Errorf("BARS_BUSY should be retryable")
	}
	secs, ok := apiErr.RetryAfterSeconds()
	if !ok || secs != 2 {
		t.Errorf("RetryAfterSeconds = %v ok=%v, want 2 true", secs, ok)
	}
}

func TestInvalidCoinLocal(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	c := newTestClient(t, srv.Start())

	_, _, _, err := c.Markets.List(context.Background(), "xxx", nil)
	if !errors.Is(err, ErrInvalidCoin) {
		t.Errorf("expected ErrInvalidCoin, got %v", err)
	}
	if len(srv.Requests()) != 0 {
		t.Errorf("should not make a request for invalid coin")
	}
}

func TestEmptyPathSegment(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	c := newTestClient(t, srv.Start())

	_, _, err := c.Markets.Get(context.Background(), "btc", "  ")
	if !errors.Is(err, ErrEmptyPathSegment) {
		t.Errorf("expected ErrEmptyPathSegment, got %v", err)
	}
}

func TestCursorPagination(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	// Page 1
	srv.Handle("GET", "/btc/markets", func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		if cursor == "" {
			srv.DataResponse(w, []map[string]any{{"market_id": "1"}}, testutil.Pagination("cur2", true, 100, 1, 0))
			return
		}
		if cursor == "cur2" {
			srv.DataResponse(w, []map[string]any{{"market_id": "2"}}, testutil.Pagination("", false, 100, 1, 0))
			return
		}
		srv.ErrorResponse(w, 400, "INVALID_CURSOR", "bad cursor")
	})
	c := newTestClient(t, srv.Start())

	var all []Market
	opts := &MarketListOptions{ListOptions: ListOptions{Limit: 100}}
	out, pg, _, err := c.Markets.List(context.Background(), "btc", opts)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	all = append(all, out...)
	for pg != nil && pg.HasMore {
		opts.Cursor = pg.NextCursor
		out, pg, _, err = c.Markets.List(context.Background(), "btc", opts)
		if err != nil {
			t.Fatalf("page 2: %v", err)
		}
		all = append(all, out...)
	}
	if len(all) != 2 {
		t.Errorf("collected %d markets, want 2", len(all))
	}
}

func TestRetryOn429(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	calls := 0
	srv.Handle("GET", "/whoami", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			srv.ErrorResponse(w, 429, "RATE_LIMIT_BURST", "slow down")
			return
		}
		srv.DataResponse(w, map[string]any{"plan": "pro"}, nil)
	})
	c := newTestClient(t, srv.Start(), WithRetry(RetryConfig{
		Enabled:      true,
		MaxAttempts:  5,
		InitialDelay: time.Millisecond,
		MaxDelay:     5 * time.Millisecond,
	}))

	out, meta, err := c.Identity.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("WhoAmI: %v", err)
	}
	if out.Plan != "pro" {
		t.Errorf("plan = %q", out.Plan)
	}
	if meta.Attempts != 3 {
		t.Errorf("attempts = %d, want 3", meta.Attempts)
	}
}

func TestRetryDoesNotRetryPermanentError(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	calls := 0
	srv.Handle("GET", "/whoami", func(w http.ResponseWriter, r *http.Request) {
		calls++
		srv.ErrorResponse(w, 401, "AUTH_INVALID", "bad key")
	})
	c := newTestClient(t, srv.Start(), WithRetry(RetryConfig{
		Enabled:      true,
		MaxAttempts:  5,
		InitialDelay: time.Millisecond,
	}))

	_, _, err := c.Identity.WhoAmI(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("permanent error should not retry, calls = %d", calls)
	}
}

func TestRetryHonorsDetailsRetryAfter(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	calls := 0
	srv.Handle("GET", "/bars", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			srv.ErrorResponse(w, http.StatusServiceUnavailable, "BARS_BUSY", "busy", map[string]any{"retry_after": 0.001})
			return
		}
		srv.DataResponse(w, map[string]any{"series": []any{}}, nil)
	})
	c := newTestClient(t, srv.Start(), WithRetry(RetryConfig{
		Enabled: true, MaxAttempts: 2, InitialDelay: time.Second, MaxDelay: time.Second,
	}))

	start := time.Now()
	_, _, meta, err := c.Bars.Get(context.Background(), &BarsOptions{Symbols: "BTCUSDT"})
	if err != nil {
		t.Fatalf("Bars.Get: %v", err)
	}
	if meta.Attempts != 2 || calls != 2 {
		t.Fatalf("attempts=%d calls=%d, want 2", meta.Attempts, calls)
	}
	if elapsed := time.Since(start); elapsed >= 100*time.Millisecond {
		t.Fatalf("retry waited %v; details.retry_after was not honored", elapsed)
	}
}

func TestAPIErrorReportsFinalAttemptCount(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.HandleError("GET", "/whoami", http.StatusTooManyRequests, "RATE_LIMIT_BURST", "slow down", nil)
	c := newTestClient(t, srv.Start(), WithRetry(RetryConfig{
		Enabled: true, MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond,
	}))

	_, _, err := c.Identity.WhoAmI(context.Background())
	apiErr := AsAPIError(err)
	if apiErr == nil {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.Attempts != 3 {
		t.Fatalf("APIError.Attempts = %d, want 3", apiErr.Attempts)
	}
}

func TestRetryDelayNeverExceedsMaxDelay(t *testing.T) {
	retry := resolveRetry(RetryConfig{Enabled: true, InitialDelay: 10 * time.Millisecond, MaxDelay: time.Millisecond})
	for attempt := 0; attempt < 10; attempt++ {
		if delay := retry.delayForAttempt(attempt, ""); delay > time.Millisecond {
			t.Fatalf("delay for attempt %d = %v, exceeds max delay", attempt, delay)
		}
	}
}

func TestDefaultRetrySleepHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepWithContext(ctx, SystemSleeper{}, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleepWithContext error = %v, want context.Canceled", err)
	}
}

func TestMetaTimeReturnsParseError(t *testing.T) {
	_, err := (Meta{Timestamp: "not-a-timestamp"}).MetaTime()
	if err == nil {
		t.Fatal("MetaTime should return an invalid timestamp error")
	}
}

func TestAuthenticatedRouteRequiresKeyLocally(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	c, err := NewClient(Config{BaseURL: srv.Start()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, _, err = c.Identity.WhoAmI(context.Background())
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("WhoAmI error = %v, want ErrMissingAPIKey", err)
	}
	if len(srv.Requests()) != 0 {
		t.Fatal("authenticated request should not be sent without a key")
	}
}

func TestTimestampParsing(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"1780774485780", 1780774485780},
		{"2026-06-07T02:27:09Z", 1780799229000},
		{"2026-06-07T02:27:09.532Z", 1780799229532},
	}
	for _, tc := range cases {
		ts, err := NewTimestamp(tc.in)
		if err != nil {
			t.Errorf("NewTimestamp(%q): %v", tc.in, err)
			continue
		}
		if ts.Millis() != tc.want {
			t.Errorf("NewTimestamp(%q).Millis() = %d, want %d", tc.in, ts.Millis(), tc.want)
		}
	}
	if _, err := NewTimestamp("not-a-date"); err == nil {
		t.Errorf("expected error for invalid timestamp")
	}
	// Empty returns unset.
	ts, err := NewTimestamp("")
	if err != nil || ts.IsSet() {
		t.Errorf("empty timestamp should be unset")
	}
}

func TestTimestampUnmarshalNumber(t *testing.T) {
	var ts Timestamp
	if err := json.Unmarshal([]byte("1780774485780"), &ts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ts.Millis() != 1780774485780 {
		t.Errorf("millis = %d", ts.Millis())
	}
}

func TestPriceLevelMarshal(t *testing.T) {
	l := PriceLevel{Price: 0.5, Size: 304.25}
	b, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "[0.5,304.25]" {
		t.Errorf("marshal = %s, want [0.5,304.25]", b)
	}
	var got PriceLevel
	if err := json.Unmarshal([]byte("[0.51,30.0]"), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Price != 0.51 || got.Size != 30.0 {
		t.Errorf("got %+v", got)
	}
}

func TestSnapshotDynamicCoinPrice(t *testing.T) {
	raw := []byte(`{"id":"1","time":"2026-06-06 19:34:45.780","market_id":"2456720","btc_price":60606.68,"price_up":0.505,"price_down":0.495,"orderbook_up":{"bids":[[0.50,304.25]],"asks":[[0.51,30.0]]}}`)
	var s Snapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.CoinPrice != 60606.68 {
		t.Errorf("coin price = %v, want 60606.68", s.CoinPrice)
	}
	if s.PriceUp != 0.505 {
		t.Errorf("price_up = %v", s.PriceUp)
	}
	if s.OrderbookUp == nil || len(s.OrderbookUp.Bids) != 1 || s.OrderbookUp.Bids[0].Size != 304.25 {
		t.Errorf("orderbook up not decoded: %+v", s.OrderbookUp)
	}
	if len(s.Raw) == 0 {
		t.Errorf("raw should be preserved")
	}
}

func TestBatchValidation(t *testing.T) {
	// >10 subrequests
	subs := make([]BatchSubrequest, 11)
	for i := range subs {
		subs[i] = BatchSubrequest{ID: "a", Method: "GET", Path: "/v3/btc/markets"}
	}
	_, err := NewBatchRequest(subs...)
	if !errors.Is(err, ErrBatchTooLarge) {
		t.Errorf("expected ErrBatchTooLarge, got %v", err)
	}
	// non-GET
	_, err = NewBatchRequest(BatchSubrequest{ID: "a", Method: "POST", Path: "/v3/btc/markets"})
	if !errors.Is(err, ErrBatchInvalidPath) {
		t.Errorf("expected ErrBatchInvalidPath, got %v", err)
	}
	// non-/v3/ path
	_, err = NewBatchRequest(BatchSubrequest{ID: "a", Method: "GET", Path: "/foo"})
	if !errors.Is(err, ErrBatchInvalidPath) {
		t.Errorf("expected ErrBatchInvalidPath, got %v", err)
	}
}

func TestBatchRun(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.Handle("POST", "/batch", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Requests []BatchSubrequest `json:"requests"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Requests) != 2 {
			t.Errorf("expected 2 subrequests, got %d", len(body.Requests))
		}
		resps := map[string]any{
			"responses": []map[string]any{
				{"id": "a", "status": 200, "body": map[string]any{"data": []any{}}},
				{"id": "b", "status": 200, "body": map[string]any{"data": []any{}}},
			},
		}
		srv.DataResponse(w, resps, nil)
	})
	c := newTestClient(t, srv.Start())

	req, err := NewBatchRequest(
		BatchSubrequest{ID: "a", Method: "GET", Path: "/v3/btc/markets?limit=1"},
		BatchSubrequest{ID: "b", Method: "GET", Path: "/v3/bars?symbols=BTCUSDT"},
	)
	if err != nil {
		t.Fatalf("NewBatchRequest: %v", err)
	}
	out, _, err := c.Batch.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(out.Responses) != 2 {
		t.Errorf("got %d responses, want 2", len(out.Responses))
	}
}

func TestPaperSignal(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.Handle("POST", "/paper/hook/pt_test", func(w http.ResponseWriter, r *http.Request) {
		if h := r.Header.Get("Authorization"); h != "" {
			t.Errorf("paper hook should not send Authorization, got %q", h)
		}
		srv.DataResponse(w, map[string]any{"filled": true, "action": "buy", "price": 0.62, "shares": 161.29, "cost_cents": 10000, "cash_cents": 990000, "position_id": "p1"}, nil)
	})
	c := newTestClient(t, srv.Start())
	// Override paper base URL to point at the mock (which has no /v3 prefix).
	c.paperBaseURL = srv.Start()

	usd := 100.0
	fill, _, err := c.Paper.Send(context.Background(), "pt_test", &PaperSignal{
		Action: PaperBuy, Venue: PaperVenuePolymarket, Market: "1234567890", USD: &usd, ClientID: "sig-0042",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !fill.Filled || fill.Price != 0.62 {
		t.Errorf("unexpected fill: %+v", fill)
	}
}

func TestNoAPIKeyInURL(t *testing.T) {
	srv := testutil.New()
	defer srv.Close()
	srv.HandleData("GET", "/whoami", map[string]any{"plan": "free"})
	c := newTestClient(t, srv.Start())

	_, _, err := c.Identity.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("WhoAmI: %v", err)
	}
	reqs := srv.Requests()
	if strings.Contains(reqs[0].Path, "df_") || strings.Contains(reqs[0].Query, "df_") {
		t.Errorf("API key leaked into URL: path=%q query=%q", reqs[0].Path, reqs[0].Query)
	}
}
