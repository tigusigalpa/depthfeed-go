package depthfeed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseBodyBytes = 10 << 20 // 10 MiB

// doRequest is the core HTTP executor. It builds the request, applies
// auth/headers, sends it, captures metadata, decodes the envelope, and
// retries retryable failures per the client's policy. result receives the
// decoded `data` payload; pagination receives the pagination block (may be
// nil). requireAuth controls whether the API key is sent.
func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body any, result any, pagination **Pagination, requireAuth, allowRetry bool) (*ResponseMetadata, error) {
	return c.doRequestTo(ctx, c.baseURL, method, path, query, body, result, pagination, requireAuth, allowRetry)
}

// doRequestTo is doRequest with an explicit base URL. It keeps requests to
// endpoints outside /v3 (currently Paper Trading) independent from mutable
// client state, so a Client remains safe for concurrent use.
func (c *Client) doRequestTo(ctx context.Context, baseURL, method, path string, query url.Values, body any, result any, pagination **Pagination, requireAuth, allowRetry bool) (*ResponseMetadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if requireAuth && c.apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	// Build query string.
	endpoint := path
	if len(query) > 0 {
		if enc := query.Encode(); enc != "" {
			endpoint += "?" + enc
		}
	}

	// Marshal body once; reuse bytes across retry attempts.
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("depthfeed: marshal request body: %w", err)
		}
	}

	retry := resolveRetry(c.retry)
	if !allowRetry || !retry.enabled {
		retry.maxAttempts = 1
	}

	start := retry.clock.Now()
	var lastMeta *ResponseMetadata
	var lastErr error

	for attempt := 1; attempt <= retry.maxAttempts; attempt++ {
		if attempt > 1 {
			retryAfter := retryAfterHint(lastErr, lastMeta)
			delay := retry.delayForAttempt(attempt-2, retryAfter)
			if retry.onRetry != nil {
				retry.onRetry(attempt-1, delay, lastErr)
			}
			c.logger.Logf("depthfeed: retrying request (attempt %d) in %s after: %v", attempt-1, delay, lastErr)
			if err := sleepWithContext(ctx, retry.sleeper, delay); err != nil {
				return lastMeta, err
			}
		}

		meta, err := c.attempt(ctx, baseURL, method, endpoint, bodyBytes, requireAuth, result, pagination)
		if meta != nil {
			meta.Attempts = attempt
			meta.Duration = retry.clock.Now().Sub(start)
		}
		lastMeta, lastErr = meta, err
		if apiErr := AsAPIError(err); apiErr != nil {
			apiErr.Attempts = attempt
		}
		if err == nil {
			return meta, nil
		}
		// Decide retryability.
		if !retry.enabled {
			return meta, err
		}
		if !shouldRetry(meta, err) {
			return meta, err
		}
	}

	return lastMeta, lastErr
}

