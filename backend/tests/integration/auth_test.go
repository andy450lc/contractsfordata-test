package integration

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/mail"
	"github.com/pixels-two/sow/backend/internal/session"
)

const allowedOrigin = "http://allowed.example"

var alice = fakeUser{
	ID:        "user_alice",
	Email:     "alice@example.com",
	FirstName: "Alice",
	LastName:  "Liddell",
	CreatedAt: time.Now().Add(-48 * time.Hour).Truncate(time.Millisecond),
	UpdatedAt: time.Now().Add(-24 * time.Hour).Truncate(time.Millisecond),
}

// startAuthApp boots the app against the fake provider.
func startAuthApp(t *testing.T, fake *fakeWorkOS, overrides map[string]string) *testApp {
	t.Helper()

	env := map[string]string{
		"WORKOS_BASE_URL": fake.Server.URL,
		"WORKOS_JWKS_URL": fake.JWKSURL(),
	}
	for k, v := range overrides {
		env[k] = v
	}
	return startApp(t, env)
}

// startAuthAppWithMail boots the app against the fake provider with a
// capturing fake mailer in place of the real one. A test then asserts
// on the email a service sent.
func startAuthAppWithMail(t *testing.T, fake *fakeWorkOS, overrides map[string]string) (*testApp, *fakeMailer) {
	t.Helper()

	env := map[string]string{
		"WORKOS_BASE_URL": fake.Server.URL,
		"WORKOS_JWKS_URL": fake.JWKSURL(),
	}
	for k, v := range overrides {
		env[k] = v
	}

	fm := &fakeMailer{}
	app := startApp(t, env, fx.Replace(fx.Annotate(fm, fx.As(new(mail.Mailer)))))
	return app, fm
}

// browser is an HTTP client with a cookie jar that never follows
// redirects, so tests inspect every 302 and Set-Cookie.
type browser struct {
	client *http.Client
}

func newBrowser(t *testing.T) *browser {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("creating cookie jar: %v", err)
	}
	return &browser{client: &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (b *browser) do(t *testing.T, method string, target string, headers map[string]string) (*http.Response, string) {
	t.Helper()

	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	return resp, string(body)
}

// sessionCookie returns the jar's session cookie for the app, or nil.
func (b *browser) sessionCookie(t *testing.T, baseURL string) *http.Cookie {
	t.Helper()

	u, err := url.Parse(baseURL + session.CookiePath)
	if err != nil {
		t.Fatalf("parsing url: %v", err)
	}
	for _, c := range b.client.Jar.Cookies(u) {
		if c.Name == "sow_session" {
			return c
		}
	}
	return nil
}

// setCookie plants a raw session cookie value in the jar.
func (b *browser) setCookie(t *testing.T, baseURL string, value string) {
	t.Helper()

	u, err := url.Parse(baseURL + session.CookiePath)
	if err != nil {
		t.Fatalf("parsing url: %v", err)
	}
	b.client.Jar.SetCookies(u, []*http.Cookie{{Name: "sow_session", Value: value, Path: session.CookiePath}})
}

// login walks login → callback for the user and returns the callback
// response and the jar-held cookie.
func (b *browser) login(t *testing.T, app *testApp, fake *fakeWorkOS, user fakeUser, returnTo string) (*http.Response, *http.Cookie) {
	t.Helper()

	resp, _ := b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/login?provider=google&returnTo="+url.QueryEscape(returnTo), nil)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("login status = %d, want 302", resp.StatusCode)
	}
	state := locationQuery(t, resp).Get("state")

	code := fake.IssueCode(user)
	resp, _ = b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/callback?code="+code+"&state="+url.QueryEscape(state), nil)
	return resp, b.sessionCookie(t, app.BaseURL)
}

func locationQuery(t *testing.T, resp *http.Response) url.Values {
	t.Helper()

	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parsing Location %q: %v", resp.Header.Get("Location"), err)
	}
	return loc.Query()
}

func setCookieHeader(resp *http.Response) string {
	return strings.Join(resp.Header.Values("Set-Cookie"), "; ")
}

func testKeyring(t *testing.T) *session.Keyring {
	t.Helper()

	ring, err := session.ParseKeyring(testSessionCookieKey)
	if err != nil {
		t.Fatalf("parsing test keyring: %v", err)
	}
	return ring
}

