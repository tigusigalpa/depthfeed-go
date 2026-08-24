package depthfeed

import (
	"encoding/json"
	"time"
)

// Meta is the common response metadata returned on every response.
type Meta struct {
	RequestID string `json:"request_id"`
	// Timestamp is the server timestamp (ISO-8601). Kept as a string to
	// preserve the exact wire form; use MetaTime() to parse.
	Timestamp string `json:"timestamp"`
}

// MetaTime parses m.Timestamp as a time.Time. It returns the zero time
// and an error only if the timestamp is present but unparseable.
func (m Meta) MetaTime() (time.Time, error) {
	if m.Timestamp == "" {
		return time.Time{}, nil
	}
	ts, err := NewTimestamp(m.Timestamp)
	if err != nil {
		return time.Time{}, err
	}
	return ts.Time(), nil
}

// Pagination is the keyset-cursor block present only on collection routes.
// Cursors are opaque and must be sent back unchanged.
type Pagination struct {
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
	Limit      int    `json:"limit"`
	Count      int    `json:"count"`
	// TotalCount is present only when include_count=true.
	TotalCount int `json:"total_count,omitempty"`
}

// HasMorePages reports whether more pages are available. Equivalent to
// p.HasMore but safe on a nil receiver.
func (p *Pagination) HasMorePages() bool {
	if p == nil {
		return false
	}
	return p.HasMore
}

// ResponseMetadata holds non-body information from an HTTP response:
// status code, response headers, timing, and the decoded envelope meta.
// It never contains the API key.
type ResponseMetadata struct {
	// HTTPStatus is the HTTP status code.
	HTTPStatus int
	// RequestID is meta.request_id (also the X-Request-Id header).
	RequestID string
	// RateLimitLimit is the X-RateLimit-Limit header (per-second burst).
	RateLimitLimit string
	// RateLimitRemaining is the X-RateLimit-Remaining header.
	RateLimitRemaining string
	// RetryAfter is the Retry-After header value, if present.
	RetryAfter string
	// RawBody is a truncated copy of the response body (max 4 KiB).
	RawBody []byte
	// Meta is the decoded envelope meta block.
	Meta Meta
	// Attempts is the number of HTTP attempts made (including the first).
	Attempts int
	// Duration is the wall-clock time of the final HTTP round-trip.
	Duration time.Duration
}

// envelope is the internal wire shape of a successful response.
type envelope struct {
	Data       json.RawMessage `json:"data"`
	Pagination *Pagination     `json:"pagination,omitempty"`
	Meta       Meta            `json:"meta"`
}

// errorEnvelope is the internal wire shape of an error response.
type errorEnvelope struct {
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}

// ErrorBody is the error object inside the error envelope.
type ErrorBody struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
