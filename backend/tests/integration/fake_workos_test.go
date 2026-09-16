package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// fakeUser is a person the fake provider knows about, keyed by email
// for the password and email-verification flows and by id everywhere
// else.
type fakeUser struct {
	ID            string
	Email         string
	FirstName     string
	LastName      string
	Password      string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type fakeSession struct {
	userID    string
	sessionID string
}

// fakeWorkOS stands in for the WorkOS API and its JWKS. It exchanges
// codes it issued, rotates refresh tokens, revokes sessions, serves users,
// and mints RS256 access tokens the real auth middleware validates.
type fakeWorkOS struct {
	Server *httptest.Server

	key  jwk.Key
	jwks []byte

	mu                sync.Mutex
	codes             map[string]fakeUser
	refreshTokens     map[string]fakeSession
	users             map[string]fakeUser
	pendingAuth       map[string]string // pending_authentication_token -> user id
	verificationCodes map[string]string // user id -> current verification code
	sentCodes         map[string]int    // user id -> verification emails sent
	resetTokens       map[string]string // reset token -> email
	lastResetToken    map[string]string // email -> most recent reset token
	revoked           []string
	failStatus        int
	failBody          map[string]any
	createUserStatus  int
	listUsersStatus   int
	sendCodeStatus    int
	resetStatus       int
	createsVerified   bool
	resetExpiryText   string
	sendGate          chan struct{}
	seq               int
	authCalls         int
	accessTTL         time.Duration
	holdAuthenticate  time.Duration
}

// fakePendingTokenLength is the character count of the pending
// authentication tokens this fake issues.
const fakePendingTokenLength = 43

func newFakeWorkOS(t *testing.T) *fakeWorkOS {
	t.Helper()

	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	key, err := jwk.Import(raw)
	if err != nil {
		t.Fatalf("importing key: %v", err)
	}
	if err := key.Set(jwk.KeyIDKey, "fake-key-1"); err != nil {
		t.Fatalf("setting kid: %v", err)
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		t.Fatalf("setting alg: %v", err)
	}

	private := jwk.NewSet()
	if err := private.AddKey(key); err != nil {
		t.Fatalf("adding key: %v", err)
	}
	public, err := jwk.PublicSetOf(private)
	if err != nil {
		t.Fatalf("deriving public set: %v", err)
	}
	jwks, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("marshaling JWKS: %v", err)
	}

	f := &fakeWorkOS{
		key:               key,
		jwks:              jwks,
		codes:             map[string]fakeUser{},
		refreshTokens:     map[string]fakeSession{},
		users:             map[string]fakeUser{},
		pendingAuth:       map[string]string{},
		verificationCodes: map[string]string{},
		sentCodes:         map[string]int{},
		resetTokens:       map[string]string{},
		lastResetToken:    map[string]string{},
		accessTTL:         5 * time.Minute,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /user_management/authenticate", f.authenticate)
	mux.HandleFunc("POST /user_management/sessions/revoke", f.revoke)
	mux.HandleFunc("GET /user_management/users/{id}", f.getUser)
	mux.HandleFunc("POST /user_management/users", f.createUser)
	mux.HandleFunc("GET /user_management/users", f.listUsers)
	mux.HandleFunc("POST /user_management/users/{id}/email_verification/send", f.sendVerificationEmail)
	mux.HandleFunc("POST /user_management/password_reset", f.createPasswordReset)
	mux.HandleFunc("POST /user_management/password_reset/confirm", f.confirmPasswordReset)
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(f.jwks)
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Server.Close)
	return f
}

// JWKSURL is where the app fetches verification keys.
func (f *fakeWorkOS) JWKSURL() string { return f.Server.URL + "/jwks" }

// AddUser makes a user known to the provider without a session.
func (f *fakeWorkOS) AddUser(u fakeUser) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[u.ID] = u
}

// IssueCode returns a one-time authorization code for the user.
func (f *fakeWorkOS) IssueCode(u fakeUser) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	code := fmt.Sprintf("code_%d", f.seq)
	f.codes[code] = u
	f.users[u.ID] = u
	return code
}

// SetAuthenticateFailure makes every authenticate call answer status
// with a generic error body. Zero restores normal behavior.
func (f *fakeWorkOS) SetAuthenticateFailure(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failStatus = status
	f.failBody = nil
}

// SetAuthenticateFailureBody makes every authenticate call answer
// status with body. Zero status restores normal behavior.
func (f *fakeWorkOS) SetAuthenticateFailureBody(status int, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failStatus = status
	f.failBody = body
}

// SetCreateUserFailure makes every create-user call answer status. Zero
// restores normal behavior.
func (f *fakeWorkOS) SetCreateUserFailure(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createUserStatus = status
}

