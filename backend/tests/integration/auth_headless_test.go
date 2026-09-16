package integration

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// allowedOriginHeaders is the header set a first-party form submission
// carries.
var allowedOriginHeaders = map[string]string{"Origin": allowedOrigin}

// account builds a provider user with a password. Verified accounts
// sign in directly. Unverified accounts answer with a pending token.
func account(id string, email string, password string, verified bool) fakeUser {
	return fakeUser{
		ID:            id,
		Email:         email,
		FirstName:     "Mad",
		LastName:      "Hatter",
		Password:      password,
		EmailVerified: verified,
		CreatedAt:     time.Now().Add(-48 * time.Hour).Truncate(time.Millisecond),
		UpdatedAt:     time.Now().Add(-24 * time.Hour).Truncate(time.Millisecond),
	}
}

// postJSON posts a JSON body and returns the response with its body.
func (b *browser) postJSON(t *testing.T, target string, headers map[string]string, body string) (*http.Response, string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, target, strings.NewReader(body))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", target, err)
	}
	defer resp.Body.Close()

	read, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	return resp, string(read)
}

// tokenAlphabet is the character class an opaque provider token draws
// from.
const tokenAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

// opaqueToken reports whether token is non-empty and drawn entirely
// from tokenAlphabet.
func opaqueToken(token string) bool {
	outside := func(r rune) bool { return !strings.ContainsRune(tokenAlphabet, r) }
	return token != "" && strings.IndexFunc(token, outside) < 0
}

// pendingToken reads the pending token out of a 202 body.
func pendingToken(t *testing.T, body string) string {
	t.Helper()

	var pending struct {
		PendingToken string `json:"pending_token"`
	}
	if err := json.Unmarshal([]byte(body), &pending); err != nil {
		t.Fatalf("decoding pending body %q: %v", body, err)
	}
	if pending.PendingToken == "" {
		t.Fatalf("pending body %q carries no token", body)
	}
	return pending.PendingToken
}

func TestSignInStartsSession(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_verified", "verified@example.com", "correct horse", true))
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	before := time.Now()
	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"verified@example.com","password":"correct horse"}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	token := decodeAccessToken(t, body)
	if token.ExpiresAt < before.Add(4*time.Minute).UnixMilli() || token.ExpiresAt > before.Add(6*time.Minute).UnixMilli() {
		t.Errorf("expires_at = %d, want about five minutes out", token.ExpiresAt)
	}
	if strings.Contains(body, "correct horse") || strings.Contains(body, "rt_") {
		t.Errorf("sign-in response leaks a secret: %s", body)
	}

	header := setCookieHeader(resp)
	for _, want := range []string{"sow_session=v1.k1.", "Path=/v1/auth", "HttpOnly", "SameSite=Strict", "Max-Age=604800"} {
		if !strings.Contains(header, want) {
			t.Errorf("Set-Cookie %q lacks %q", header, want)
		}
	}

	cookie := b.sessionCookie(t, app.BaseURL)
	if cookie == nil {
		t.Fatal("jar holds no session cookie after sign-in")
	}
	payload, err := testKeyring(t).Open(cookie.Value)
	if err != nil {
		t.Fatalf("opening cookie: %v", err)
	}
	if payload.UserID != "user_verified" || payload.RefreshToken == "" {
		t.Errorf("cookie payload = %+v", payload)
	}

	resp, body = b.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", allowedOriginHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("refresh after sign-in status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
}

func TestSignInRefusesIdentically(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_known", "known@example.com", "correct horse", true))
	app := startAuthApp(t, fake, nil)

	cases := []struct {
		name string
		body string
	}{
		{name: "wrong password", body: `{"email":"known@example.com","password":"wrong password"}`},
		{name: "unknown email", body: `{"email":"stranger@example.com","password":"correct horse"}`},
	}

	var bodies []string
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := newBrowser(t)
			resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders, tc.body)

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body %s)", resp.StatusCode, body)
			}
			if !strings.Contains(body, `"error":"invalid_credentials"`) {
				t.Errorf("body = %q, want invalid_credentials", body)
			}
			if setCookieHeader(resp) != "" || b.sessionCookie(t, app.BaseURL) != nil {
				t.Error("a refused sign-in set a session cookie")
			}
			bodies = append(bodies, body)
		})
	}

	if len(bodies) == 2 && bodies[0] != bodies[1] {
		t.Errorf("refusals differ: %q vs %q", bodies[0], bodies[1])
	}
}

