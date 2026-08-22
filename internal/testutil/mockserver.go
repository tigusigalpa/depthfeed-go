// Package testutil provides a mock HTTP server for testing the
// depthfeed-go SDK without calling the real API. It records requests and
// serves configurable canned responses.
package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

// MockServer is a configurable DepthFeed API mock. It records every
// request it receives and dispatches user-supplied handlers by path.
type MockServer struct {
	Server *httptest.Server

	mu       sync.Mutex
	requests []RecordedRequest
	// handlers maps "METHOD /path" (without query) to a handler function.
	handlers map[string]http.HandlerFunc
	// defaultHandler handles unmatched paths.
	defaultHandler http.HandlerFunc
}

// RecordedRequest is a single request received by the mock server.
type RecordedRequest struct {
	Method  string
	Path    string // path without query
	Query   string // raw query string
	Headers http.Header
	Body    []byte
}

// New creates a new MockServer with no handlers. Call Start to get a URL.
func New() *MockServer {
	m := &MockServer{
		handlers: make(map[string]http.HandlerFunc),
	}
	m.defaultHandler = func(w http.ResponseWriter, r *http.Request) {
		m.ErrorResponse(w, 404, "ENDPOINT_NOT_FOUND", "no mock handler for "+r.Method+" "+r.URL.Path)
	}
	m.Server = httptest.NewServer(http.HandlerFunc(m.serve))
	return m
}

// Start returns the server URL (host:port only, no path).
func (m *MockServer) Start() string { return m.Server.URL }

// Close stops the server.
func (m *MockServer) Close() { m.Server.Close() }

// Handle registers a handler for a METHOD + path (path without query).
func (m *MockServer) Handle(method, path string, h http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[method+" "+path] = h
}

// HandleData responds to a single-object route with a 200 envelope
// wrapping the given data object.
func (m *MockServer) HandleData(method, path string, data any) {
	m.Handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		m.DataResponse(w, data, nil)
	})
}

// HandleList responds to a collection route with a 200 envelope wrapping
// the given data array and pagination block.
func (m *MockServer) HandleList(method, path string, data any, pagination any) {
	m.Handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		m.DataResponse(w, data, pagination)
	})
}

// HandleError responds with an error envelope.
func (m *MockServer) HandleError(method, path string, status int, code, message string, details any) {
	m.Handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		m.ErrorResponse(w, status, code, message, details)
	})
}

// SetDefaultHandler sets the handler for unmatched paths.
func (m *MockServer) SetDefaultHandler(h http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultHandler = h
}

func (m *MockServer) serve(w http.ResponseWriter, r *http.Request) {
	// Read the body for recording, then restore it so handlers can read it.
	body := []byte{}
	if r.Body != nil {
		body = readAll(r.Body)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	m.mu.Lock()
	m.requests = append(m.requests, RecordedRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Query:   r.URL.RawQuery,
		Headers: r.Header.Clone(),
		Body:    body,
	})
	handler, ok := m.handlers[r.Method+" "+r.URL.Path]
	if !ok {
		handler = m.defaultHandler
	}
	m.mu.Unlock()
	handler(w, r)
}

// DataResponse writes a 200 success envelope.
func (m *MockServer) DataResponse(w http.ResponseWriter, data any, pagination any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", "req_test_123")
	w.Header().Set("X-RateLimit-Limit", "25")
	w.Header().Set("X-RateLimit-Remaining", "24")
	resp := map[string]any{
		"data": data,
		"meta": map[string]string{
			"request_id": "req_test_123",
			"timestamp":  "2026-06-07T02:27:09.532Z",
		},
	}
	if pagination != nil {
		resp["pagination"] = pagination
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// ErrorResponse writes an error envelope.
func (m *MockServer) ErrorResponse(w http.ResponseWriter, status int, code, message string, details ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", "req_test_err")
	if status == 429 {
		w.Header().Set("Retry-After", "1")
	}
	w.WriteHeader(status)
	body := map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"meta": map[string]string{
			"request_id": "req_test_err",
			"timestamp":  "2026-06-07T02:27:09.532Z",
		},
	}
	if len(details) > 0 && details[0] != nil {
		body["error"].(map[string]any)["details"] = details[0]
	}
	_ = json.NewEncoder(w).Encode(body)
}

// Requests returns a copy of all recorded requests.
func (m *MockServer) Requests() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]RecordedRequest, len(m.requests))
	copy(out, m.requests)
	return out
}

// Reset clears recorded requests.
func (m *MockServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = nil
}

// Pagination builds a pagination block.
func Pagination(next string, hasMore bool, limit, count, total int) map[string]any {
	pg := map[string]any{
		"next_cursor": next,
		"has_more":    hasMore,
		"limit":       limit,
		"count":       count,
	}
	if total > 0 {
		pg["total_count"] = total
	}
	return pg
}

func readAll(r interface{ Read(p []byte) (int, error) }) []byte {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf
}

// EnvelopeData builds a raw JSON envelope bytes for ad-hoc handlers.
func EnvelopeData(data any) []byte {
	b, _ := json.Marshal(map[string]any{
		"data": data,
		"meta": map[string]string{"request_id": "req_test", "timestamp": "2026-06-07T02:27:09.532Z"},
	})
	return b
}

// ErrorEnvelope builds raw JSON error envelope bytes.
func ErrorEnvelope(code, message string, details any) []byte {
	body := map[string]any{
		"error": map[string]any{"code": code, "message": message},
		"meta":  map[string]string{"request_id": "req_test_err", "timestamp": "2026-06-07T02:27:09.532Z"},
	}
	if details != nil {
		body["error"].(map[string]any)["details"] = details
	}
	b, _ := json.Marshal(body)
	return b
}

// Ensure fmt is used.
var _ = fmt.Sprintf