func TestLoginRedirectsToGoogle(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	resp, _ := b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/login?provider=google&returnTo=%2Fagreements%2F1", nil)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, fake.Server.URL+"/user_management/authorize?") {
		t.Fatalf("Location = %q, want the provider's authorize endpoint", loc)
	}
	q := locationQuery(t, resp)
	for key, want := range map[string]string{
		"client_id":     "client_test",
		"redirect_uri":  "http://localhost:8080/v1/auth/callback",
		"response_type": "code",
		"provider":      "GoogleOAuth",
	} {
		if got := q.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if q.Has("screen_hint") {
		t.Errorf("screen_hint = %q, want the parameter absent", q.Get("screen_hint"))
	}

	returnTo, err := session.VerifyState(testKeyring(t).ActiveKey(), q.Get("state"), time.Now())
	if err != nil {
		t.Fatalf("state does not verify: %v", err)
	}
	if returnTo != "/agreements/1" {
		t.Errorf("state returnTo = %q, want /agreements/1", returnTo)
	}
}

func TestLoginDefaultsAndValidation(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	resp, _ := b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/login?provider=google&returnTo=https%3A%2F%2Fevil.example", nil)
	q := locationQuery(t, resp)
	returnTo, err := session.VerifyState(testKeyring(t).ActiveKey(), q.Get("state"), time.Now())
	if err != nil || returnTo != "/dashboard" {
		t.Errorf("returnTo = %q (err %v), want /dashboard", returnTo, err)
	}

	for _, query := range []string{"", "?provider=authkit"} {
		resp, body := b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/login"+query, nil)
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") {
			t.Errorf("login%q: status = %d body = %q, want 400 validation_failed", query, resp.StatusCode, body)
		}
	}
}

func TestCallbackSetsCookieAndProvisionsUser(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)

	resp, cookie := b.login(t, app, fake, alice, "/agreements/1")

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("callback status = %d, want 302", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "http://localhost:5173/callback?returnTo=%2Fagreements%2F1" {
		t.Errorf("Location = %q", got)
	}

	header := setCookieHeader(resp)
	for _, want := range []string{"sow_session=v1.k1.", "Path=/v1/auth", "HttpOnly", "SameSite=Strict", "Max-Age=604800"} {
		if !strings.Contains(header, want) {
			t.Errorf("Set-Cookie %q lacks %q", header, want)
		}
	}
	if strings.Contains(header, "Secure") {
		t.Errorf("dev stage cookie must not be Secure: %q", header)
	}
	if cookie == nil {
		t.Fatal("jar holds no session cookie")
	}

	payload, err := testKeyring(t).Open(cookie.Value)
	if err != nil {
		t.Fatalf("opening cookie: %v", err)
	}
	if payload.UserID != alice.ID || payload.SessionID == "" || payload.RefreshToken == "" {
		t.Errorf("cookie payload = %+v", payload)
	}

	stored, err := newUserStore(t).Get(t.Context(), alice.ID)
	if err != nil || stored == nil {
		t.Fatalf("user row after callback: %+v (err %v)", stored, err)
	}
	if stored.Email != alice.Email || stored.Name != "Alice Liddell" {
		t.Errorf("stored user = %+v", stored)
	}
}