func TestSignInUnverifiedPending(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_unverified", "unverified@example.com", "correct horse", false))
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"unverified@example.com","password":"correct horse"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}
	if token := pendingToken(t, body); !strings.HasPrefix(token, "pat_") {
		t.Errorf("pending_token = %q, want the provider's token", token)
	}
	if setCookieHeader(resp) != "" || b.sessionCookie(t, app.BaseURL) != nil {
		t.Error("a pending sign-in set a session cookie")
	}
	if stored, _ := newUserStore(t).Get(t.Context(), "user_unverified"); stored != nil {
		t.Errorf("a pending sign-in provisioned a user: %+v", stored)
	}
}

func TestSignInValidation(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)

	cases := []struct {
		name      string
		body      string
		wantField string
	}{
		{name: "malformed email", body: `{"email":"not-an-address","password":"correct horse"}`, wantField: "email"},
		{name: "short password", body: `{"email":"someone@example.com","password":"seven77"}`, wantField: "password"},
		{name: "long password", body: `{"email":"someone@example.com","password":"` + strings.Repeat("a", 257) + `"}`, wantField: "password"},
		{name: "long email", body: `{"email":"` + strings.Repeat("a", 250) + `@example.com","password":"correct horse"}`, wantField: "email"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders, tc.body)

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %s)", resp.StatusCode, body)
			}
			if !strings.Contains(body, "validation_failed") || !strings.Contains(body, `"field":"`+tc.wantField+`"`) {
				t.Errorf("body = %q, want validation_failed naming %s", body, tc.wantField)
			}
		})
	}

	if calls := fake.AuthenticateCalls(); calls != 0 {
		t.Errorf("provider was called %d times for invalid input, want 0", calls)
	}
}

func TestSignInOriginAndLimiter(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_origin", "origin@example.com", "correct horse", true))
	credentials := `{"email":"origin@example.com","password":"correct horse"}`

	t.Run("origin", func(t *testing.T) {
		app := startAuthApp(t, fake, nil)

		for name, headers := range map[string]map[string]string{
			"missing": nil,
			"foreign": {"Origin": "https://evil.example"},
		} {
			resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/sign-in", headers, credentials)
			if resp.StatusCode != http.StatusForbidden || !strings.Contains(body, "forbidden_origin") {
				t.Errorf("%s origin: status %d body %s, want 403 forbidden_origin", name, resp.StatusCode, body)
			}
		}
	})

	t.Run("limiter", func(t *testing.T) {
		app := startAuthApp(t, fake, map[string]string{"AUTH_RATE_LIMIT_RPS": "1", "AUTH_RATE_LIMIT_BURST": "2"})
		b := newBrowser(t)

		var statuses []int
		for range 3 {
			resp, _ := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders, credentials)
			statuses = append(statuses, resp.StatusCode)
		}
		if statuses[2] != http.StatusTooManyRequests {
			t.Errorf("statuses = %v, want 429 last", statuses)
		}
	})
}

func TestSignInProviderDown(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_down", "down@example.com", "correct horse", true))
	app := startAuthApp(t, fake, nil)
	fake.SetAuthenticateFailure(http.StatusInternalServerError)
	defer fake.SetAuthenticateFailure(0)
	b := newBrowser(t)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"down@example.com","password":"correct horse"}`)

	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "provider_unavailable") {
		t.Fatalf("status %d body %s, want 503 provider_unavailable", resp.StatusCode, body)
	}
	if setCookieHeader(resp) != "" || b.sessionCookie(t, app.BaseURL) != nil {
		t.Error("a failed sign-in set a session cookie")
	}
}

func TestSignInProviderTimeout(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_slow", "slow@example.com", "correct horse", true))
	app := startAuthApp(t, fake, map[string]string{"WORKOS_TIMEOUT_SECONDS": "1"})
	fake.HoldAuthenticate(2 * time.Second)
	b := newBrowser(t)

	before := time.Now()
	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"slow@example.com","password":"correct horse"}`)
	elapsed := time.Since(before)

	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "provider_unavailable") {
		t.Fatalf("status %d body %s, want 503 provider_unavailable", resp.StatusCode, body)
	}
	if elapsed > 2*time.Second {
		t.Errorf("sign-in took %s, want an answer near the one second timeout", elapsed)
	}
	if setCookieHeader(resp) != "" || b.sessionCookie(t, app.BaseURL) != nil {
		t.Error("a timed-out sign-in set a session cookie")
	}
}