// SetListUsersFailure makes every user-list call answer status. Zero
// restores normal behavior.
func (f *fakeWorkOS) SetListUsersFailure(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listUsersStatus = status
}

// SetSendVerificationFailure makes every verification-send call answer
// status. Zero restores normal behavior.
func (f *fakeWorkOS) SetSendVerificationFailure(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCodeStatus = status
}

// SetPasswordResetFailure makes both password-reset calls answer
// status. Zero restores normal behavior.
func (f *fakeWorkOS) SetPasswordResetFailure(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resetStatus = status
}

// SetCreatesVerifiedUsers makes create-user mark new users verified, as
// a provider with email verification switched off does.
func (f *fakeWorkOS) SetCreatesVerifiedUsers(on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createsVerified = on
}

// SetResetExpiryText makes every password-reset answer state text as
// its expiry. An empty string restores a real timestamp.
func (f *fakeWorkOS) SetResetExpiryText(text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resetExpiryText = text
}

// GateSendVerification holds every later verification send and returns
// the call that opens the gate. A held send gives up after two seconds.
func (f *fakeWorkOS) GateSendVerification() func() {
	f.mu.Lock()
	defer f.mu.Unlock()

	gate := make(chan struct{})
	f.sendGate = gate
	return sync.OnceFunc(func() { close(gate) })
}

// HoldAuthenticate makes every later authenticate call sleep d before
// it answers. Zero restores normal behavior.
func (f *fakeWorkOS) HoldAuthenticate(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.holdAuthenticate = d
}

// awaitAuthenticateHold sleeps for the duration a test set on the
// authenticate endpoint. Callers hold no lock.
func (f *fakeWorkOS) awaitAuthenticateHold() {
	f.mu.Lock()
	hold := f.holdAuthenticate
	f.mu.Unlock()

	if hold > 0 {
		time.Sleep(hold)
	}
}

// awaitSendGate waits for an open send gate. Callers hold no lock.
func (f *fakeWorkOS) awaitSendGate() {
	f.mu.Lock()
	gate := f.sendGate
	f.mu.Unlock()

	if gate == nil {
		return
	}
	select {
	case <-gate:
	case <-time.After(2 * time.Second):
	}
}

// forced writes the status a test pinned on an endpoint and reports
// whether it wrote one. The caller holds f.mu.
func (f *fakeWorkOS) forced(w http.ResponseWriter, status int) bool {
	if status == 0 {
		return false
	}
	writeJSON(w, status, map[string]string{"code": "forced_failure", "message": "forced failure"})
	return true
}

// pendingTokenFor builds a pending authentication token as long as a
// real one.
func pendingTokenFor(seq int) string {
	token := fmt.Sprintf("pat_%d", seq)
	return token + strings.Repeat("x", fakePendingTokenLength-len(token))
}

// RevokedSessions lists the session ids revoked so far.
func (f *fakeWorkOS) RevokedSessions() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.revoked...)
}

// AuthenticateCalls counts authenticate requests received.
func (f *fakeWorkOS) AuthenticateCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.authCalls
}

// SentCodes counts the verification emails sent for a user id.
func (f *fakeWorkOS) SentCodes(userID string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sentCodes[userID]
}

