package depthfeed

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestTimestampAndIntervalHelpers(t *testing.T) {
	fromTime := NewTimestampFromTime(time.UnixMilli(1234))
	fromMillis := NewTimestampFromMillis(-42)
	if !fromTime.IsSet() || fromTime.Millis() != 1234 || fromTime.Time().UnixMilli() != 1234 {
		t.Fatal("timestamp from time was not preserved")
	}
	if fromMillis.String() != "-42" {
		t.Fatalf("timestamp string = %q", fromMillis.String())
	}
	for _, in := range []string{"2026-06-07T02:27:09.532Z", "2026-06-07T02:27:09", "2026-06-07 02:27:09", "2026-06-07"} {
		if _, err := NewTimestamp(in); err != nil {
			t.Fatalf("NewTimestamp(%q): %v", in, err)
		}
	}
	if _, err := NewTimestamp("999999999999999999999999999"); err == nil {
		t.Fatal("expected overflowing timestamp error")
	}
	for _, interval := range []Interval{"", IntervalRaw, "RAW", "0", Interval1m} {
		_ = interval.String()
	}

	var ts Timestamp
	for _, raw := range []string{"null", `"2026-06-07T02:27:09Z"`, "12.9"} {
		if err := json.Unmarshal([]byte(raw), &ts); err != nil {
			t.Fatalf("unmarshal %s: %v", raw, err)
		}
	}
	if _, err := json.Marshal(Timestamp{}); err != nil {
		t.Fatal(err)
	}
}

func TestOptionAndErrorHelpers(t *testing.T) {
	q := url.Values{}
	start := NewTimestampFromMillis(1)
	end := NewTimestampFromMillis(2)
	(&ListOptions{Limit: 10, Cursor: "cursor", IncludeCount: true, StartTime: start, EndTime: end, Interval: Interval1m}).ApplyToValues(q)
	applyBoolFilter(q, "yes", Bool(true))
	applyBoolFilter(q, "no", Bool(false))
	applyBoolFilter(q, "skip", nil)
	applyStr(q, "name", "value")
	applyStr(q, "empty", "")
	applyInt(q, "count", 2)
	applyInt(q, "zero", 0)
	if q.Get("limit") != "10" || q.Get("no") != "false" || q.Get("interval") != "1m" {
		t.Fatalf("unexpected options: %v", q)
	}

	for _, value := range []any{float64(1), float32(2), int(3), int64(4), json.Number("5"), "6"} {
		if _, err := toFloat(value); err != nil {
			t.Fatalf("toFloat(%T): %v", value, err)
		}
	}
	if _, err := toFloat(struct{}{}); err == nil {
		t.Fatal("expected unsupported toFloat error")
	}

	for _, status := range []int{400, 401, 402, 403, 404, 429, 500, 418} {
		err := newStatusError(&APIError{HTTPStatus: status, Message: "message"})
		if AsAPIError(err) == nil {
			t.Fatalf("status %d did not keep APIError", status)
		}
	}
	apiErr := &APIError{Code: ErrCodeRateLimitBurst, Details: map[string]any{"retry_after": "2"}}
	if !apiErr.IsRateLimited() || !apiErr.IsRetryable() {
		t.Fatal("retryable rate-limit error was not recognized")
	}
	if seconds, ok := apiErr.RetryAfterSeconds(); !ok || seconds != 2 {
		t.Fatalf("RetryAfterSeconds = %v, %v", seconds, ok)
	}
	planErr := &APIError{Code: ErrCodeEndpointNotInPlan}
	if !planErr.IsPlanGated() || (&APIError{Cause: io.EOF}).Error() == "" || (&APIError{}).Error() == "" {
		t.Fatal("APIError helpers failed")
	}
}

func TestClientConfigurationOptions(t *testing.T) {
	httpClient := &http.Client{}
	logger := &recordingLogger{}
	client, err := NewClient(Config{},
		WithBaseURL("https://example.test/v3"), WithWebSocketURL("wss://example.test/stream"),
		WithHTTPClient(httpClient), WithTimeout(time.Second), WithAuthStrategy(AuthHeader),
		WithUserAgent("coverage-test"), WithRetry(RetryConfig{Enabled: true}), WithLogger(logger))
	if err != nil || client.baseURL != "https://example.test/v3" || client.httpClient != httpClient || client.authStrategy != AuthHeader || client.logger != logger {
		t.Fatalf("client options were not applied: %+v, %v", client, err)
	}
	if (SystemClock{}).Now().IsZero() {
		t.Fatal("system clock returned zero time")
	}

	oldLookup := osLookupEnv
	t.Cleanup(func() { osLookupEnv = oldLookup })
	osLookupEnv = func(key string) string {
		switch key {
		case "DEPTHFEED_API_KEY":
			return "df_env"
		case "DEPTHFEED_BASE_URL":
			return "https://env.test/v3"
		case "DEPTHFEED_WS_URL":
			return "wss://env.test/stream"
		case "DEPTHFEED_AUTH_STRATEGY":
			return string(AuthHeader)
		case "DEPTHFEED_USER_AGENT":
			return "env-agent"
		default:
			return ""
		}
	}
	fromEnv, err := NewClientFromEnv()
	if err != nil || fromEnv.apiKey != "df_env" || fromEnv.userAgent != "env-agent" || fromEnv.authStrategy != AuthHeader {
		t.Fatalf("environment configuration was not applied: %+v, %v", fromEnv, err)
	}
}