func TestSignInPasswordMethodDisabled(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_sso_only", "sso@example.com", "correct horse", true))
	app := startAuthApp(t, fake, nil)
	fake.SetAuthenticateFailureBody(http.StatusBadRequest, map[string]any{
		"error":             "sso_required",
		"error_description": "Password sign-in is disabled",
	})
	defer clearProviderFailures(fake)
	b := newBrowser(t)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"sso@example.com","password":"correct horse"}`)

	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, `"error":"provider_unavailable"`) {
		t.Fatalf("status %d body %s, want 503 provider_unavailable", resp.StatusCode, body)
	}
	if setCookieHeader(resp) != "" || b.sessionCookie(t, app.BaseURL) != nil {
		t.Error("a disabled password method set a session cookie")
	}
}

// clearProviderFailures restores normal behavior on every endpoint a
// forced-status test pinned.
func clearProviderFailures(fake *fakeWorkOS) {
	fake.SetAuthenticateFailure(0)
	fake.SetCreateUserFailure(0)
	fake.SetListUsersFailure(0)
	fake.SetSendVerificationFailure(0)
	fake.SetPasswordResetFailure(0)
}

func TestProviderQuotaAnswersUnavailable(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_quota", "quota@example.com", "correct horse", true))
	app, mailer := startAuthAppWithMail(t, fake, nil)

	cases := []struct {
		name  string
		path  string
		body  string
		force func()
	}{
		{
			name:  "sign-up",
			path:  "/v1/auth/sign-up",
			body:  `{"first_name":"Mad","last_name":"Hatter","email":"quota@example.com","password":"correct horse"}`,
			force: func() { fake.SetCreateUserFailure(http.StatusTooManyRequests) },
		},
		{
			name:  "forgot-password",
			path:  "/v1/auth/forgot-password",
			body:  `{"email":"quota@example.com"}`,
			force: func() { fake.SetPasswordResetFailure(http.StatusTooManyRequests) },
		},
		{
			name:  "sign-in",
			path:  "/v1/auth/sign-in",
			body:  `{"email":"quota@example.com","password":"correct horse"}`,
			force: func() { fake.SetAuthenticateFailure(http.StatusTooManyRequests) },
		},
		{
			name:  "reset-password",
			path:  "/v1/auth/reset-password",
			body:  `{"token":"prt_quota","password":"a brand new secret"}`,
			force: func() { fake.SetPasswordResetFailure(http.StatusTooManyRequests) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mailer.Reset()
			tc.force()
			defer clearProviderFailures(fake)

			resp, body := newBrowser(t).postJSON(t, app.BaseURL+tc.path, allowedOriginHeaders, tc.body)

			if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "provider_unavailable") {
				t.Fatalf("status %d body %s, want 503 provider_unavailable", resp.StatusCode, body)
			}
			if strings.Contains(body, "pending_token") {
				t.Errorf("body = %q, want no pending token", body)
			}
			if sent := mailer.Sent(); len(sent) != 0 {
				t.Errorf("sent %d messages, want 0", len(sent))
			}
		})
	}
}

func TestVerifyEmailStartsSession(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_pending", "pending@example.com", "correct horse", false)
	fake.AddUser(user)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	_, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"pending@example.com","password":"correct horse"}`)
	token := pendingToken(t, body)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/verify-email", allowedOriginHeaders,
		`{"pending_token":"`+token+`","code":"`+fake.CurrentCode(user.Email)+`"}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	decodeAccessToken(t, body)
	if !strings.Contains(setCookieHeader(resp), "sow_session=v1.k1.") {
		t.Errorf("Set-Cookie = %q, want a session cookie", setCookieHeader(resp))
	}

	stored, err := newUserStore(t).Get(t.Context(), user.ID)
	if err != nil || stored == nil {
		t.Fatalf("user row after verification: %+v (err %v)", stored, err)
	}
	if stored.Email != user.Email || stored.Name != "Mad Hatter" {
		t.Errorf("stored user = %+v", stored)
	}

	resp, body = newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"pending@example.com","password":"correct horse"}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("sign-in after verification: status %d body %s, want 200", resp.StatusCode, body)
	}
}

func TestVerifyEmailRejectsCode(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_codes", "codes@example.com", "correct horse", false)
	fake.AddUser(user)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	_, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"codes@example.com","password":"correct horse"}`)
	token := pendingToken(t, body)
	code := fake.CurrentCode(user.Email)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/verify-email", allowedOriginHeaders,
		`{"pending_token":"`+token+`","code":"000000"}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `"error":"invalid_code"`) {
		t.Errorf("wrong code: status %d body %s, want 400 invalid_code", resp.StatusCode, body)
	}
	if setCookieHeader(resp) != "" {
		t.Errorf("a wrong code set a cookie: %q", setCookieHeader(resp))
	}

	resp, body = b.postJSON(t, app.BaseURL+"/v1/auth/verify-email", allowedOriginHeaders,
		`{"pending_token":"`+token+`","code":"`+code+`"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("correct code: status %d body %s, want 200", resp.StatusCode, body)
	}

	resp, body = b.postJSON(t, app.BaseURL+"/v1/auth/verify-email", allowedOriginHeaders,
		`{"pending_token":"`+token+`","code":"`+code+`"}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `"error":"invalid_code"`) {
		t.Errorf("reused token: status %d body %s, want 400 invalid_code", resp.StatusCode, body)
	}

	resp, body = b.postJSON(t, app.BaseURL+"/v1/auth/verify-email", allowedOriginHeaders,
		`{"pending_token":"`+token+`","code":"12345"}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `"field":"code"`) {
		t.Errorf("malformed code: status %d body %s, want 400 naming code", resp.StatusCode, body)
	}
}

