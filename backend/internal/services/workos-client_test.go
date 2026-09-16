package services

import (
	"net/url"
	"testing"

	"github.com/workos/workos-go/v10"

	"github.com/pixels-two/sow/backend/internal/config"
)

func TestAuthorizationURLSelectsGoogle(t *testing.T) {
	t.Parallel()

	client := workos.NewClient("sk_test",
		workos.WithClientID("client_test"),
		workos.WithBaseURL("https://provider.example"),
	)
	cfg := config.AppConfig{WorkOSRedirectURI: "http://localhost:8080/v1/auth/callback"}

	raw := NewSDKWorkOSClient(client.UserManagement(), cfg).AuthorizationURL("state-value", "google")

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parsing %q: %v", raw, err)
	}
	query := parsed.Query()
	for key, want := range map[string]string{
		"provider":     "GoogleOAuth",
		"redirect_uri": "http://localhost:8080/v1/auth/callback",
		"state":        "state-value",
		"client_id":    "client_test",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if query.Has("screen_hint") {
		t.Errorf("screen_hint = %q, want the parameter absent", query.Get("screen_hint"))
	}
}