// WaitForSentCodes waits for n verification emails to a user id. The
// test fails when they do not arrive within two seconds.
func (f *fakeWorkOS) WaitForSentCodes(t *testing.T, userID string, n int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for f.SentCodes(userID) < n {
		if time.Now().After(deadline) {
			t.Fatalf("sent %d verification emails for %s, want %d", f.SentCodes(userID), userID, n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// CurrentCode returns the verification code on file for an email
// address, or "" when the address is unknown.
func (f *fakeWorkOS) CurrentCode(email string) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	user, ok := f.userByEmail(email)
	if !ok {
		return ""
	}
	return f.verificationCodes[user.ID]
}

// LastResetToken returns the most recently issued password reset
// token for an email address, or "" when none was issued.
func (f *fakeWorkOS) LastResetToken(email string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastResetToken[email]
}

// PasswordOf returns the password on file for an email address.
func (f *fakeWorkOS) PasswordOf(email string) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	user, _ := f.userByEmail(email)
	return user.Password
}

// UserOf returns the user the provider holds for an address. The
// second result reports whether the address is known.
func (f *fakeWorkOS) UserOf(email string) (fakeUser, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.userByEmail(email)
}

// userByEmail finds a user by address. Callers hold f.mu.
func (f *fakeWorkOS) userByEmail(email string) (fakeUser, bool) {
	for _, u := range f.users {
		if u.Email == email {
			return u, true
		}
	}
	return fakeUser{}, false
}

// MintAccessToken signs an access token the app's JWKS lookup accepts.
func (f *fakeWorkOS) MintAccessToken(t *testing.T, userID string, sessionID string, ttl time.Duration) string {
	t.Helper()
	return f.mint(userID, sessionID, ttl)
}

func (f *fakeWorkOS) mint(userID string, sessionID string, ttl time.Duration) string {
	now := time.Now()
	builder := jwt.NewBuilder().
		Issuer("https://api.workos.com").
		IssuedAt(now.Add(-time.Minute)).
		Expiration(now.Add(ttl)).
		Claim("sid", sessionID)
	if userID != "" {
		builder = builder.Subject(userID)
	}
	token, err := builder.Build()
	if err != nil {
		panic(err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), f.key))
	if err != nil {
		panic(err)
	}
	return string(signed)
}

func (f *fakeWorkOS) authenticate(w http.ResponseWriter, r *http.Request) {
	f.awaitAuthenticateHold()

	f.mu.Lock()
	defer f.mu.Unlock()
	f.authCalls++

	if f.failStatus != 0 {
		body := f.failBody
		if body == nil {
			body = map[string]any{"error": "invalid_grant", "error_description": "forced failure"}
		}
		writeJSON(w, f.failStatus, body)
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	var session fakeSession
	var ok bool
	switch body["grant_type"] {
	case "authorization_code":
		session, ok = f.authenticateWithCode(w, body)
	case "refresh_token":
		session, ok = f.authenticateWithRefreshToken(w, body)
	case "password":
		session, ok = f.authenticateWithPassword(w, body)
	case "urn:workos:oauth:grant-type:email-verification:code":
		session, ok = f.authenticateWithEmailVerification(w, body)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
		return
	}
	if !ok {
		return
	}

	f.seq++
	refreshToken := fmt.Sprintf("rt_%d", f.seq)
	f.refreshTokens[refreshToken] = session
	writeJSON(w, http.StatusOK, map[string]any{
		"user":          userJSON(f.users[session.userID]),
		"access_token":  f.mint(session.userID, session.sessionID, f.accessTTL),
		"refresh_token": refreshToken,
	})
}

// authenticateWithCode redeems a one-time authorization code. The
// caller holds f.mu.
func (f *fakeWorkOS) authenticateWithCode(w http.ResponseWriter, body map[string]any) (fakeSession, bool) {
	code, _ := body["code"].(string)
	user, ok := f.codes[code]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "unknown code"})
		return fakeSession{}, false
	}
	delete(f.codes, code)
	f.seq++
	return fakeSession{userID: user.ID, sessionID: fmt.Sprintf("session_%d", f.seq)}, true
}

// authenticateWithRefreshToken rotates a refresh token. The caller
// holds f.mu.
func (f *fakeWorkOS) authenticateWithRefreshToken(w http.ResponseWriter, body map[string]any) (fakeSession, bool) {
	token, _ := body["refresh_token"].(string)
	existing, ok := f.refreshTokens[token]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "unknown refresh token"})
		return fakeSession{}, false
	}
	delete(f.refreshTokens, token)
	return existing, true
}

// authenticateWithPassword checks an email and password. An unverified
// user answers 403 with a pending authentication token and no session.
// The caller holds f.mu.
func (f *fakeWorkOS) authenticateWithPassword(w http.ResponseWriter, body map[string]any) (fakeSession, bool) {
	email, _ := body["email"].(string)
	password, _ := body["password"].(string)

	user, found := f.userByEmail(email)
	if !found || user.Password != password {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_credentials"})
		return fakeSession{}, false
	}

	if !user.EmailVerified {
		f.seq++
		token := pendingTokenFor(f.seq)
		f.pendingAuth[token] = user.ID
		if f.verificationCodes[user.ID] == "" {
			f.verificationCodes[user.ID] = "123456"
		}
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":                         "email_verification_required",
			"pending_authentication_token": token,
			"email_verification_id":        fmt.Sprintf("ev_%d", f.seq),
		})
		return fakeSession{}, false
	}

	f.seq++
	return fakeSession{userID: user.ID, sessionID: fmt.Sprintf("session_%d", f.seq)}, true
}

// authenticateWithEmailVerification redeems a pending authentication
// token and a code, marking the user verified. The caller holds f.mu.
func (f *fakeWorkOS) authenticateWithEmailVerification(w http.ResponseWriter, body map[string]any) (fakeSession, bool) {
	token, _ := body["pending_authentication_token"].(string)
	code, _ := body["code"].(string)

	userID, ok := f.pendingAuth[token]
	if !ok || f.verificationCodes[userID] != code {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_one_time_code"})
		return fakeSession{}, false
	}
	delete(f.pendingAuth, token)

	user := f.users[userID]
	user.EmailVerified = true
	f.users[userID] = user

	f.seq++
	return fakeSession{userID: userID, sessionID: fmt.Sprintf("session_%d", f.seq)}, true
}

