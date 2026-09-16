package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/workos/workos-go/v10"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/logging"
	"github.com/pixels-two/sow/backend/internal/mail"
	"github.com/pixels-two/sow/backend/internal/session"
)

// AuthService owns the session lifecycle: social login, credential
// exchange, cookie refresh, and logout.
type AuthService struct {
	cfg      config.AppConfig
	keyring  *session.Keyring
	provider WorkOSClient
	users    *UserService
	mailer   mail.Mailer
	now      func() time.Time
}

func NewAuthService(cfg config.AppConfig, keyring *session.Keyring, provider WorkOSClient, users *UserService, mailer mail.Mailer) *AuthService {
	return &AuthService{cfg: cfg, keyring: keyring, provider: provider, users: users, mailer: mailer, now: time.Now}
}

// LoginResult is what a completed code exchange yields.
type LoginResult struct {
	CookieValue string
	ReturnTo    string
	UserID      string
}

// Session is a started session: the sealed cookie value and the access
// token the web app holds in memory.
type Session struct {
	CookieValue string
	AccessToken string
	ExpiresAt   int64
	UserID      string
}

// PendingVerification is the provider's reference for a sign-in that
// waits on an emailed code.
type PendingVerification struct {
	PendingToken string
}

// RefreshResult is what a successful refresh yields.
type RefreshResult struct {
	AccessToken string
	ExpiresAt   int64
	CookieValue string
	UserID      string
}

// LoginURL builds the social provider's authorization URL with a
// signed state.
func (s *AuthService) LoginURL(returnTo string, provider string) (string, error) {
	state, err := session.IssueState(s.keyring.ActiveKey(), returnTo, s.now())
	if err != nil {
		return "", fmt.Errorf("issuing login state: %w", err)
	}
	return s.provider.AuthorizationURL(state, provider), nil
}

// CompleteLogin verifies the state, exchanges the code, provisions the
// user, and seals the session cookie.
func (s *AuthService) CompleteLogin(ctx context.Context, code string, state string, ip string, userAgent string) (LoginResult, error) {
	returnTo, err := session.VerifyState(s.keyring.ActiveKey(), state, s.now())
	if err != nil {
		return LoginResult{}, fmt.Errorf("verifying login state: %w", err)
	}
	if code == "" {
		return LoginResult{}, fmt.Errorf("reading authorization code: %w", common.ErrUnauthorized)
	}

	resp, err := s.provider.AuthenticateWithCode(ctx, code, ip, userAgent)
	if err != nil {
		return LoginResult{}, providerError("exchanging authorization code", err)
	}

	started, err := s.startSession(ctx, resp)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{CookieValue: started.CookieValue, ReturnTo: returnTo, UserID: started.UserID}, nil
}

// SignIn exchanges an email and password with the provider. An
// unverified address yields a pending verification. A wrong password
// and an unknown address yield the same refusal.
func (s *AuthService) SignIn(ctx context.Context, email string, password string, ip string, userAgent string) (Session, *PendingVerification, error) {
	resp, err := s.provider.AuthenticateWithPassword(ctx, strings.TrimSpace(email), password, ip, userAgent)
	if err != nil {
		pending, classified := passwordAuthError("authenticating with password", err)
		if pending != nil {
			return Session{}, pending, nil
		}
		return Session{}, nil, classified
	}

	started, err := s.startSession(ctx, resp)
	if err != nil {
		return Session{}, nil, err
	}
	return started, nil, nil
}

// SignUp creates the account at the provider and starts email
// verification. A provider that requires no verification yields a
// started session. The answer reveals nothing about which addresses are
// registered.
func (s *AuthService) SignUp(ctx context.Context, firstName string, lastName string, email string, password string, ip string, userAgent string) (Session, *PendingVerification, error) {
	email = strings.TrimSpace(email)

	_, err := s.provider.CreateUser(ctx, email, strings.TrimSpace(firstName), strings.TrimSpace(lastName), password)
	if err != nil {
		pending, refusalErr := s.signUpRefusal(ctx, email, err)
		return Session{}, pending, refusalErr
	}

	resp, authErr := s.provider.AuthenticateWithPassword(ctx, email, password, ip, userAgent)
	if authErr == nil {
		started, startErr := s.startSession(ctx, resp)
		return started, nil, startErr
	}

	pending, classified := passwordAuthError("starting email verification", authErr)
	if pending == nil {
		return Session{}, nil, classified
	}
	return Session{}, pending, nil
}

