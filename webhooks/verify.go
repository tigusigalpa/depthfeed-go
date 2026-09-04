// Package webhooks provides server-side verification helpers for
// DepthFeed Sports injury webhook deliveries.
//
// Verification calculates HMAC-SHA256 over the EXACT raw HTTP request
// body and compares it against the X-DepthFeed-Signature header with a
// constant-time comparison. Do NOT deserialize and reserialize the JSON
// before verification — verify the raw bytes first.
//
// The SDK does not create an externally reachable webhook receiver by
// itself; these are helpers only. See examples/verify-webhook for a
// net/http receiver pattern.
//
// Docs: https://depthfeed.com/docs#ep-sports
package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// SignatureHeader is the header carrying the HMAC-SHA256 signature.
const SignatureHeader = "X-DepthFeed-Signature"

// EventHeader is the header carrying the event type metadata.
const EventHeader = "X-DepthFeed-Event"

// ErrEmptySignature is returned when the signature header is empty.
var ErrEmptySignature = errors.New("webhooks: empty signature")

// VerifySportsInjury verifies a Sports injury webhook delivery. It
// computes HMAC-SHA256 over the exact raw request body using the signing
// secret and compares it against the provided signature with a
// constant-time comparison (hmac.Equal).
//
// rawBody must be the exact bytes received on the wire — do not
// re-serialize JSON before verifying. The signature may be hex-encoded
// (the default DepthFeed format) or prefixed with "sha256=".
//
// Returns true when the signature matches, false (with nil error) on a
// mismatch, and an error only when the signature is empty/malformed.
func VerifySportsInjury(rawBody []byte, signature, secret string) (bool, error) {
	if secret == "" {
		return false, errors.New("webhooks: empty secret")
	}
	sig := strings.TrimSpace(signature)
	if sig == "" {
		return false, ErrEmptySignature
	}
	// Strip an optional "sha256=" prefix.
	sig = strings.TrimPrefix(sig, "sha256=")
	expected, err := hex.DecodeString(sig)
	if err != nil {
		return false, errors.New("webhooks: signature is not valid hex")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	computed := mac.Sum(nil)
	return hmac.Equal(computed, expected), nil
}

// VerifyPaperFill verifies a Paper Trading fill callback. The signature is
// HMAC-SHA256 over "timestamp.body" using the signing secret, compared
// with a constant-time comparison. The timestamp is sent in the
// x-depthfeed-timestamp header.
func VerifyPaperFill(rawBody []byte, timestamp, signature, secret string) (bool, error) {
	if secret == "" {
		return false, errors.New("webhooks: empty secret")
	}
	sig := strings.TrimSpace(signature)
	if sig == "" {
		return false, ErrEmptySignature
	}
	sig = strings.TrimPrefix(sig, "sha256=")
	expected, err := hex.DecodeString(sig)
	if err != nil {
		return false, errors.New("webhooks: signature is not valid hex")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write(rawBody)
	computed := mac.Sum(nil)
	return hmac.Equal(computed, expected), nil
}