func (f *fakeWorkOS) revoke(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var body struct {
		SessionID string `json:"session_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	f.revoked = append(f.revoked, body.SessionID)
	for token, session := range f.refreshTokens {
		if session.sessionID == body.SessionID {
			delete(f.refreshTokens, token)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (f *fakeWorkOS) getUser(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	user, ok := f.users[r.PathValue("id")]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"code": "entity_not_found", "message": "User not found"})
		return
	}
	writeJSON(w, http.StatusOK, userJSON(user))
}

// createUser rejects a taken email or a password containing "weak",
// else creates an unverified user with a seeded verification code.
func (f *fakeWorkOS) createUser(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.forced(w, f.createUserStatus) {
		return
	}

	var body struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	if _, ok := f.userByEmail(body.Email); ok {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"code":   "user_creation_error",
			"errors": []map[string]string{{"code": "email_not_available"}},
		})
		return
	}
	if strings.Contains(body.Password, "weak") {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"code":    "password_strength_error",
			"message": "Password is too weak",
		})
		return
	}

	f.seq++
	now := time.Now()
	user := fakeUser{
		ID:            fmt.Sprintf("user_%d", f.seq),
		Email:         body.Email,
		FirstName:     body.FirstName,
		LastName:      body.LastName,
		Password:      body.Password,
		EmailVerified: f.createsVerified,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	f.users[user.ID] = user
	if !f.createsVerified {
		f.verificationCodes[user.ID] = "123456"
	}
	writeJSON(w, http.StatusCreated, userJSON(user))
}

// listUsers answers the email-filtered list the SDK's List call
// issues. Only the filter this fake needs is implemented.
func (f *fakeWorkOS) listUsers(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.forced(w, f.listUsersStatus) {
		return
	}

	data := []map[string]any{}
	if user, ok := f.userByEmail(r.URL.Query().Get("email")); ok {
		data = append(data, userJSON(user))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":          data,
		"list_metadata": map[string]any{"before": nil, "after": nil},
	})
}

// sendVerificationEmail rotates the user's code and counts the send.
func (f *fakeWorkOS) sendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	f.awaitSendGate()

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.forced(w, f.sendCodeStatus) {
		return
	}

	id := r.PathValue("id")
	if _, ok := f.users[id]; !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"code": "entity_not_found", "message": "User not found"})
		return
	}

	f.seq++
	f.verificationCodes[id] = fmt.Sprintf("%06d", f.seq)
	f.sentCodes[id]++
	writeJSON(w, http.StatusOK, map[string]any{})
}

// createPasswordReset issues a reset token for a known address.
func (f *fakeWorkOS) createPasswordReset(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.forced(w, f.resetStatus) {
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	user, ok := f.userByEmail(body.Email)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"code": "entity_not_found", "message": "User not found"})
		return
	}

	f.seq++
	token := fmt.Sprintf("prt_%d", f.seq)
	f.resetTokens[token] = user.Email
	f.lastResetToken[user.Email] = token

	expiresAt := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339Nano)
	if f.resetExpiryText != "" {
		expiresAt = f.resetExpiryText
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"password_reset_token": token,
		"password_reset_url":   f.Server.URL + "/reset-password?token=" + token,
		"expires_at":           expiresAt,
	})
}

// confirmPasswordReset consumes a reset token, sets the new password,
// and verifies the email as WorkOS does on a successful reset.
func (f *fakeWorkOS) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.forced(w, f.resetStatus) {
		return
	}

	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	email, ok := f.resetTokens[body.Token]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_password_reset_token"})
		return
	}
	user, ok := f.userByEmail(email)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_password_reset_token"})
		return
	}
	if strings.Contains(body.NewPassword, "weak") {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"code":    "password_strength_error",
			"message": "Password is too weak",
		})
		return
	}
	delete(f.resetTokens, body.Token)

	user.Password = body.NewPassword
	user.EmailVerified = true
	f.users[user.ID] = user

	writeJSON(w, http.StatusOK, map[string]any{"user": userJSON(user)})
}

func userJSON(u fakeUser) map[string]any {
	return map[string]any{
		"object":         "user",
		"id":             u.ID,
		"email":          u.Email,
		"first_name":     u.FirstName,
		"last_name":      u.LastName,
		"email_verified": u.EmailVerified,
		"created_at":     u.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated_at":     u.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
