// Package session seals the refresh session into a cookie and signs the
// state that travels through the hosted sign-in round trip.
package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pixels-two/sow/backend/internal/common"
)

const (
	// CookiePath scopes the session cookie to the session endpoints.
	CookiePath = "/v1/auth"

	cookieVersion = "v1"
	keyLength     = 32
)

var (
	// ErrInvalidCookie marks a cookie value that cannot be opened.
	ErrInvalidCookie = errors.New("invalid session cookie")
	// ErrInvalidKeyring marks a malformed SESSION_COOKIE_KEY value.
	ErrInvalidKeyring = errors.New("invalid session cookie key")
)

// Payload is the plaintext sealed into the session cookie.
type Payload struct {
	RefreshToken string `json:"rt"`
	SessionID    string `json:"sid"`
	UserID       string `json:"uid"`
	IssuedAt     int64  `json:"iat"`
}

// Keyring holds the cookie keys. The first configured key seals. Every
// key opens.
type Keyring struct {
	activeID string
	keys     map[string][]byte
}

// ParseKeyring reads a comma-separated list of kid:base64 entries.
func ParseKeyring(spec string) (*Keyring, error) {
	ring := &Keyring{keys: map[string][]byte{}}

	for i, entry := range strings.Split(spec, ",") {
		id, key, err := parseKeyEntry(strings.TrimSpace(entry))
		if err != nil {
			return nil, err
		}
		if _, dup := ring.keys[id]; dup {
			return nil, fmt.Errorf("duplicate key id %q: %w", id, ErrInvalidKeyring)
		}

		ring.keys[id] = key
		if i == 0 {
			ring.activeID = id
		}
	}
	return ring, nil
}

func parseKeyEntry(entry string) (string, []byte, error) {
	id, encoded, found := strings.Cut(entry, ":")
	if !found || id == "" || strings.Contains(id, ".") {
		return "", nil, fmt.Errorf("entry needs the form kid:base64: %w", ErrInvalidKeyring)
	}

	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, fmt.Errorf("decoding key %q: %w", id, ErrInvalidKeyring)
	}
	if len(key) != keyLength {
		return "", nil, fmt.Errorf("key %q must be %d bytes: %w", id, keyLength, ErrInvalidKeyring)
	}
	return id, key, nil
}

// ActiveKey returns the key that seals new cookies and signs state.
func (k *Keyring) ActiveKey() []byte {
	return k.keys[k.activeID]
}

// Seal encrypts the payload with the active key. The value carries the
// version, the key id, and the nonce so any configured key can open it.
func (k *Keyring) Seal(p Payload) (string, error) {
	plaintext, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("encoding session payload: %w", err)
	}

	gcm, err := newGCM(k.ActiveKey())
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	encoded := base64.RawURLEncoding.EncodeToString(sealed)
	return cookieVersion + "." + k.activeID + "." + encoded, nil
}

// Open decrypts a cookie value sealed by any configured key.
func (k *Keyring) Open(value string) (Payload, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 || parts[0] != cookieVersion {
		return Payload{}, ErrInvalidCookie
	}

	key, ok := k.keys[parts[1]]
	if !ok {
		return Payload{}, ErrInvalidCookie
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Payload{}, ErrInvalidCookie
	}

	gcm, err := newGCM(key)
	if err != nil {
		return Payload{}, err
	}
	size := gcm.NonceSize()
	if len(raw) < size {
		return Payload{}, ErrInvalidCookie
	}

	plaintext, err := gcm.Open(nil, raw[:size], raw[size:], nil)
	if err != nil {
		return Payload{}, ErrInvalidCookie
	}

	var payload Payload
	if err := json.Unmarshal(plaintext, &payload); err != nil || payload.RefreshToken == "" {
		return Payload{}, ErrInvalidCookie
	}
	return payload, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating gcm: %w", err)
	}
	return gcm, nil
}

// NewCookie builds the session cookie with the mandated attributes.
func NewCookie(value string, maxAge time.Duration, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     common.SessionCookieName,
		Value:    value,
		Path:     CookiePath,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}

// ClearCookie builds the cookie that removes the session cookie.
func ClearCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     common.SessionCookieName,
		Value:    "",
		Path:     CookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}