// signUpRefusal answers a provider refusal to create the user. A weak
// password carries the provider's reason. A taken address emails its
// holder and yields a decoy pending verification.
func (s *AuthService) signUpRefusal(ctx context.Context, email string, err error) (*PendingVerification, error) {
	apiErr, refused := providerRefusal(err)
	if !refused {
		return nil, providerFailure("creating provider user", err, common.ErrProviderUnavailable)
	}
	if weakPassword(apiErr) {
		return nil, &common.WeakPasswordError{Message: apiErr.Message}
	}

	s.sendAsync(ctx, mail.AccountExistsMessage(s.cfg.WebAppURL, email), "sending account exists email failed")

	token, tokenErr := decoyToken()
	if tokenErr != nil {
		return nil, tokenErr
	}
	return &PendingVerification{PendingToken: token}, nil
}

// ResendVerification asks the provider to email a fresh code. An
// unknown address and a verified address both leave the provider
// untouched. The send runs on its own goroutine, and every address
// returns from the same point.
func (s *AuthService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.provider.FindUserByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, common.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return providerFailure("finding user by address", err, common.ErrProviderUnavailable)
	}
	if user.EmailVerified {
		return nil
	}

	userID := user.ID
	sendCtx := context.WithoutCancel(ctx)
	go func() {
		sendErr := s.provider.SendVerificationEmail(sendCtx, userID)
		if sendErr == nil {
			return
		}
		failure := providerFailure("resending verification code", sendErr, common.ErrProviderUnavailable)
		logging.Warn(sendCtx, "resending verification code failed",
			common.LogKeyError, failure.Error(),
			common.LogKeyUserID, userID,
		)
	}()
	return nil
}

// ForgotPassword emails a reset link when the provider issues a token
// for the address. An address the provider does not know leaves
// nothing sent. The send runs on its own goroutine, and every address
// returns from the same point.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	email = strings.TrimSpace(email)

	reset, err := s.provider.CreatePasswordReset(ctx, email)
	if err != nil {
		if _, refused := providerRefusal(err); refused {
			return nil
		}
		return providerFailure("creating password reset", err, common.ErrProviderUnavailable)
	}

	expiresAt, err := time.Parse(time.RFC3339, reset.ExpiresAt)
	if err != nil {
		return fmt.Errorf("parsing reset expiry: %w: %w", err, common.ErrProviderUnavailable)
	}

	webAppURL := strings.TrimRight(s.cfg.WebAppURL, "/")
	link := webAppURL + "/reset-password?token=" + url.QueryEscape(reset.PasswordResetToken)
	s.sendAsync(ctx, mail.PasswordResetMessage(webAppURL, email, link, expiresAt),
		"sending password reset email failed", common.LogKeyUserID, reset.UserID)
	return nil
}

// sendAsync sends msg on its own goroutine and returns at once. A
// failed send is logged as a warning under failure with args.
func (s *AuthService) sendAsync(ctx context.Context, msg mail.Message, failure string, args ...any) {
	sendCtx := context.WithoutCancel(ctx)
	go func() {
		err := s.mailer.Send(sendCtx, msg)
		if err == nil {
			return
		}
		logging.Warn(sendCtx, failure, append([]any{common.LogKeyError, err.Error()}, args...)...)
	}()
}

// ResetPassword confirms the reset token, sets the new password, and
// signs the user in. A spent or unknown token yields
// ErrInvalidResetToken.
func (s *AuthService) ResetPassword(ctx context.Context, token string, password string, ip string, userAgent string) (Session, error) {
	confirmed, err := s.provider.ConfirmPasswordReset(ctx, token, password)
	if err != nil {
		return Session{}, resetPasswordError(err)
	}
	if confirmed.User == nil {
		return Session{}, fmt.Errorf("reading reset confirmation: %w", common.ErrProviderUnavailable)
	}

	resp, err := s.provider.AuthenticateWithPassword(ctx, confirmed.User.Email, password, ip, userAgent)
	if err != nil {
		return Session{}, providerFailure("signing in after password reset", err, common.ErrProviderUnavailable)
	}

	return s.startSession(ctx, resp)
}

