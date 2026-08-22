package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func signHex(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func signHexPaper(secret string, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySportsInjury(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"league":"nba","event":"injury","status":"out"}`)
	sig := signHex(secret, body)

	ok, err := VerifySportsInjury(body, sig, secret)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Errorf("expected ok=true")
	}

	// Tampered body.
	ok, _ = VerifySportsInjury([]byte(`{"league":"nba","event":"tampered"}`), sig, secret)
	if ok {
		t.Errorf("tampered body should not verify")
	}

	// Wrong secret.
	ok, _ = VerifySportsInjury(body, sig, "wrong")
	if ok {
		t.Errorf("wrong secret should not verify")
	}
}

func TestVerifySportsInjurySha256Prefix(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"x":1}`)
	sig := "sha256=" + signHex(secret, body)
	ok, err := VerifySportsInjury(body, sig, secret)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Errorf("expected ok=true with sha256= prefix")
	}
}

func TestVerifySportsInjuryEmptySig(t *testing.T) {
	_, err := VerifySportsInjury([]byte("x"), "", "secret")
	if err != ErrEmptySignature {
		t.Errorf("expected ErrEmptySignature, got %v", err)
	}
}

func TestVerifySportsInjuryBadHex(t *testing.T) {
	_, err := VerifySportsInjury([]byte("x"), "not-hex!", "secret")
	if err == nil || !strings.Contains(err.Error(), "not valid hex") {
		t.Errorf("expected hex error, got %v", err)
	}
}

func TestVerifySportsInjuryEmptySecret(t *testing.T) {
	_, err := VerifySportsInjury([]byte("x"), "abcd", "")
	if err == nil || !strings.Contains(err.Error(), "empty secret") {
		t.Errorf("expected empty secret error, got %v", err)
	}
}

func TestVerifyPaperFill(t *testing.T) {
	secret := "whsec_paper"
	ts := "1780774485780"
	body := []byte(`{"filled":true,"price":0.62}`)
	sig := signHexPaper(secret, ts, body)

	ok, err := VerifyPaperFill(body, ts, sig, secret)
	if err != nil {
		t.Fatalf("VerifyPaperFill: %v", err)
	}
	if !ok {
		t.Errorf("expected ok=true")
	}

	// Tampered timestamp.
	ok, _ = VerifyPaperFill(body, "1780774485781", sig, secret)
	if ok {
		t.Errorf("tampered timestamp should not verify")
	}
}