type recordingLogger struct{}

func (*recordingLogger) Logf(string, ...any) {}

func TestRetryHelpers(t *testing.T) {
	state := resolveRetry(RetryConfig{})
	if !state.clock.Now().IsZero() && (state.maxAttempts != 4 || state.initialDelay != 250*time.Millisecond) {
		t.Fatal("retry defaults were not applied")
	}
	if got := state.delayForAttempt(0, "0.001"); got != time.Millisecond {
		t.Fatalf("numeric retry-after = %v", got)
	}
	if got := state.delayForAttempt(0, time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)); got != 0 {
		t.Fatalf("expired retry-after = %v", got)
	}
	if !retryableHTTPStatus(503) || retryableHTTPStatus(501) || !isRetryableNetworkError(io.ErrUnexpectedEOF) {
		t.Fatal("retry status classification failed")
	}
	if isRetryableNetworkError(context.Canceled) || isRetryableNetworkError(errors.New("permanent")) {
		t.Fatal("permanent errors should not be retryable")
	}
	urlErr := &url.Error{Err: timeoutError{}}
	if !isRetryableNetworkError(urlErr) || !isRetryableNetworkCause(errors.New("connection reset by peer")) {
		t.Fatal("transient errors should be retryable")
	}
	if !contains("depthfeed", "feed") || contains("go", "depthfeed") || indexOf("abc", "b") != 1 || indexOf("abc", "x") != -1 {
		t.Fatal("string helpers failed")
	}
	if len(truncateBody([]byte("abcd"), 2)) != 2 || len(truncateBody([]byte("a"), 2)) != 1 {
		t.Fatal("body truncation failed")
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

var _ net.Error = timeoutError{}

func TestCustomUnmarshalersPreserveRawData(t *testing.T) {
	var venue PolymarketVenueMarket
	if err := json.Unmarshal([]byte(`{"asset_id":"a","condition_id":"c","slug":"s","base_asset":"btc","market_type":"15m","outcomes":["Up"],"event":"e"}`), &venue); err != nil || venue.AssetID != "a" || len(venue.Raw) == 0 {
		t.Fatalf("venue decode: %+v, %v", venue, err)
	}
	var prediction BinancePredictionMarket
	if err := json.Unmarshal([]byte(`{"market_id":"m","topic_id":"t","condition_id":"c","base_asset":"btc","market_type":"15m","status":"open","chain":"bsc"}`), &prediction); err != nil || prediction.MarketID != "m" {
		t.Fatalf("prediction decode: %+v, %v", prediction, err)
	}
	var drilldown ScreenerDrilldown
	if err := json.Unmarshal([]byte(`{"asset":"btc","window":"15m","books":{}}`), &drilldown); err != nil || drilldown.Asset != "btc" {
		t.Fatalf("drilldown decode: %+v, %v", drilldown, err)
	}
	var overview Overview
	var sportsOverview SportsOverview
	var game SportsGame
	var market SportsMarketSingle
	var player SportsPlayerProfile
	var odds SportsOddsLine
	var history SportsOddsHistory
	var series BinancePredictionTimeSeries
	for _, test := range []struct {
		data string
		out  any
	}{
		{`{"assets":[{}]}`, &overview}, {`{"leagues":[{}]}`, &sportsOverview},
		{`{"game_slug":"g","league":"nba"}`, &game}, {`{"condition_id":"c","outcomes":["yes"],"tokens":["t"]}`, &market},
		{`{"athlete_id":"p"}`, &player}, {`{"line":1}`, &odds}, {`{"history":[]}`, &history},
		{`{"market_id":"m","metric":"chance","points":[{"timestamp":"1","value":0.5}]}`, &series},
	} {
		if err := json.Unmarshal([]byte(test.data), test.out); err != nil {
			t.Fatalf("custom decode %s: %v", test.data, err)
		}
	}
	if game.GameSlug != "g" || market.ConditionID != "c" || player.AthleteID != "p" || series.MarketID != "m" {
		t.Fatal("custom fields were not decoded")
	}
}