func TestSignUpCreatesAndPends(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-up", allowedOriginHeaders,
		`{"first_name":" Mad ","last_name":"Hatter","email":"fresh@example.com","password":"correct horse"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}
	if token := pendingToken(t, body); !strings.HasPrefix(token, "pat_") {
		t.Errorf("pending_token = %q, want the provider's token", token)
	}

	created, ok := fake.UserOf("fresh@example.com")
	if !ok {
		t.Fatal("the provider holds no user for the address")
	}
	if created.FirstName != "Mad" || created.LastName != "Hatter" {
		t.Errorf("created user name = %q %q, want trimmed Mad Hatter", created.FirstName, created.LastName)
	}
	if created.Password != "correct horse" || created.EmailVerified {
		t.Errorf("created user = %+v, want the submitted password and an unverified email", created)
	}
	if fake.CurrentCode("fresh@example.com") == "" {
		t.Error("no verification code is on file for the new user")
	}

	if setCookieHeader(resp) != "" || b.sessionCookie(t, app.BaseURL) != nil {
		t.Error("sign-up set a session cookie")
	}
	if stored, _ := newUserStore(t).Get(t.Context(), created.ID); stored != nil {
		t.Errorf("sign-up provisioned a user before verification: %+v", stored)
	}
}

func TestSignUpDuplicateIsIndistinguishable(t *testing.T) {
	fake := newFakeWorkOS(t)
	existing := account("user_taken", "taken@example.com", "correct horse", true)
	fake.AddUser(existing)
	fake.AddUser(account("user_pending_shape", "shape@example.com", "correct horse", false))
	app, mailer := startAuthAppWithMail(t, fake, nil)
	b := newBrowser(t)

	_, shapeBody := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"shape@example.com","password":"correct horse"}`)
	issued := pendingToken(t, shapeBody)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-up", allowedOriginHeaders,
		`{"first_name":"Someone","last_name":"Else","email":"taken@example.com","password":"another horse"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}
	token := pendingToken(t, body)
	if len(token) != len(issued) {
		t.Errorf("decoy pending_token %q is %d characters, want %d like the provider's", token, len(token), len(issued))
	}
	if !opaqueToken(token) {
		t.Errorf("decoy pending_token = %q, want the character class of a provider token", token)
	}
	if strings.Contains(body, "taken") || strings.Contains(body, "exists") {
		t.Errorf("body = %q, want nothing about the existing account", body)
	}

	held, _ := fake.UserOf("taken@example.com")
	if held.ID != existing.ID || held.Password != existing.Password {
		t.Errorf("held user = %+v, want the existing account untouched", held)
	}
	if fake.SentCodes(existing.ID) != 0 || fake.CurrentCode(existing.Email) != "" {
		t.Error("a verification code was sent to the existing account")
	}

	sent := mailer.WaitForSent(t, 1)
	if len(sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(sent))
	}
	if sent[0].To != existing.Email || sent[0].Subject != "You already have a SoW account" {
		t.Errorf("message = %+v, want the account exists email to the holder", sent[0])
	}
	for _, want := range []string{"already exists", "http://localhost:5173/", "http://localhost:5173/forgot-password"} {
		if !strings.Contains(sent[0].Text, want) {
			t.Errorf("message body lacks %q:\n%s", want, sent[0].Text)
		}
	}

	resp, body = b.postJSON(t, app.BaseURL+"/v1/auth/verify-email", allowedOriginHeaders,
		`{"pending_token":"`+token+`","code":"123456"}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `"error":"invalid_code"`) {
		t.Errorf("verifying the decoy token: status %d body %s, want 400 invalid_code", resp.StatusCode, body)
	}
}

