package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pixels-two/sow/backend/internal/common"
)

// StateTTL bounds how long a hosted sign-in round trip may take.
const StateTTL = 10 * time.Minute

// DefaultReturnTo is where an invalid or absent return path lands.
const DefaultReturnTo = "/dashboard"

type statePayload struct {
	Nonce     string `json:"n"`
	ReturnTo  string `json:"r"`
	ExpiresAt int64  `json:"e"`
}

// IssueState signs a return path with a nonce and an expiry.
func IssueState(key []byte, returnTo string, now time.Time) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generating state nonce: %w", err)
	}

	payload, err := json.Marshal(statePayload{
		Nonce:     base64.RawURLEncoding.EncodeToString(nonce),
		ReturnTo:  SafeReturnTo(returnTo),
		ExpiresAt: now.Add(StateTTL).UnixMilli(),
	})
	if err != nil {
		return "", fmt.Errorf("encoding state: %w", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + base64.RawURLEncoding.EncodeToString(sign(key, encoded)), nil
}

// VerifyState checks the signature and expiry and returns the validated
// return path. Every failure maps to common.ErrInvalidState.
func VerifyState(key []byte, state string, now time.Time) (string, error) {
	encoded, signature, found := strings.Cut(state, ".")
	if !found {
		return "", fmt.Errorf("parsing state: %w", common.ErrInvalidState)
	}

	mac, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil || !hmac.Equal(mac, sign(key, encoded)) {
		return "", fmt.Errorf("verifying state signature: %w", common.ErrInvalidState)
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decoding state: %w", common.ErrInvalidState)
	}

	var payload statePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("parsing state payload: %w", common.ErrInvalidState)
	}
	if now.UnixMilli() > payload.ExpiresAt {
		return "", fmt.Errorf("state expired: %w", common.ErrInvalidState)
	}
	return SafeReturnTo(payload.ReturnTo), nil
}

func sign(key []byte, message string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return mac.Sum(nil)
}

// SafeReturnTo validates a post-authentication destination. Only
// same-app absolute paths pass. Anything else becomes the dashboard.
func SafeReturnTo(value string) string {
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return DefaultReturnTo
	}
	if strings.Contains(value, "\\") {
		return DefaultReturnTo
	}
	return value
}