// VerifyEmail completes a pending sign-in with the code the provider
// emailed. A wrong, expired, or reused code yields ErrInvalidCode.
func (s *AuthService) VerifyEmail(ctx context.Context, pendingToken string, code string, ip string, userAgent string) (Session, error) {
	resp, err := s.provider.AuthenticateWithEmailVerification(ctx, pendingToken, code, ip, userAgent)
	if err != nil {
		return Session{}, verificationError("verifying email code", err)
	}

	started, err := s.startSession(ctx, resp)
	if err != nil {
		return Session{}, err
	}
	return started, nil
}

// Refresh turns the session cookie into a fresh access token and
// re-seals the rotated refresh token.
func (s *AuthService) Refresh(ctx context.Context, cookieValue string, ip string, userAgent string) (RefreshResult, error) {
	payload, err := s.keyring.Open(cookieValue)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("opening session cookie: %w", common.ErrUnauthorized)
	}

	resp, err := s.provider.AuthenticateWithRefreshToken(ctx, payload.RefreshToken, ip, userAgent)
	if err != nil {
		return RefreshResult{}, providerError("refreshing session", err)
	}

	claims, err := parseAccessToken(resp.AccessToken)
	if err != nil {
		return RefreshResult{}, err
	}

	cookie, err := s.sealSession(resp, payload.UserID)
	if err != nil {
		return RefreshResult{}, err
	}
	return RefreshResult{
		AccessToken: resp.AccessToken,
		ExpiresAt:   claims.expiresAt.UnixMilli(),
		CookieValue: cookie,
		UserID:      payload.UserID,
	}, nil
}

// Logout revokes the provider session named in the cookie. An
// unreadable cookie means there is nothing to revoke. A provider failure
// is logged and swallowed so the cookie still clears.
func (s *AuthService) Logout(ctx context.Context, cookieValue string) string {
	payload, err := s.keyring.Open(cookieValue)
	if err != nil || payload.SessionID == "" {
		return ""
	}

	if err := s.provider.RevokeSession(ctx, payload.SessionID); err != nil {
		logging.Warn(ctx, "revoking provider session failed",
			common.LogKeyError, err.Error(),
			common.LogKeyUserID, payload.UserID,
		)
	}
	return payload.UserID
}

// startSession provisions the user record and seals the session
// cookie from a completed provider authentication.
func (s *AuthService) startSession(ctx context.Context, resp *workos.AuthenticateResponse) (Session, error) {
	user, err := s.users.UpsertFromProvider(ctx, resp.User)
	if err != nil {
		return Session{}, fmt.Errorf("provisioning user: %w", err)
	}

	claims, err := parseAccessToken(resp.AccessToken)
	if err != nil {
		return Session{}, err
	}

	cookie, err := s.sealSession(resp, user.ID)
	if err != nil {
		return Session{}, err
	}
	return Session{
		CookieValue: cookie,
		AccessToken: resp.AccessToken,
		ExpiresAt:   claims.expiresAt.UnixMilli(),
		UserID:      user.ID,
	}, nil
}

func (s *AuthService) sealSession(resp *workos.AuthenticateResponse, userID string) (string, error) {
	claims, err := parseAccessToken(resp.AccessToken)
	if err != nil {
		return "", err
	}

	cookie, err := s.keyring.Seal(session.Payload{
		RefreshToken: resp.RefreshToken,
		SessionID:    claims.sessionID,
		UserID:       userID,
		IssuedAt:     s.now().UnixMilli(),
	})
	if err != nil {
		return "", fmt.Errorf("sealing session cookie: %w", err)
	}
	return cookie, nil
}

type accessTokenClaims struct {
	subject   string
	sessionID string
	expiresAt time.Time
}

// parseAccessToken reads the claims the session needs. The token came
// from the provider over TLS, so no signature check happens here. The
// bearer middleware verifies it on every later request.
func parseAccessToken(raw string) (accessTokenClaims, error) {
	token, err := jwt.ParseInsecure([]byte(raw))
	if err != nil {
		return accessTokenClaims{}, fmt.Errorf("parsing provider access token: %w: %w", err, common.ErrProviderUnavailable)
	}

	expiresAt, ok := token.Expiration()
	if !ok {
		return accessTokenClaims{}, fmt.Errorf("provider access token has no expiry: %w", common.ErrProviderUnavailable)
	}

	subject, _ := token.Subject()
	var sessionID string
	_ = token.Get("sid", &sessionID)

	return accessTokenClaims{subject: subject, sessionID: sessionID, expiresAt: expiresAt}, nil
}