func TestCallbackFailuresRedirectWithoutSession(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	key := testKeyring(t).ActiveKey()

	freshState := func() string {
		state, err := session.IssueState(key, "/dashboard", time.Now())
		if err != nil {
			t.Fatalf("issuing state: %v", err)
		}
		return state
	}
	expiredState, err := session.IssueState(key, "/dashboard", time.Now().Add(-session.StateTTL-time.Minute))
	if err != nil {
		t.Fatalf("issuing expired state: %v", err)
	}
	valid := freshState()
	tampered := valid[:len(valid)-1] + "x"

	bob := fakeUser{ID: "user_bob", Email: "bob@example.com", CreatedAt: alice.CreatedAt, UpdatedAt: alice.UpdatedAt}

	cases := []struct {
		name     string
		query    string
		failWith int
		wantCode string
	}{
		{name: "missing state", query: "code=" + fake.IssueCode(bob), wantCode: "invalid_state"},
		{name: "tampered state", query: "code=" + fake.IssueCode(bob) + "&state=" + url.QueryEscape(tampered), wantCode: "invalid_state"},
		{name: "expired state", query: "code=" + fake.IssueCode(bob) + "&state=" + url.QueryEscape(expiredState), wantCode: "invalid_state"},
		{name: "missing code", query: "state=" + url.QueryEscape(freshState()), wantCode: "flow_incomplete"},
		{name: "unknown code", query: "code=nope&state=" + url.QueryEscape(freshState()), wantCode: "flow_incomplete"},
		{name: "provider refuses", query: "code=" + fake.IssueCode(bob) + "&state=" + url.QueryEscape(freshState()), failWith: http.StatusBadRequest, wantCode: "flow_incomplete"},
		{name: "provider down", query: "code=" + fake.IssueCode(bob) + "&state=" + url.QueryEscape(freshState()), failWith: http.StatusBadGateway, wantCode: "provider_unavailable"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake.SetAuthenticateFailure(tc.failWith)
			defer fake.SetAuthenticateFailure(0)
			b := newBrowser(t)

			resp, _ := b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/callback?"+tc.query, nil)

			if resp.StatusCode != http.StatusFound {
				t.Fatalf("status = %d, want 302", resp.StatusCode)
			}
			if got := resp.Header.Get("Location"); got != "http://localhost:5173/?error="+tc.wantCode {
				t.Errorf("Location = %q, want error %s", got, tc.wantCode)
			}
			if header := setCookieHeader(resp); !strings.Contains(header, "Max-Age=0") {
				t.Errorf("Set-Cookie = %q, want a clearing cookie", header)
			}
			if b.sessionCookie(t, app.BaseURL) != nil {
				t.Error("jar holds a session cookie after a failed callback")
			}
		})
	}

	if stored, _ := newUserStore(t).Get(t.Context(), bob.ID); stored != nil {
		t.Errorf("failed callbacks provisioned a user: %+v", stored)
	}
}

func TestCallbackUnreachableProvider(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, map[string]string{"WORKOS_BASE_URL": "http://127.0.0.1:9"})
	b := newBrowser(t)

	state, err := session.IssueState(testKeyring(t).ActiveKey(), "/dashboard", time.Now())
	if err != nil {
		t.Fatalf("issuing state: %v", err)
	}
	resp, _ := b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/callback?code=x&state="+url.QueryEscape(state), nil)

	if got := resp.Header.Get("Location"); got != "http://localhost:5173/?error=provider_unavailable" {
		t.Errorf("Location = %q, want provider_unavailable", got)
	}
}

func TestRefreshRotatesCookieAndMintsToken(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	b := newBrowser(t)
	_, first := b.login(t, app, fake, alice, "/dashboard")

	origin := map[string]string{"Origin": allowedOrigin}
	before := time.Now()
	resp, body := b.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", origin)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	token := decodeAccessToken(t, body)
	if token.ExpiresAt < before.Add(4*time.Minute).UnixMilli() || token.ExpiresAt > before.Add(6*time.Minute).UnixMilli() {
		t.Errorf("expires_at = %d, want about five minutes out", token.ExpiresAt)
	}
	if strings.Contains(body, "refresh_token") || strings.Contains(body, "rt_") {
		t.Errorf("refresh response leaks the refresh token: %s", body)
	}

	second := b.sessionCookie(t, app.BaseURL)
	if second == nil || second.Value == first.Value {
		t.Fatal("refresh did not rotate the cookie")
	}

	resp, body = getWithAuth(t, app.BaseURL+"/v1/me", "Bearer "+token.AccessToken)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, alice.Email) {
		t.Errorf("/v1/me with refreshed token: status %d body %s", resp.StatusCode, body)
	}

	resp, _ = b.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", origin)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("second refresh status = %d, want 200", resp.StatusCode)
	}

	stale := newBrowser(t)
	stale.setCookie(t, app.BaseURL, first.Value)
	resp, _ = stale.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", origin)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("refresh with the rotated-out cookie status = %d, want 401", resp.StatusCode)
	}
}