func TestSignUpVerifiedByProviderStartsSession(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.SetCreatesVerifiedUsers(true)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-up", allowedOriginHeaders,
		`{"first_name":"Dor","last_name":"Mouse","email":"settled@example.com","password":"correct horse"}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	decodeAccessToken(t, body)
	if !strings.Contains(setCookieHeader(resp), "sow_session=v1.k1.") {
		t.Errorf("Set-Cookie = %q, want a session cookie", setCookieHeader(resp))
	}
	if b.sessionCookie(t, app.BaseURL) == nil {
		t.Fatal("jar holds no session cookie after sign-up")
	}

	created, ok := fake.UserOf("settled@example.com")
	if !ok {
		t.Fatal("the provider holds no user for the address")
	}
	stored, err := newUserStore(t).Get(t.Context(), created.ID)
	if err != nil || stored == nil {
		t.Fatalf("user row after sign-up: %+v (err %v)", stored, err)
	}
	if stored.Name != "Dor Mouse" || stored.Email != "settled@example.com" {
		t.Errorf("stored user = %+v", stored)
	}
}

func TestSignUpDuplicateAnswersBeforeMailLeaves(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_slowmail", "slowmail@example.com", "correct horse", true))
	app, mailer := startAuthAppWithMail(t, fake, nil)
	release := mailer.Gate()
	defer release()

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/sign-up", allowedOriginHeaders,
		`{"first_name":"Someone","last_name":"Else","email":"slowmail@example.com","password":"another horse"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}
	if sent := mailer.Sent(); len(sent) != 0 {
		t.Fatalf("captured %d messages while the send was held, want 0", len(sent))
	}

	release()
	if sent := mailer.WaitForSent(t, 1); sent[0].To != "slowmail@example.com" {
		t.Errorf("message = %+v, want the account exists email to the holder", sent[0])
	}
}

func TestSignUpWeakPassword(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/sign-up", allowedOriginHeaders,
		`{"first_name":"Mad","last_name":"Hatter","email":"weakling@example.com","password":"weakweak"}`)

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body %s)", resp.StatusCode, body)
	}
	if !strings.Contains(body, `"error":"weak_password"`) || !strings.Contains(body, `"message":"Password is too weak"`) {
		t.Errorf("body = %q, want weak_password carrying the provider message", body)
	}
	if _, ok := fake.UserOf("weakling@example.com"); ok {
		t.Error("a rejected password created a user")
	}
}

func TestSignUpValidation(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)

	cases := []struct {
		name      string
		body      string
		wantField string
	}{
		{name: "empty first name", body: `{"first_name":"","last_name":"Hatter","email":"a@example.com","password":"correct horse"}`, wantField: "first_name"},
		{name: "long last name", body: `{"first_name":"Mad","last_name":"` + strings.Repeat("a", 101) + `","email":"a@example.com","password":"correct horse"}`, wantField: "last_name"},
		{name: "malformed email", body: `{"first_name":"Mad","last_name":"Hatter","email":"not-an-address","password":"correct horse"}`, wantField: "email"},
		{name: "short password", body: `{"first_name":"Mad","last_name":"Hatter","email":"a@example.com","password":"seven77"}`, wantField: "password"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/sign-up", allowedOriginHeaders, tc.body)

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %s)", resp.StatusCode, body)
			}
			if !strings.Contains(body, "validation_failed") || !strings.Contains(body, `"field":"`+tc.wantField+`"`) {
				t.Errorf("body = %q, want validation_failed naming %s", body, tc.wantField)
			}
		})
	}
}

