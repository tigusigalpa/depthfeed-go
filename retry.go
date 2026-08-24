package depthfeed

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// retryState holds resolved retry configuration with safe defaults.
type retryState struct {
	enabled      bool
	maxAttempts  int
	initialDelay time.Duration
	maxDelay     time.Duration
	onRetry      func(int, time.Duration, error)
	clock        Clock
	sleeper      Sleeper
}

func resolveRetry(r RetryConfig) retryState {
	s := retryState{
		enabled:      r.Enabled,
		maxAttempts:  r.MaxAttempts,
		initialDelay: r.InitialDelay,
		maxDelay:     r.MaxDelay,
		onRetry:      r.OnRetry,
		clock:        r.Clock,
		sleeper:      r.Sleeper,
	}
	if s.maxAttempts < 1 {
		s.maxAttempts = 4
	}
	if s.initialDelay <= 0 {
		s.initialDelay = 250 * time.Millisecond
	}
	if s.maxDelay <= 0 {
		s.maxDelay = 5 * time.Second
	}
	if s.clock == nil {
		s.clock = SystemClock{}
	}
	if s.sleeper == nil {
		s.sleeper = SystemSleeper{}
	}
	return s
}

// delayForAttempt returns a jittered exponential backoff delay for the
// given (0-based) retry index, capped at maxDelay. It honors a Retry-After
// hint (seconds or HTTP-date) when provided.
func (s retryState) delayForAttempt(attempt int, retryAfter string) time.Duration {
	if retryAfter = strings.TrimSpace(retryAfter); retryAfter != "" {
		if secs, err := strconv.ParseFloat(retryAfter, 64); err == nil && secs >= 0 {
			d := time.Duration(secs * float64(time.Second))
			if d > s.maxDelay {
				d = s.maxDelay
			}
			return d
		}
		if retryAt, err := http.ParseTime(retryAfter); err == nil {
			if d := time.Until(retryAt); d > 0 {
				if d > s.maxDelay {
					d = s.maxDelay
				}
				return d
			}
			return 0
		}
	}
	// Exponential backoff: initial * 2^attempt, capped, plus jitter up to initial.
	backoff := s.initialDelay << uint(attempt)
	if backoff <= 0 || backoff > s.maxDelay {
		backoff = s.maxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(s.initialDelay) + 1))
	delay := backoff + jitter
	if delay > s.maxDelay {
		return s.maxDelay
	}
	return delay
}

// retryAfterHint selects the API-provided retry hint. error.details.retry_after
// takes precedence over the HTTP Retry-After header, as documented for
// BARS_BUSY responses.
func retryAfterHint(err error, meta *ResponseMetadata) string {
	if apiErr := AsAPIError(err); apiErr != nil {
		if secs, ok := apiErr.RetryAfterSeconds(); ok {
			return strconv.FormatFloat(secs, 'f', -1, 64)
		}
	}
	if meta != nil {
		return meta.RetryAfter
	}
	return ""
}

// retryableHTTPStatus reports whether a bare HTTP status (no parsed code)
// should be retried.
func retryableHTTPStatus(status int) bool {
	return status == 429 || status == 500 || status == 502 || status == 503 || status == 504
}

// isRetryableNetworkError reports whether a transport-level error is
// transient enough to retry.
func isRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		return isRetryableNetworkCause(urlErr.Err)
	}
	return isRetryableNetworkCause(err)
}

func isRetryableNetworkCause(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// Connection refused / reset are transient for retries.
	var msg string
	if err != nil {
		msg = err.Error()
	}
	if contains(msg, "connection refused") || contains(msg, "connection reset") ||
		contains(msg, "broken pipe") || contains(msg, "no such host") {
		return true
	}
	return false
}

// parseFloat parses a string as float64.
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// truncateBody limits a raw body to n bytes.
func truncateBody(body []byte, n int) []byte {
	if len(body) > n {
		return body[:n]
	}
	return body
}

// contains is a dependency-free substring check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
