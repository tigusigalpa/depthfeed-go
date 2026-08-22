package depthfeed

import (
	"encoding/json"
	"errors"
	"fmt"
)

// APIError is the typed error returned for a non-2xx DepthFeed response.
// It carries the HTTP status, the stable DepthFeed error code, message,
// details, request id, rate-limit headers, and a truncated raw body so
// callers can branch programmatically on the API code rather than
// string-matching human-readable messages. It never contains the API key.
//
// Use errors.Is against the sentinel errors (ErrAuthMissing, ErrNotFound,
// …) and errors.As to extract the full *APIError.
type APIError struct {
	// HTTPStatus is the HTTP status code.
	HTTPStatus int
	// Code is the stable DepthFeed error code (e.g. INVALID_COIN).
	Code ErrorCode
	// Message is the human-readable message from the error envelope.
	Message string
	// Details is the error.details object, if present. Common keys:
	//   - max_history_days, oldest_allowed (HISTORY_LIMIT_EXCEEDED)
	//   - scope ("account" | "shared_pool") (HISTORY_* / concurrency)
	//   - retry_after (seconds) (BARS_BUSY)
	Details map[string]any
	// RequestID is meta.request_id (also X-Request-Id header).
	RequestID string
	// RetryAfter is the Retry-After header value, if present.
	RetryAfter string
	// RateLimitLimit is the X-RateLimit-Limit header.
	RateLimitLimit string
	// RateLimitRemaining is the X-RateLimit-Remaining header.
	RateLimitRemaining string
	// RawBody is a truncated copy of the response body (max 4 KiB).
	RawBody []byte
	// Attempts is the number of HTTP attempts before this error.
	Attempts int
	// Cause is the underlying transport error, if the failure was
	// network-level rather than an API error envelope.
	Cause error
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("depthfeed: api error %s (status %d): %s", e.Code, e.HTTPStatus, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("depthfeed: request failed (status %d): %v", e.HTTPStatus, e.Cause)
	}
	return fmt.Sprintf("depthfeed: api error (status %d): %s", e.HTTPStatus, e.Message)
}

// Unwrap returns the underlying cause (if any) so errors.Is works for
// network errors. Sentinel unwrapping is handled by wrappedAPIError.
func (e *APIError) Unwrap() error { return e.Cause }

// IsRateLimited reports whether the error is a rate-limit or historical
// concurrency/pool error (429 codes).
func (e *APIError) IsRateLimited() bool {
	switch e.Code {
	case ErrCodeRateLimitBurst, ErrCodeRateLimitSustained,
		ErrCodeHistoryConcurrencyLimit, ErrCodeHistoryPoolBusy:
		return true
	}
	return e.HTTPStatus == 429
}

// IsRetryable reports whether the error is documented as retryable with
// backoff (rate limits, BARS_BUSY, UPSTREAM_UNAVAILABLE, AUTH_UNAVAILABLE,
// INTERNAL_ERROR, and transient network errors).
func (e *APIError) IsRetryable() bool {
	if e == nil {
		return false
	}
	if e.Code != "" {
		return e.Code.IsRetryable()
	}
	if e.Cause != nil {
		return isRetryableNetworkError(e.Cause)
	}
	return e.HTTPStatus == 429 || e.HTTPStatus >= 500
}

// IsPlanGated reports whether the error is a plan/entitlement failure
// (402 COIN_NOT_IN_PLAN / ENDPOINT_NOT_IN_PLAN / SYMBOL_LIMIT_EXCEEDED,
// 403 HISTORY_LIMIT_EXCEEDED, 402 TICKS_NOT_IN_PLAN).
func (e *APIError) IsPlanGated() bool {
	switch e.Code {
	case ErrCodeCoinNotInPlan, ErrCodeEndpointNotInPlan,
		ErrCodeSymbolLimitExceeded, ErrCodeHistoryLimitExceeded,
		ErrCodeTicksNotInPlan:
		return true
	}
	return false
}