func TestSignUpThenVerifyProvisions(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	_, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-up", allowedOriginHeaders,
		`{"first_name":"March","last_name":"Hare","email":"march@example.com","password":"correct horse"}`)
	token := pendingToken(t, body)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/verify-email", allowedOriginHeaders,
		`{"pending_token":"`+token+`","code":"`+fake.CurrentCode("march@example.com")+`"}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	decodeAccessToken(t, body)
	if !strings.Contains(setCookieHeader(resp), "sow_session=v1.k1.") {
		t.Errorf("Set-Cookie = %q, want a session cookie", setCookieHeader(resp))
	}

	created, _ := fake.UserOf("march@example.com")
	stored, err := newUserStore(t).Get(t.Context(), created.ID)
	if err != nil || stored == nil {
		t.Fatalf("user row after verification: %+v (err %v)", stored, err)
	}
	if stored.Name != "March Hare" || stored.Email != "march@example.com" {
		t.Errorf("stored user = %+v", stored)
	}
}

func TestResendVerificationRotatesCode(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_resend", "resend@example.com", "correct horse", false)
	fake.AddUser(user)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"resend@example.com","password":"correct horse"}`)
	first := fake.CurrentCode(user.Email)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/resend-verification", allowedOriginHeaders,
		`{"email":"resend@example.com"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}

	fake.WaitForSentCodes(t, user.ID, 1)
	if second := fake.CurrentCode(user.Email); second == first || second == "" {
		t.Errorf("code = %q, want a fresh code replacing %q", second, first)
	}
	if sends := fake.SentCodes(user.ID); sends != 1 {
		t.Errorf("send count = %d, want 1", sends)
	}
}

func TestResendVerificationNeutral(t *testing.T) {
	fake := newFakeWorkOS(t)
	verified := account("user_settled", "settled@example.com", "correct horse", true)
	fake.AddUser(verified)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	for _, email := range []string{"nobody@example.com", "settled@example.com"} {
		resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/resend-verification", allowedOriginHeaders,
			`{"email":"`+email+`"}`)
		if resp.StatusCode != http.StatusAccepted {
			t.Errorf("%s: status = %d, want 202 (body %s)", email, resp.StatusCode, body)
		}
	}

	if sends := fake.SentCodes(verified.ID); sends != 0 {
		t.Errorf("send count = %d, want 0", sends)
	}
	if code := fake.CurrentCode(verified.Email); code != "" {
		t.Errorf("code = %q, want none for a verified address", code)
	}
}

func TestResendVerificationProviderDown(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_listing", "listing@example.com", "correct horse", false))
	app := startAuthApp(t, fake, nil)
	fake.SetListUsersFailure(http.StatusInternalServerError)
	defer clearProviderFailures(fake)

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/resend-verification", allowedOriginHeaders,
		`{"email":"listing@example.com"}`)

	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "provider_unavailable") {
		t.Fatalf("status %d body %s, want 503 provider_unavailable", resp.StatusCode, body)
	}
}

func TestResendVerificationSendFailureStillAccepts(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_sending", "sending@example.com", "correct horse", false)
	fake.AddUser(user)
	app := startAuthApp(t, fake, nil)
	fake.SetSendVerificationFailure(http.StatusInternalServerError)
	defer clearProviderFailures(fake)

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/resend-verification", allowedOriginHeaders,
		`{"email":"sending@example.com"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}
	for _, line := range app.Logs.WaitForLine(t, "resending verification code failed") {
		if strings.Contains(line, user.Email) {
			t.Errorf("a log line carries the address: %s", line)
		}
	}
}

func TestResendVerificationAnswersBeforeTheCodeLeaves(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_held", "held@example.com", "correct horse", false)
	fake.AddUser(user)
	app := startAuthApp(t, fake, nil)
	release := fake.GateSendVerification()
	defer release()

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/resend-verification", allowedOriginHeaders,
		`{"email":"held@example.com"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}
	if sends := fake.SentCodes(user.ID); sends != 0 {
		t.Fatalf("send count = %d while the send was held, want 0", sends)
	}

	release()
	fake.WaitForSentCodes(t, user.ID, 1)
}

// resetExpiry reads the expiry the password reset email states.
func resetExpiry(t *testing.T, text string) time.Time {
	t.Helper()

	_, tail, found := strings.Cut(text, "expires at ")
	if !found {
		t.Fatalf("message body states no expiry:\n%s", text)
	}
	stamp, _, _ := strings.Cut(tail, ".")

	parsed, err := time.Parse(time.RFC1123, stamp)
	if err != nil {
		t.Fatalf("expiry %q is not RFC 1123: %v", stamp, err)
	}
	return parsed
}

func TestForgotPasswordEmailsLink(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_forgetful", "forgetful@example.com", "correct horse", true)
	fake.AddUser(user)
	app, mailer := startAuthAppWithMail(t, fake, nil)
	b := newBrowser(t)

	before := time.Now()
	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/forgot-password", allowedOriginHeaders,
		`{"email":"forgetful@example.com"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}

	sent := mailer.WaitForSent(t, 1)
	if len(sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(sent))
	}
	if sent[0].To != user.Email || sent[0].Subject != "Reset your SoW password" {
		t.Errorf("message = %+v, want the reset email to the holder", sent[0])
	}

	token := fake.LastResetToken(user.Email)
	if token == "" {
		t.Fatal("the provider issued no reset token")
	}
	if !strings.Contains(sent[0].Text, "http://localhost:5173/reset-password?token="+token) {
		t.Errorf("message body carries no reset link:\n%s", sent[0].Text)
	}

	expiry := resetExpiry(t, sent[0].Text)
	if expiry.Before(before.Add(14*time.Minute)) || expiry.After(before.Add(16*time.Minute)) {
		t.Errorf("stated expiry = %s, want about fifteen minutes out", expiry)
	}

	if strings.Contains(app.Logs.String(), "prt_") {
		t.Error("the application log carries a reset token")
	}
}