// sleepWithContext makes the built-in retry wait interruptible. Existing
// custom Sleeper implementations remain supported; they may additionally
// implement SleepContext for the same cancellation behavior.
func sleepWithContext(ctx context.Context, sleeper Sleeper, delay time.Duration) error {
	if contextSleeper, ok := sleeper.(interface {
		SleepContext(context.Context, time.Duration) error
	}); ok {
		return contextSleeper.SleepContext(ctx, delay)
	}
	if _, ok := sleeper.(SystemSleeper); ok {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
	// Preserve the original custom Sleeper contract for test doubles and
	// integrations created before context-aware sleeping was available.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	sleeper.Sleep(delay)
	return ctx.Err()
}

// shouldRetry decides whether a failed attempt should be retried.
func shouldRetry(meta *ResponseMetadata, err error) bool {
	if err == nil {
		return false
	}
	apiErr := AsAPIError(err)
	if apiErr != nil {
		// Retry only on documented retryable codes / statuses.
		if apiErr.Code != "" {
			return apiErr.Code.IsRetryable()
		}
		if apiErr.Cause != nil {
			return isRetryableNetworkError(apiErr.Cause)
		}
		return retryableHTTPStatus(apiErr.HTTPStatus)
	}
	// Network-level error with no meta.
	if meta == nil {
		return isRetryableNetworkError(err)
	}
	return retryableHTTPStatus(meta.HTTPStatus)
}

// attempt performs a single HTTP round-trip and decodes the envelope.
func (c *Client) attempt(ctx context.Context, baseURL, method, endpoint string, bodyBytes []byte, requireAuth bool, result any, pagination **Pagination) (*ResponseMetadata, error) {
	fullURL := baseURL + endpoint

	var reader io.Reader
	if len(bodyBytes) > 0 {
		reader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), fullURL, reader)
	if err != nil {
		return nil, fmt.Errorf("depthfeed: build request: %w", err)
	}

	c.applyHeaders(req, requireAuth, bodyBytes != nil)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &APIError{Cause: fmt.Errorf("depthfeed: do request: %w", err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, &APIError{Cause: fmt.Errorf("depthfeed: read response body: %w", err)}
	}
	if len(respBody) > maxResponseBodyBytes {
		return nil, &APIError{Message: "response body too large"}
	}

	meta := &ResponseMetadata{
		HTTPStatus:         resp.StatusCode,
		RequestID:          resp.Header.Get("X-Request-Id"),
		RateLimitLimit:     resp.Header.Get("X-RateLimit-Limit"),
		RateLimitRemaining: resp.Header.Get("X-RateLimit-Remaining"),
		RetryAfter:         resp.Header.Get("Retry-After"),
		RawBody:            truncateBody(respBody, 4096),
		Duration:           time.Since(start),
		Attempts:           1,
	}

	// Non-2xx: build typed APIError.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{
			HTTPStatus:         meta.HTTPStatus,
			RequestID:          meta.RequestID,
			RetryAfter:         meta.RetryAfter,
			RateLimitLimit:     meta.RateLimitLimit,
			RateLimitRemaining: meta.RateLimitRemaining,
			RawBody:            meta.RawBody,
		}
		var ee errorEnvelope
		if json.Unmarshal(respBody, &ee) == nil {
			apiErr.Code = ee.Error.Code
			apiErr.Message = ee.Error.Message
			apiErr.Details = ee.Error.Details
			meta.Meta = ee.Meta
			if meta.RequestID == "" {
				meta.RequestID = ee.Meta.RequestID
				apiErr.RequestID = ee.Meta.RequestID
			}
		} else {
			apiErr.Message = extractFallbackMessage(respBody)
		}
		meta.Meta.RequestID = meta.RequestID
		return meta, newStatusError(apiErr)
	}

	// 2xx: decode success envelope.
	var env envelope
	if len(bytes.TrimSpace(respBody)) == 0 {
		// Empty 2xx (e.g. 204) — nothing to decode.
		return meta, nil
	}
	if err := json.Unmarshal(respBody, &env); err != nil {
		return meta, fmt.Errorf("depthfeed: decode response envelope: %w", err)
	}
	meta.Meta = env.Meta
	if meta.RequestID == "" {
		meta.RequestID = env.Meta.RequestID
	}

	if result != nil && len(env.Data) > 0 {
		if err := decodeData(env.Data, result); err != nil {
			return meta, fmt.Errorf("depthfeed: decode response data: %w", err)
		}
	}
	if pagination != nil {
		*pagination = env.Pagination
	}
	return meta, nil
}

// decodeData decodes the raw `data` JSON into result. When result is a
// *json.RawMessage, the raw bytes are assigned without copying.
func decodeData(data json.RawMessage, result any) error {
	if r, ok := result.(*json.RawMessage); ok {
		*r = append((*r)[:0], data...)
		return nil
	}
	return json.Unmarshal(data, result)
}

// applyHeaders sets auth, accept, user-agent, and content-type headers.
func (c *Client) applyHeaders(req *http.Request, requireAuth, hasBody bool) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if requireAuth && c.apiKey != "" {
		switch c.authStrategy {
		case AuthHeader:
			req.Header.Set("X-API-Key", c.apiKey)
		default:
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
	}
}

// extractFallbackMessage returns a short message from a non-JSON error body.
func extractFallbackMessage(body []byte) string {
	if len(body) == 0 {
		return "unknown error"
	}
	snippet := string(body)
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	return snippet
}

// pathEscape escapes a single path segment and rejects empty values.
func pathEscape(s string) (string, error) {
	if strings.TrimSpace(s) == "" {
		return "", ErrEmptyPathSegment
	}
	return url.PathEscape(s), nil
}
