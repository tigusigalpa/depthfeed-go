package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/tigusigalpa/depthfeed-go/webhooks"
)

// Example: a minimal net/http receiver for DepthFeed Sports injury
// webhooks that verifies the HMAC-SHA256 signature over the exact raw
// request body before parsing JSON.
//
// Run: go run examples/verify-webhook/main.go
// Then POST to http://localhost:8080/webhook with headers:
//
//	X-DepthFeed-Signature: <hex HMAC>
//	X-DepthFeed-Event: injury
//
// And body: the JSON payload.
//
// The signing secret is read from DEPTHFEED_WEBHOOK_SECRET.
func main() {
	secret := os.Getenv("DEPTHFEED_WEBHOOK_SECRET")
	if secret == "" {
		fmt.Fprintln(os.Stderr, "Set DEPTHFEED_WEBHOOK_SECRET to your signing secret")
		os.Exit(1)
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		// Read the EXACT raw body — do not re-serialize JSON before verifying.
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		sig := r.Header.Get(webhooks.SignatureHeader)
		event := r.Header.Get(webhooks.EventHeader)

		ok, err := webhooks.VerifySportsInjury(rawBody, sig, secret)
		if err != nil {
			http.Error(w, "verification error: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		fmt.Printf("verified event=%s body=%s\n", event, string(rawBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	fmt.Println("listening on :8080 — POST to /webhook")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}
