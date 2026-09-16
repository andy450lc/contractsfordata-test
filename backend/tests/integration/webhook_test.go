package integration

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/workos/workos-go/v10"
)

const webhookSecret = "whsec_test"

// signedHeader builds a valid WorkOS-Signature header for the body.
func signedHeader(body string, signedAt time.Time) string {
	timestamp := fmt.Sprintf("%d", signedAt.UnixMilli())
	signature := workos.ComputeWebhookSignature(webhookSecret, timestamp, body)
	return fmt.Sprintf("t=%s, v1=%s", timestamp, signature)
}

func userEventBody(event string, userID string, email string, firstName string, updatedAt time.Time) string {
	created := updatedAt.Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	updated := updatedAt.UTC().Format(time.RFC3339Nano)
	return fmt.Sprintf(`{
		"id": "event_%s_%d",
		"event": %q,
		"data": {
			"object": "user",
			"id": %q,
			"email": %q,
			"first_name": %q,
			"last_name": "Hook",
			"email_verified": true,
			"created_at": %q,
			"updated_at": %q
		},
		"created_at": %q
	}`, userID, updatedAt.UnixNano(), event, userID, email, firstName, created, updated, updated)
}

func postWebhook(t *testing.T, baseURL string, body string, sigHeader string) (*http.Response, string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/webhooks/workos", strings.NewReader(body))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if sigHeader != "" {
		req.Header.Set("WorkOS-Signature", sigHeader)
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatalf("POST webhook: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	return resp, string(respBody)
}

func TestWebhookUserCreatedInsertsRow(t *testing.T) {
	app := startApp(t, nil)
	store := newUserStore(t)

	now := time.Now().UTC().Truncate(time.Millisecond)
	body := userEventBody("user.created", "user_wh_created", "created@example.com", "Webhook", now)

	resp, respBody := postWebhook(t, app.BaseURL, body, signedHeader(body, time.Now()))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", resp.StatusCode, respBody)
	}

	got, err := store.Get(t.Context(), "user_wh_created")
	if err != nil || got == nil {
		t.Fatalf("user row after user.created: %+v (err %v)", got, err)
	}
	if got.Name != "Webhook Hook" || got.Email != "created@example.com" || !got.UpdatedAt.Equal(now) {
		t.Errorf("row = %+v", got)
	}

	for _, line := range app.Logs.Lines() {
		if strings.Contains(line, "user event applied") && strings.Contains(line, "created@example.com") {
			t.Errorf("log line leaks the email: %s", line)
		}
	}
}

func TestWebhookDuplicateAndStaleDeliveries(t *testing.T) {
	app := startApp(t, nil)
	store := newUserStore(t)

	now := time.Now().UTC().Truncate(time.Millisecond)
	created := userEventBody("user.created", "user_wh_order", "order@example.com", "Original", now)
	for i := range 2 {
		resp, _ := postWebhook(t, app.BaseURL, created, signedHeader(created, time.Now()))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("delivery %d status = %d, want 200", i+1, resp.StatusCode)
		}
	}

	stale := userEventBody("user.updated", "user_wh_order", "stale@example.com", "Stale", now.Add(-time.Minute))
	if resp, _ := postWebhook(t, app.BaseURL, stale, signedHeader(stale, time.Now())); resp.StatusCode != http.StatusOK {
		t.Fatalf("stale delivery status: %d", resp.StatusCode)
	}
	got, err := store.Get(t.Context(), "user_wh_order")
	if err != nil || got == nil || got.Name != "Original Hook" || got.Email != "order@example.com" {
		t.Errorf("stale event changed the row: %+v (err %v)", got, err)
	}

	newer := userEventBody("user.updated", "user_wh_order", "renamed@example.com", "Renamed", now.Add(time.Minute))
	if resp, _ := postWebhook(t, app.BaseURL, newer, signedHeader(newer, time.Now())); resp.StatusCode != http.StatusOK {
		t.Fatalf("newer delivery status: %d", resp.StatusCode)
	}
	got, err = store.Get(t.Context(), "user_wh_order")
	if err != nil || got == nil || got.Name != "Renamed Hook" || got.Email != "renamed@example.com" {
		t.Errorf("newer event was not applied: %+v (err %v)", got, err)
	}
}

func TestWebhookRejectsBadSignatures(t *testing.T) {
	app := startApp(t, nil)
	store := newUserStore(t)

	now := time.Now().UTC().Truncate(time.Millisecond)
	body := userEventBody("user.created", "user_wh_forged", "forged@example.com", "Forged", now)

	cases := map[string]string{
		"missing":   "",
		"malformed": "nonsense",
		"tampered":  signedHeader(body+" ", time.Now()),
		"stale":     signedHeader(body, time.Now().Add(-time.Hour)),
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			resp, respBody := postWebhook(t, app.BaseURL, body, header)
			if resp.StatusCode != http.StatusUnauthorized || !strings.Contains(respBody, "unauthorized") {
				t.Errorf("status %d body %q, want 401 unauthorized", resp.StatusCode, respBody)
			}
		})
	}

	if got, _ := store.Get(t.Context(), "user_wh_forged"); got != nil {
		t.Errorf("forged delivery created a row: %+v", got)
	}
}

func TestWebhookDeleteAndUnknownEvents(t *testing.T) {
	app := startApp(t, nil)
	store := newUserStore(t)

	now := time.Now().UTC().Truncate(time.Millisecond)
	created := userEventBody("user.created", "user_wh_gone", "gone@example.com", "Gone", now)
	if resp, _ := postWebhook(t, app.BaseURL, created, signedHeader(created, time.Now())); resp.StatusCode != http.StatusOK {
		t.Fatalf("seeding status: %d", resp.StatusCode)
	}

	deleted := userEventBody("user.deleted", "user_wh_gone", "gone@example.com", "Gone", now.Add(time.Minute))
	resp, _ := postWebhook(t, app.BaseURL, deleted, signedHeader(deleted, time.Now()))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", resp.StatusCode)
	}
	if got, _ := store.Get(t.Context(), "user_wh_gone"); got != nil {
		t.Errorf("row survived user.deleted: %+v", got)
	}

	unknown := `{"id":"event_x","event":"organization.created","data":{"id":"org_1"},"created_at":"2026-09-03T00:00:00Z"}`
	resp, _ = postWebhook(t, app.BaseURL, unknown, signedHeader(unknown, time.Now()))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unknown event status = %d, want 200", resp.StatusCode)
	}

	garbage := `{"not":"an event"}`
	resp, body := postWebhook(t, app.BaseURL, garbage, signedHeader(garbage, time.Now()))
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "bad_request") {
		t.Errorf("garbage status %d body %q, want 400 bad_request", resp.StatusCode, body)
	}
}