func TestForgotPasswordNeutral(t *testing.T) {
	fake := newFakeWorkOS(t)
	app, mailer := startAuthAppWithMail(t, fake, nil)

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/forgot-password", allowedOriginHeaders,
		`{"email":"nobody@example.com"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}
	if sent := mailer.Sent(); len(sent) != 0 {
		t.Errorf("sent %d messages for an unknown address, want 0", len(sent))
	}
}

func TestForgotPasswordMailFailureStill202(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_unreachable", "unreachable@example.com", "correct horse", true))
	app, mailer := startAuthAppWithMail(t, fake, nil)
	mailer.FailWith(errors.New("provider refused the message"))

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/forgot-password", allowedOriginHeaders,
		`{"email":"unreachable@example.com"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}

	warnings := 0
	for _, line := range app.Logs.WaitForLine(t, "sending password reset email failed") {
		if strings.Contains(line, "sending password reset email failed") {
			warnings++
			if !strings.Contains(line, `"level":"WARN"`) {
				t.Errorf("send failure logged at the wrong level: %s", line)
			}
			if strings.Contains(line, "unreachable@example.com") {
				t.Errorf("the log line carries the address: %s", line)
			}
		}
	}
	if warnings != 1 {
		t.Errorf("logged %d send failures, want 1", warnings)
	}
}

func TestForgotPasswordUnreadableExpiry(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_expiry", "expiry@example.com", "correct horse", true))
	fake.SetResetExpiryText("not a timestamp")
	app, mailer := startAuthAppWithMail(t, fake, nil)

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/forgot-password", allowedOriginHeaders,
		`{"email":"expiry@example.com"}`)

	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "provider_unavailable") {
		t.Fatalf("status %d body %s, want 503 provider_unavailable", resp.StatusCode, body)
	}
	if sent := mailer.Sent(); len(sent) != 0 {
		t.Errorf("sent %d messages with no readable expiry, want 0", len(sent))
	}
}

func TestForgotPasswordAnswersBeforeMailLeaves(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_waiting", "waiting@example.com", "correct horse", true)
	fake.AddUser(user)
	app, mailer := startAuthAppWithMail(t, fake, nil)
	release := mailer.Gate()
	defer release()

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/forgot-password", allowedOriginHeaders,
		`{"email":"waiting@example.com"}`)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", resp.StatusCode, body)
	}
	if sent := mailer.Sent(); len(sent) != 0 {
		t.Fatalf("captured %d messages while the send was held, want 0", len(sent))
	}

	release()
	if sent := mailer.WaitForSent(t, 1); sent[0].To != user.Email {
		t.Errorf("message = %+v, want the reset email to the holder", sent[0])
	}
}