func TestRefreshRefusals(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	origin := map[string]string{"Origin": allowedOrigin}

	t.Run("no cookie", func(t *testing.T) {
		resp, body := newBrowser(t).do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", origin)
		if resp.StatusCode != http.StatusUnauthorized || !strings.Contains(body, "unauthorized") {
			t.Errorf("status %d body %s, want 401 unauthorized", resp.StatusCode, body)
		}
		if !strings.Contains(setCookieHeader(resp), "Max-Age=0") {
			t.Errorf("Set-Cookie = %q, want a clearing cookie", setCookieHeader(resp))
		}
	})

	t.Run("forged cookie", func(t *testing.T) {
		b := newBrowser(t)
		b.setCookie(t, app.BaseURL, "v1.k1.notreal")
		resp, _ := b.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", origin)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("provider refuses", func(t *testing.T) {
		b := newBrowser(t)
		b.login(t, app, fake, alice, "/dashboard")
		fake.SetAuthenticateFailure(http.StatusBadRequest)
		defer fake.SetAuthenticateFailure(0)

		resp, _ := b.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", origin)
		if resp.StatusCode != http.StatusUnauthorized || !strings.Contains(setCookieHeader(resp), "Max-Age=0") {
			t.Errorf("status %d Set-Cookie %q, want 401 with a clearing cookie", resp.StatusCode, setCookieHeader(resp))
		}
	})

	t.Run("provider down keeps the cookie", func(t *testing.T) {
		b := newBrowser(t)
		b.login(t, app, fake, alice, "/dashboard")
		fake.SetAuthenticateFailure(http.StatusBadGateway)
		defer fake.SetAuthenticateFailure(0)

		resp, body := b.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", origin)
		if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "provider_unavailable") {
			t.Errorf("status %d body %s, want 503 provider_unavailable", resp.StatusCode, body)
		}
		if setCookieHeader(resp) != "" {
			t.Errorf("Set-Cookie = %q, want none", setCookieHeader(resp))
		}
		if b.sessionCookie(t, app.BaseURL) == nil {
			t.Error("cookie was dropped during a provider outage")
		}
	})

	t.Run("missing origin", func(t *testing.T) {
		b := newBrowser(t)
		b.login(t, app, fake, alice, "/dashboard")
		calls := fake.AuthenticateCalls()

		resp, body := b.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", nil)
		if resp.StatusCode != http.StatusForbidden || !strings.Contains(body, "forbidden_origin") {
			t.Errorf("status %d body %s, want 403 forbidden_origin", resp.StatusCode, body)
		}
		if fake.AuthenticateCalls() != calls {
			t.Error("origin check ran after the provider was called")
		}
	})

	t.Run("foreign origin", func(t *testing.T) {
		b := newBrowser(t)
		b.login(t, app, fake, alice, "/dashboard")
		resp, _ := b.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", map[string]string{"Origin": "https://evil.example"})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want 403", resp.StatusCode)
		}
	})
}

func TestLogoutRevokesAndClears(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	origin := map[string]string{"Origin": allowedOrigin}
	b := newBrowser(t)
	_, cookie := b.login(t, app, fake, alice, "/dashboard")
	payload, err := testKeyring(t).Open(cookie.Value)
	if err != nil {
		t.Fatalf("opening cookie: %v", err)
	}

	resp, _ := b.do(t, http.MethodPost, app.BaseURL+"/v1/auth/logout", origin)

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", resp.StatusCode)
	}
	if !strings.Contains(setCookieHeader(resp), "Max-Age=0") {
		t.Errorf("Set-Cookie = %q, want a clearing cookie", setCookieHeader(resp))
	}
	if revoked := fake.RevokedSessions(); len(revoked) != 1 || revoked[0] != payload.SessionID {
		t.Errorf("revoked sessions = %v, want [%s]", revoked, payload.SessionID)
	}
	if b.sessionCookie(t, app.BaseURL) != nil {
		t.Error("jar still holds the session cookie after logout")
	}

	old := newBrowser(t)
	old.setCookie(t, app.BaseURL, cookie.Value)
	resp, _ = old.do(t, http.MethodPost, app.BaseURL+"/v1/auth/refresh", origin)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("refresh with the revoked cookie status = %d, want 401", resp.StatusCode)
	}

	resp, _ = newBrowser(t).do(t, http.MethodPost, app.BaseURL+"/v1/auth/logout", origin)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("logout without a cookie status = %d, want 204", resp.StatusCode)
	}

	resp, _ = newBrowser(t).do(t, http.MethodPost, app.BaseURL+"/v1/auth/logout", map[string]string{"Origin": "https://evil.example"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("logout from a foreign origin status = %d, want 403", resp.StatusCode)
	}
}

