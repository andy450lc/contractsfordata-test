package session

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	keyA = "k1:A9Keki18RmRaac8WO5FQeo9czkzVdL2hFHzp/aDm4IQ="
	keyB = "k2:Yy0M1kq2GdH1o3S4fJ1i2mHq9j0oI7eZg4C0bWcVzNA="
)

func mustRing(t *testing.T, spec string) *Keyring {
	t.Helper()
	ring, err := ParseKeyring(spec)
	if err != nil {
		t.Fatalf("parsing keyring %q: %v", spec, err)
	}
	return ring
}

func TestSealAndOpenRoundTrip(t *testing.T) {
	ring := mustRing(t, keyA)
	payload := Payload{RefreshToken: "rt_1", SessionID: "sess_1", UserID: "user_1", IssuedAt: 1756250000000}

	sealed, err := ring.Seal(payload)
	if err != nil {
		t.Fatalf("sealing: %v", err)
	}
	if !strings.HasPrefix(sealed, "v1.k1.") {
		t.Fatalf("sealed value = %q, want v1.k1. prefix", sealed)
	}
	if strings.Contains(sealed, "rt_1") {
		t.Fatal("sealed value leaks the refresh token")
	}

	opened, err := ring.Open(sealed)
	if err != nil {
		t.Fatalf("opening: %v", err)
	}
	if opened != payload {
		t.Fatalf("opened = %+v, want %+v", opened, payload)
	}
}

func TestOpenRejectsBadValues(t *testing.T) {
	ring := mustRing(t, keyA)
	sealed, err := ring.Seal(Payload{RefreshToken: "rt_1"})
	if err != nil {
		t.Fatalf("sealing: %v", err)
	}

	tampered := []byte(sealed)
	tampered[len(tampered)-1] ^= 0x01

	other := mustRing(t, keyB)
	sealedByOther, err := other.Seal(Payload{RefreshToken: "rt_2"})
	if err != nil {
		t.Fatalf("sealing with other key: %v", err)
	}

	cases := map[string]string{
		"empty":         "",
		"wrong version": "v0" + sealed[2:],
		"missing parts": "v1.k1",
		"unknown kid":   sealedByOther,
		"tampered":      string(tampered),
		"not base64":    "v1.k1.***",
		"short payload": "v1.k1.AAAA",
		"empty refresh": mustSeal(t, ring, Payload{}),
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ring.Open(value); !errors.Is(err, ErrInvalidCookie) {
				t.Fatalf("Open(%q) error = %v, want ErrInvalidCookie", value, err)
			}
		})
	}
}

func mustSeal(t *testing.T, ring *Keyring, payload Payload) string {
	t.Helper()
	sealed, err := ring.Seal(payload)
	if err != nil {
		t.Fatalf("sealing: %v", err)
	}
	return sealed
}

func TestKeyRotationOpensOldCookies(t *testing.T) {
	old := mustRing(t, keyA)
	sealed := mustSeal(t, old, Payload{RefreshToken: "rt_old"})

	rotated := mustRing(t, keyB+","+keyA)
	opened, err := rotated.Open(sealed)
	if err != nil {
		t.Fatalf("rotated ring failed to open old cookie: %v", err)
	}
	if opened.RefreshToken != "rt_old" {
		t.Fatalf("refresh token = %q, want rt_old", opened.RefreshToken)
	}

	fresh := mustSeal(t, rotated, Payload{RefreshToken: "rt_new"})
	if !strings.HasPrefix(fresh, "v1.k2.") {
		t.Fatalf("new cookies must use the first key, got %q", fresh)
	}
	if _, err := old.Open(fresh); !errors.Is(err, ErrInvalidCookie) {
		t.Fatalf("old ring opened a cookie sealed with an unknown key: %v", err)
	}
}

func TestParseKeyringRejectsMalformedSpecs(t *testing.T) {
	cases := map[string]string{
		"no colon":      "k1",
		"empty id":      ":A9Keki18RmRaac8WO5FQeo9czkzVdL2hFHzp/aDm4IQ=",
		"dot in id":     "k.1:A9Keki18RmRaac8WO5FQeo9czkzVdL2hFHzp/aDm4IQ=",
		"not base64":    "k1:***",
		"short key":     "k1:c2hvcnQ=",
		"duplicate ids": keyA + "," + keyA,
		"empty":         "",
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseKeyring(spec); !errors.Is(err, ErrInvalidKeyring) {
				t.Fatalf("ParseKeyring(%q) error = %v, want ErrInvalidKeyring", spec, err)
			}
		})
	}
}

func TestCookieAttributes(t *testing.T) {
	cookie := NewCookie("value", 7*24*time.Hour, true)
	if cookie.Name != "sow_session" || cookie.Path != CookiePath || !cookie.HttpOnly || !cookie.Secure {
		t.Fatalf("cookie attributes = %+v", cookie)
	}
	if cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge != 604800 {
		t.Fatalf("cookie SameSite/MaxAge = %v/%d", cookie.SameSite, cookie.MaxAge)
	}

	clear := ClearCookie(false)
	if clear.MaxAge >= 0 || clear.Value != "" || clear.Secure {
		t.Fatalf("clear cookie = %+v", clear)
	}
	if !strings.Contains(clear.String(), "Max-Age=0") {
		t.Fatalf("clear cookie header = %q, want Max-Age=0", clear.String())
	}
}