// RetryAfterSeconds returns the retry delay in seconds, preferring the
// error.details.retry_after field (BARS_BUSY) and falling back to the
// Retry-After header. Returns 0 and ok=false when no hint is present.
func (e *APIError) RetryAfterSeconds() (secs float64, ok bool) {
	if v, present := e.Details["retry_after"]; present {
		if f, err := toFloat(v); err == nil && f > 0 {
			return f, true
		}
	}
	if e.RetryAfter != "" {
		if f, err := toFloat(e.RetryAfter); err == nil && f > 0 {
			return f, true
		}
	}
	return 0, false
}

// Sentinel errors for use with errors.Is. They correspond to the
// documented HTTP status / error-code categories.
var (
	// ErrMissingAPIKey is returned locally when a key is required but absent.
	ErrMissingAPIKey = errors.New("depthfeed: missing API key — set DEPTHFEED_API_KEY or pass a key to NewClient")
	// ErrInvalidCoin is returned locally for an unsupported {coin}.
	ErrInvalidCoin = errors.New("depthfeed: invalid coin — must be one of btc, eth, sol, xrp, doge, bnb, hype")
	// ErrInvalidVenue is returned locally for an unsupported Binance {venue}.
	ErrInvalidVenue = errors.New("depthfeed: invalid venue — must be spot or futures")
	// ErrEmptyPathSegment is returned locally for an empty {slug}/{ticker}/{id}.
	ErrEmptyPathSegment = errors.New("depthfeed: path segment must not be empty")
	// ErrBatchTooLarge is returned locally for >10 batch subrequests.
	ErrBatchTooLarge = errors.New("depthfeed: batch supports at most 10 subrequests")
	// ErrBatchInvalidPath is returned locally for a non-GET /v3/ subrequest.
	ErrBatchInvalidPath = errors.New("depthfeed: batch subrequests must be GET /v3/ paths")

	// HTTP-status sentinels (also matched by errors.Is on *APIError).
	ErrBadRequest       = errors.New("depthfeed: bad request (400)")
	ErrUnauthorized     = errors.New("depthfeed: unauthorized (401)")
	ErrPaymentRequired  = errors.New("depthfeed: payment required / not in plan (402)")
	ErrForbidden        = errors.New("depthfeed: forbidden / history window exceeded (403)")
	ErrNotFound         = errors.New("depthfeed: not found (404)")
	ErrRateLimited      = errors.New("depthfeed: rate limited (429)")
	ErrServerError      = errors.New("depthfeed: server error (5xx)")
	ErrTransportFailure = errors.New("depthfeed: transport failure")
)

// wrappedAPIError wraps *APIError with a status sentinel so both
// errors.Is (sentinel) and errors.As (*APIError) work.
type wrappedAPIError struct {
	*APIError
	sentinel error
}

// Unwrap returns the *APIError and the sentinel so errors.Is and
// errors.As both work.
func (w *wrappedAPIError) Unwrap() []error {
	return []error{w.APIError, w.sentinel}
}

// newStatusError builds a wrapped error attaching the status sentinel.
func newStatusError(apiErr *APIError) error {
	switch {
	case apiErr.HTTPStatus == 400:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrBadRequest}
	case apiErr.HTTPStatus == 401:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrUnauthorized}
	case apiErr.HTTPStatus == 402:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrPaymentRequired}
	case apiErr.HTTPStatus == 403:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrForbidden}
	case apiErr.HTTPStatus == 404:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrNotFound}
	case apiErr.HTTPStatus == 429:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrRateLimited}
	case apiErr.HTTPStatus >= 500:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrServerError}
	default:
		return apiErr
	}
}

// AsAPIError extracts *APIError from err (including wrapped errors).
// Returns nil if err is not an API error.
func AsAPIError(err error) *APIError {
	if err == nil {
		return nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

// toFloat converts a numeric any to float64.
func toFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case json.Number:
		return x.Float64()
	case string:
		return parseFloat(x)
	}
	return 0, fmt.Errorf("not numeric")
}
