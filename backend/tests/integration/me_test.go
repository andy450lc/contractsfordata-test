package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pixels-two/sow/backend/internal/models"
)

type accessTokenBody struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

func decodeAccessToken(t *testing.T, body string) accessTokenBody {
	t.Helper()

	var token accessTokenBody
	if err := json.Unmarshal([]byte(body), &token); err != nil {
		t.Fatalf("decoding access token body %q: %v", body, err)
	}
	if token.AccessToken == "" {
		t.Fatalf("access token body %q has no token", body)
	}
	return token
}

func getWithAuth(t *testing.T, target string, authorization string) (*http.Response, string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	return resp, string(body)
}

func TestMeRejectsInvalidCredentials(t *testing.T) {
	fake := newFakeWorkOS(t)
	other := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)

	cases := map[string]string{
		"missing header":    "",
		"not bearer":        "Basic abc123",
		"malformed token":   "Bearer not-a-jwt",
		"expired token":     "Bearer " + fake.MintAccessToken(t, "user_expired", "s", -time.Hour),
		"wrong signing key": "Bearer " + other.MintAccessToken(t, "user_wrong_key", "s", time.Hour),
		"missing subject":   "Bearer " + fake.MintAccessToken(t, "", "s", time.Hour),
	}

	for name, authorization := range cases {
		t.Run(name, func(t *testing.T) {
			resp, body := getWithAuth(t, app.BaseURL+"/v1/me", authorization)

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", resp.StatusCode)
			}
			if !strings.Contains(body, `"error":"unauthorized"`) || strings.Contains(body, "email") {
				t.Errorf("body = %q, want a bare unauthorized envelope", body)
			}
		})
	}
}

func TestMeReturnsStoredRecord(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)

	now := time.Now().UTC().Truncate(time.Millisecond)
	store := newUserStore(t)
	if _, err := store.UpsertIfNewer(t.Context(), models.User{
		ID: "user_stored", Email: "stored@example.com", Name: "Stored User",
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seeding user: %v", err)
	}

	token := fake.MintAccessToken(t, "user_stored", "session_stored", time.Hour)
	resp, body := getWithAuth(t, app.BaseURL+"/v1/me", "Bearer "+token)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	var got struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		UpdatedAt int64  `json:"updated_at"`
	}
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if got.ID != "user_stored" || got.Email != "stored@example.com" || got.Name != "Stored User" || got.UpdatedAt != now.UnixMilli() {
		t.Errorf("body = %+v", got)
	}

	found := false
	for _, line := range app.Logs.Lines() {
		if strings.Contains(line, `"user_id":"user_stored"`) {
			found = true
		}
	}
	if !found {
		t.Error("no log line carries the authenticated user id")
	}
}

func TestMeProvisionsMissingRecordFromProvider(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)

	late := fakeUser{
		ID: "user_late", Email: "late@example.com", FirstName: "Late", LastName: "Arrival",
		CreatedAt: time.Now().Add(-time.Hour).Truncate(time.Millisecond),
		UpdatedAt: time.Now().Truncate(time.Millisecond),
	}
	fake.AddUser(late)

	token := fake.MintAccessToken(t, late.ID, "session_late", time.Hour)
	resp, body := getWithAuth(t, app.BaseURL+"/v1/me", "Bearer "+token)

	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "late@example.com") {
		t.Fatalf("status %d body %s, want 200 with the provisioned user", resp.StatusCode, body)
	}
	stored, err := newUserStore(t).Get(t.Context(), late.ID)
	if err != nil || stored == nil || stored.Name != "Late Arrival" {
		t.Errorf("stored user = %+v (err %v)", stored, err)
	}

	unknown := fake.MintAccessToken(t, "user_ghost", "session_ghost", time.Hour)
	resp, body = getWithAuth(t, app.BaseURL+"/v1/me", "Bearer "+unknown)
	if resp.StatusCode != http.StatusNotFound || !strings.Contains(body, "user_not_found") {
		t.Errorf("unknown user: status %d body %s, want 404 user_not_found", resp.StatusCode, body)
	}
}
