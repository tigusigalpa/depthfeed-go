package testutil

import (
	"io"
	"net/http"
	"testing"
)

func TestMockServerHelpers(t *testing.T) {
	m := New()
	defer m.Close()
	m.HandleData(http.MethodGet, "/data", map[string]string{"ok": "yes"})
	m.HandleList(http.MethodGet, "/list", []any{}, Pagination("next", true, 10, 1, 2))
	m.HandleError(http.MethodGet, "/error", http.StatusBadRequest, "BAD", "bad", map[string]any{"field": "x"})
	for _, path := range []string{"/data", "/list", "/error", "/missing"} {
		resp, err := http.Get(m.Start() + path)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
	}
	if len(m.Requests()) != 4 {
		t.Fatalf("requests = %d", len(m.Requests()))
	}
	m.Reset()
	if len(m.Requests()) != 0 {
		t.Fatal("Reset did not clear requests")
	}
	m.SetDefaultHandler(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	resp, err := http.Get(m.Start() + "/custom")
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("custom handler: %v, %v", resp, err)
	}
	_ = resp.Body.Close()
	if len(EnvelopeData(map[string]any{"x": 1})) == 0 || len(ErrorEnvelope("BAD", "bad", nil)) == 0 || len(ErrorEnvelope("BAD", "bad", map[string]any{})) == 0 {
		t.Fatal("envelope helpers returned no data")
	}
}