// providerError classifies a provider failure. A 4xx answer means the
// provider refused the credential. Anything else means the provider was
// unavailable.
func providerError(operation string, err error) error {
	if _, refused := providerRefusal(err); refused {
		return providerFailure(operation, err, common.ErrUnauthorized)
	}
	return providerFailure(operation, err, common.ErrProviderUnavailable)
}

// pendingTokenLength is the character count of a decoy pending token.
// It mirrors the length of a provider pending authentication token.
const pendingTokenLength = 43

// decoyToken builds an opaque token shaped like the provider's own. A
// sign-up for an address that already has an account answers with one.
func decoyToken() (string, error) {
	raw := make([]byte, pendingTokenLength)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating pending token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)[:pendingTokenLength], nil
}

// passwordAuthError classifies a failed password authentication. An
// answer that names an unverified email carries the pending token the
// code step needs, regardless of its status. A refusal that names the
// password method as disabled reads as the environment being
// misconfigured. Any other provider refusal reads as a credential
// mismatch.
func passwordAuthError(operation string, err error) (*PendingVerification, error) {
	var apiErr *workos.APIError
	if errors.As(err, &apiErr) && providerCode(apiErr) == workos.EmailVerificationRequiredCode && apiErr.PendingAuthenticationToken != "" {
		return &PendingVerification{PendingToken: apiErr.PendingAuthenticationToken}, nil
	}
	refusedErr, refused := providerRefusal(err)
	if !refused {
		return nil, providerFailure(operation, err, common.ErrProviderUnavailable)
	}
	if providerCode(refusedErr) == workos.SSORequiredCode {
		return nil, providerFailure(operation, err, common.ErrProviderUnavailable)
	}
	return nil, providerFailure(operation, err, common.ErrInvalidCredentials)
}

// resetPasswordError classifies a refused reset confirmation. A weak
// password carries the provider's reason. Any other refusal reads as a
// spent or unknown token.
func resetPasswordError(err error) error {
	apiErr, refused := providerRefusal(err)
	if !refused {
		return providerFailure("confirming password reset", err, common.ErrProviderUnavailable)
	}
	if weakPassword(apiErr) {
		return &common.WeakPasswordError{Message: apiErr.Message}
	}
	return providerFailure("confirming password reset", err, common.ErrInvalidResetToken)
}

// verificationError classifies a failed code exchange. A provider
// refusal reads as a rejected code.
func verificationError(operation string, err error) error {
	if _, refused := providerRefusal(err); refused {
		return providerFailure(operation, err, common.ErrInvalidCode)
	}
	return providerFailure(operation, err, common.ErrProviderUnavailable)
}

// weakPassword reports whether the provider refused a password for its
// strength.
func weakPassword(apiErr *workos.APIError) bool {
	return strings.Contains(providerCode(apiErr), "password_strength")
}

// providerRefusal reports whether the provider refused the submitted
// values and returns that answer. Transport failures, 5xx answers, and
// answers below 400 are not refusals. Answers of 401, 403, 408, and 429
// name the platform's own credentials, latency, or quota, and are not
// refusals.
func providerRefusal(err error) (*workos.APIError, bool) {
	var apiErr *workos.APIError
	if !errors.As(err, &apiErr) {
		return nil, false
	}
	if apiErr.StatusCode < http.StatusBadRequest || apiErr.StatusCode >= http.StatusInternalServerError {
		return nil, false
	}

	switch apiErr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusRequestTimeout, http.StatusTooManyRequests:
		return nil, false
	default:
		return apiErr, true
	}
}

// providerFailure builds the error a failed provider call becomes. An
// answer from the provider contributes its status and code. Diverging
// code fields contribute both. Any other failure carries its own
// message.
func providerFailure(operation string, err error, sentinel error) error {
	var apiErr *workos.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Code != apiErr.ErrorCode {
			return fmt.Errorf("%s: provider status %d code %q error %q: %w", operation, apiErr.StatusCode, apiErr.Code, apiErr.ErrorCode, sentinel)
		}
		return fmt.Errorf("%s: provider status %d code %q: %w", operation, apiErr.StatusCode, providerCode(apiErr), sentinel)
	}
	return fmt.Errorf("%s: %w: %w", operation, err, sentinel)
}

// providerCode names the error code a provider answer carries. An
// OAuth-style answer names it in its own field.
func providerCode(apiErr *workos.APIError) string {
	if apiErr.Code != "" {
		return apiErr.Code
	}
	return apiErr.ErrorCode
}