func TestProviderErrorsStayOutOfTheLog(t *testing.T) {
	fake := newFakeWorkOS(t)
	fake.AddUser(account("user_leaky", "leaky@example.com", "correct horse", true))
	app := startAuthApp(t, fake, nil)
	fake.SetAuthenticateFailureBody(http.StatusUnprocessableEntity, map[string]any{
		"code":                         "mfa_enrollment",
		"pending_authentication_token": "pat_secret_never_logged",
	})
	defer clearProviderFailures(fake)

	resp, body := newBrowser(t).postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"leaky@example.com","password":"correct horse"}`)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", resp.StatusCode, body)
	}
	if strings.Contains(body, "pat_secret_never_logged") {
		t.Errorf("body = %q, want no pending token", body)
	}
	if strings.Contains(app.Logs.String(), "pat_secret_never_logged") {
		t.Error("the application log carries the provider's pending authentication token")
	}
}

func TestResetPasswordProviderRefusesSignIn(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_locked", "locked@example.com", "correct horse", true)
	fake.AddUser(user)
	app, _ := startAuthAppWithMail(t, fake, nil)
	b := newBrowser(t)

	b.postJSON(t, app.BaseURL+"/v1/auth/forgot-password", allowedOriginHeaders, `{"email":"locked@example.com"}`)
	token := fake.LastResetToken(user.Email)
	if token == "" {
		t.Fatal("the provider issued no reset token")
	}

	fake.SetAuthenticateFailure(http.StatusBadRequest)
	defer clearProviderFailures(fake)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/reset-password", allowedOriginHeaders,
		`{"token":"`+token+`","password":"a brand new secret"}`)

	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "provider_unavailable") {
		t.Fatalf("status %d body %s, want 503 provider_unavailable", resp.StatusCode, body)
	}
	if setCookieHeader(resp) != "" || b.sessionCookie(t, app.BaseURL) != nil {
		t.Error("a failed sign-in after the reset set a session cookie")
	}
}

func TestResetPasswordSignsIn(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_resetting", "resetting@example.com", "correct horse", false)
	fake.AddUser(user)
	app, _ := startAuthAppWithMail(t, fake, nil)
	b := newBrowser(t)

	b.postJSON(t, app.BaseURL+"/v1/auth/forgot-password", allowedOriginHeaders, `{"email":"resetting@example.com"}`)
	token := fake.LastResetToken(user.Email)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/reset-password", allowedOriginHeaders,
		`{"token":"`+token+`","password":"a brand new secret"}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	decodeAccessToken(t, body)
	if !strings.Contains(setCookieHeader(resp), "sow_session=v1.k1.") {
		t.Errorf("Set-Cookie = %q, want a session cookie", setCookieHeader(resp))
	}

	if got := fake.PasswordOf(user.Email); got != "a brand new secret" {
		t.Errorf("stored password = %q, want the new one", got)
	}
	if held, _ := fake.UserOf(user.Email); !held.EmailVerified {
		t.Error("the reset left the address unverified")
	}

	resp, body = b.postJSON(t, app.BaseURL+"/v1/auth/reset-password", allowedOriginHeaders,
		`{"token":"`+token+`","password":"another secret"}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `"error":"invalid_reset_token"`) {
		t.Errorf("reusing the token: status %d body %s, want 400 invalid_reset_token", resp.StatusCode, body)
	}
}

func TestResetPasswordRejectsToken(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/reset-password", allowedOriginHeaders,
		`{"token":"prt_nothing","password":"a brand new secret"}`)

	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `"error":"invalid_reset_token"`) {
		t.Fatalf("status %d body %s, want 400 invalid_reset_token", resp.StatusCode, body)
	}
	if setCookieHeader(resp) != "" || b.sessionCookie(t, app.BaseURL) != nil {
		t.Error("a rejected reset set a session cookie")
	}
}

func TestResetPasswordWeak(t *testing.T) {
	fake := newFakeWorkOS(t)
	user := account("user_weakening", "weakening@example.com", "correct horse", true)
	fake.AddUser(user)
	app, _ := startAuthAppWithMail(t, fake, nil)
	b := newBrowser(t)

	b.postJSON(t, app.BaseURL+"/v1/auth/forgot-password", allowedOriginHeaders, `{"email":"weakening@example.com"}`)
	token := fake.LastResetToken(user.Email)

	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/reset-password", allowedOriginHeaders,
		`{"token":"`+token+`","password":"weakweak"}`)

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body %s)", resp.StatusCode, body)
	}
	if !strings.Contains(body, `"error":"weak_password"`) || !strings.Contains(body, `"message":"Password is too weak"`) {
		t.Errorf("body = %q, want weak_password carrying the provider message", body)
	}
	if got := fake.PasswordOf(user.Email); got != "correct horse" {
		t.Errorf("stored password = %q, want the old one", got)
	}
}

func TestLegacyAccountSignsIn(t *testing.T) {
	fake := newFakeWorkOS(t)
	legacy := account("user_legacy", "legacy@example.com", "correct horse", true)
	app := startAuthApp(t, fake, nil)

	_, callbackCookie := newBrowser(t).login(t, app, fake, legacy, "/dashboard")
	if callbackCookie == nil {
		t.Fatal("the callback flow set no session cookie")
	}
	provisioned, err := newUserStore(t).Get(t.Context(), legacy.ID)
	if err != nil || provisioned == nil {
		t.Fatalf("user row after the callback flow: %+v (err %v)", provisioned, err)
	}

	b := newBrowser(t)
	resp, body := b.postJSON(t, app.BaseURL+"/v1/auth/sign-in", allowedOriginHeaders,
		`{"email":"legacy@example.com","password":"correct horse"}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	decodeAccessToken(t, body)

	cookie := b.sessionCookie(t, app.BaseURL)
	if cookie == nil {
		t.Fatal("jar holds no session cookie after sign-in")
	}
	payload, err := testKeyring(t).Open(cookie.Value)
	if err != nil {
		t.Fatalf("opening cookie: %v", err)
	}
	if payload.UserID != legacy.ID {
		t.Errorf("cookie user id = %q, want the account created through the callback", payload.UserID)
	}

	stored, err := newUserStore(t).Get(t.Context(), legacy.ID)
	if err != nil || stored == nil {
		t.Fatalf("user row after sign-in: %+v (err %v)", stored, err)
	}
	if stored.ID != provisioned.ID || stored.Email != provisioned.Email {
		t.Errorf("stored user = %+v, want the same record as %+v", stored, provisioned)
	}
}
