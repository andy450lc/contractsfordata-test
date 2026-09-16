package session

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pixels-two/sow/backend/internal/common"
)

func TestStateRoundTrip(t *testing.T) {
	key := mustRing(t, keyA).ActiveKey()
	now := time.Now()

	state, err := IssueState(key, "/agreements/7?tab=files", now)
	if err != nil {
		t.Fatalf("issuing state: %v", err)
	}

	returnTo, err := VerifyState(key, state, now.Add(9*time.Minute))
	if err != nil {
		t.Fatalf("verifying state: %v", err)
	}
	if returnTo != "/agreements/7?tab=files" {
		t.Fatalf("returnTo = %q", returnTo)
	}
}

func TestStateRejections(t *testing.T) {
	key := mustRing(t, keyA).ActiveKey()
	otherKey := mustRing(t, keyB).ActiveKey()
	now := time.Now()

	valid, err := IssueState(key, "/dashboard", now)
	if err != nil {
		t.Fatalf("issuing state: %v", err)
	}
	tampered := []byte(valid)
	// Flip the second-to-last byte, not the last: RawURLEncoding of a
	// 32-byte HMAC ends in a base64 character with 2 unused bits, so
	// tampering it can round-trip to the exact same decoded signature
	// and the test would flake.
	tampered[len(tampered)-2] ^= 0x01
	foreign, err := IssueState(otherKey, "/dashboard", now)
	if err != nil {
		t.Fatalf("issuing foreign state: %v", err)
	}

	cases := map[string]struct {
		state string
		at    time.Time
	}{
		"empty":       {state: "", at: now},
		"no dot":      {state: strings.ReplaceAll(valid, ".", "_"), at: now},
		"tampered":    {state: string(tampered), at: now},
		"foreign key": {state: foreign, at: now},
		"expired":     {state: valid, at: now.Add(StateTTL + time.Second)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := VerifyState(key, tc.state, tc.at); !errors.Is(err, common.ErrInvalidState) {
				t.Fatalf("VerifyState error = %v, want ErrInvalidState", err)
			}
		})
	}
}

func TestStateNormalizesReturnTo(t *testing.T) {
	key := mustRing(t, keyA).ActiveKey()
	now := time.Now()

	state, err := IssueState(key, "https://evil.example", now)
	if err != nil {
		t.Fatalf("issuing state: %v", err)
	}
	returnTo, err := VerifyState(key, state, now)
	if err != nil {
		t.Fatalf("verifying state: %v", err)
	}
	if returnTo != DefaultReturnTo {
		t.Fatalf("returnTo = %q, want %q", returnTo, DefaultReturnTo)
	}
}

func TestSafeReturnTo(t *testing.T) {
	cases := map[string]string{
		"/dashboard":           "/dashboard",
		"/agreements/1?x=1":    "/agreements/1?x=1",
		"//evil.example":       DefaultReturnTo,
		"https://evil.example": DefaultReturnTo,
		"/a\\b":                DefaultReturnTo,
		"dashboard":            DefaultReturnTo,
		"":                     DefaultReturnTo,
	}
	for input, want := range cases {
		if got := SafeReturnTo(input); got != want {
			t.Errorf("SafeReturnTo(%q) = %q, want %q", input, got, want)
		}
	}
}