func TestAuthEndpointsHaveStricterRateLimit(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, map[string]string{"AUTH_RATE_LIMIT_RPS": "1", "AUTH_RATE_LIMIT_BURST": "3"})
	b := newBrowser(t)

	var statuses []int
	for range 4 {
		resp, _ := b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/login?provider=google", nil)
		statuses = append(statuses, resp.StatusCode)
	}
	if statuses[3] != http.StatusTooManyRequests {
		t.Fatalf("fourth login status = %v, want 429 last", statuses)
	}

	token := fake.MintAccessToken(t, alice.ID, "session_x", time.Hour)
	fake.AddUser(alice)
	resp, _ := getWithAuth(t, app.BaseURL+"/v1/me", "Bearer "+token)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/v1/me during an auth burst status = %d, want 200", resp.StatusCode)
	}
}

func TestGoogleRoundTrip(t *testing.T) {
	fake := newFakeWorkOS(t)
	app := startAuthApp(t, fake, nil)
	users := newUserStore(t)

	returning := fakeUser{
		ID:        "user_google_returning",
		Email:     "returning@example.com",
		FirstName: "Cheshire",
		LastName:  "Cat",
		CreatedAt: alice.CreatedAt,
		UpdatedAt: alice.UpdatedAt,
	}
	arriving := fakeUser{
		ID:        "user_google_arriving",
		Email:     "arriving@example.com",
		FirstName: "White",
		LastName:  "Rabbit",
		CreatedAt: alice.CreatedAt,
		UpdatedAt: alice.UpdatedAt,
	}

	google := func(t *testing.T, user fakeUser, returnTo string) (*http.Response, *http.Cookie) {
		t.Helper()
		b := newBrowser(t)

		resp, _ := b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/login?provider=google&returnTo="+url.QueryEscape(returnTo), nil)
		if resp.StatusCode != http.StatusFound {
			t.Fatalf("login status = %d, want 302", resp.StatusCode)
		}
		if !strings.HasPrefix(resp.Header.Get("Location"), fake.Server.URL+"/user_management/authorize?") {
			t.Fatalf("Location = %q, want the provider's authorize endpoint", resp.Header.Get("Location"))
		}
		if got := locationQuery(t, resp).Get("provider"); got != "GoogleOAuth" {
			t.Errorf("provider = %q, want GoogleOAuth", got)
		}
		state := locationQuery(t, resp).Get("state")

		code := fake.IssueCode(user)
		resp, _ = b.do(t, http.MethodGet, app.BaseURL+"/v1/auth/callback?code="+code+"&state="+url.QueryEscape(state), nil)
		return resp, b.sessionCookie(t, app.BaseURL)
	}

	t.Run("returning account holder", func(t *testing.T) {
		google(t, returning, "/dashboard")

		resp, cookie := google(t, returning, "/agreements/1")
		if got := resp.Header.Get("Location"); got != "http://localhost:5173/callback?returnTo=%2Fagreements%2F1" {
			t.Errorf("Location = %q", got)
		}
		if cookie == nil {
			t.Fatal("jar holds no session cookie")
		}
		payload, err := testKeyring(t).Open(cookie.Value)
		if err != nil {
			t.Fatalf("opening cookie: %v", err)
		}
		if payload.UserID != returning.ID {
			t.Errorf("cookie user id = %q, want %q", payload.UserID, returning.ID)
		}

		stored, err := users.Get(t.Context(), returning.ID)
		if err != nil || stored == nil {
			t.Fatalf("user row: %+v (err %v)", stored, err)
		}
		if stored.ID != returning.ID || stored.Email != returning.Email {
			t.Errorf("stored user = %+v, want the same record", stored)
		}
	})

	t.Run("first arrival", func(t *testing.T) {
		if stored, _ := users.Get(t.Context(), arriving.ID); stored != nil {
			t.Fatalf("user row exists before the first sign-in: %+v", stored)
		}

		resp, cookie := google(t, arriving, "/dashboard")
		if resp.StatusCode != http.StatusFound || cookie == nil {
			t.Fatalf("callback status = %d, cookie = %v", resp.StatusCode, cookie)
		}

		stored, err := users.Get(t.Context(), arriving.ID)
		if err != nil || stored == nil {
			t.Fatalf("user row after first sign-in: %+v (err %v)", stored, err)
		}
		if stored.Name != "White Rabbit" || stored.Email != arriving.Email {
			t.Errorf("stored user = %+v, want the provider's name and address", stored)
		}
	})
}
