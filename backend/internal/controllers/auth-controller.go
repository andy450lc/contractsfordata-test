package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/logging"
	"github.com/pixels-two/sow/backend/internal/models/dto"
	"github.com/pixels-two/sow/backend/internal/services"
	"github.com/pixels-two/sow/backend/internal/session"
)

// AuthController serves the session endpoints. The cookie is set and
// cleared here and read nowhere else.
type AuthController struct {
	auth      *services.AuthService
	webAppURL string
	maxAge    time.Duration
	secure    bool
}

func NewAuthController(auth *services.AuthService, cfg config.AppConfig) *AuthController {
	return &AuthController{
		auth:      auth,
		webAppURL: strings.TrimRight(cfg.WebAppURL, "/"),
		maxAge:    time.Duration(cfg.SessionMaxAgeSeconds) * time.Second,
		secure:    cfg.Deployed(),
	}
}

// Login redirects the browser to the social provider's own sign-in
// screen.
func (ac *AuthController) Login(c *echo.Context) error {
	query := c.Get(common.EchoContextKeyValidatedDTO).(*dto.LoginQueryDto)

	target, err := ac.auth.LoginURL(query.ReturnTo, query.Provider)
	if err != nil {
		return fmt.Errorf("building login url: %w", err)
	}

	return c.Redirect(http.StatusFound, target)
}

// SignIn exchanges an email and password for a session. An unverified
// address answers 202 with the pending token the code step needs.
func (ac *AuthController) SignIn(c *echo.Context) error {
	ctx := c.Request().Context()
	body := c.Get(common.EchoContextKeyValidatedDTO).(*dto.SignInDto)

	started, pending, err := ac.auth.SignIn(ctx, body.Email, body.Password, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return fmt.Errorf("signing in: %w", err)
	}

	if pending != nil {
		return c.JSON(http.StatusAccepted, dto.PendingVerification{PendingToken: pending.PendingToken})
	}
	return ac.respondSession(c, started)
}

// SignUp creates an account and starts email verification. The answer
// is the same for a free address and a taken one. A provider that
// requires no verification answers 200 with a session.
func (ac *AuthController) SignUp(c *echo.Context) error {
	ctx := c.Request().Context()
	body := c.Get(common.EchoContextKeyValidatedDTO).(*dto.SignUpDto)

	started, pending, err := ac.auth.SignUp(ctx, body.FirstName, body.LastName, body.Email, body.Password, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return fmt.Errorf("signing up: %w", err)
	}

	if pending != nil {
		return c.JSON(http.StatusAccepted, dto.PendingVerification{PendingToken: pending.PendingToken})
	}
	return ac.respondSession(c, started)
}

// ResendVerification emails a fresh code. Every well-formed address
// gets the same answer.
func (ac *AuthController) ResendVerification(c *echo.Context) error {
	ctx := c.Request().Context()
	body := c.Get(common.EchoContextKeyValidatedDTO).(*dto.EmailDto)

	if err := ac.auth.ResendVerification(ctx, body.Email); err != nil {
		return fmt.Errorf("resending verification: %w", err)
	}

	return c.NoContent(http.StatusAccepted)
}

// VerifyEmail completes a pending sign-in with the emailed code.
func (ac *AuthController) VerifyEmail(c *echo.Context) error {
	ctx := c.Request().Context()
	body := c.Get(common.EchoContextKeyValidatedDTO).(*dto.VerifyEmailDto)

	started, err := ac.auth.VerifyEmail(ctx, body.PendingToken, body.Code, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return fmt.Errorf("verifying email: %w", err)
	}

	return ac.respondSession(c, started)
}

// ForgotPassword emails a reset link. Every well-formed address gets
// the same answer.
func (ac *AuthController) ForgotPassword(c *echo.Context) error {
	ctx := c.Request().Context()
	body := c.Get(common.EchoContextKeyValidatedDTO).(*dto.EmailDto)

	if err := ac.auth.ForgotPassword(ctx, body.Email); err != nil {
		return fmt.Errorf("starting password reset: %w", err)
	}

	return c.NoContent(http.StatusAccepted)
}

// ResetPassword sets a new password from a reset link and signs the
// user in.
func (ac *AuthController) ResetPassword(c *echo.Context) error {
	ctx := c.Request().Context()
	body := c.Get(common.EchoContextKeyValidatedDTO).(*dto.ResetPasswordDto)

	started, err := ac.auth.ResetPassword(ctx, body.Token, body.Password, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return fmt.Errorf("resetting password: %w", err)
	}

	return ac.respondSession(c, started)
}

// Callback completes the social flow. Success sets the cookie and sends
// the browser to the web app's callback page. Every failure clears the
// cookie and sends the browser to the sign-in page with an error code.
func (ac *AuthController) Callback(c *echo.Context) error {
	ctx := c.Request().Context()
	query := c.Get(common.EchoContextKeyValidatedDTO).(*dto.CallbackQueryDto)

	result, err := ac.auth.CompleteLogin(ctx, query.Code, query.State, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		c.SetCookie(session.ClearCookie(ac.secure))
		return c.Redirect(http.StatusFound, ac.webAppURL+"/?error="+callbackErrorCode(ctx, err))
	}

	ac.attachUser(c, result.UserID)
	c.SetCookie(session.NewCookie(result.CookieValue, ac.maxAge, ac.secure))
	return c.Redirect(http.StatusFound, ac.webAppURL+"/callback?returnTo="+url.QueryEscape(result.ReturnTo))
}

// Refresh exchanges the cookie for a fresh access token.
func (ac *AuthController) Refresh(c *echo.Context) error {
	ctx := c.Request().Context()

	cookie, err := c.Request().Cookie(common.SessionCookieName)
	if err != nil {
		c.SetCookie(session.ClearCookie(ac.secure))
		return fmt.Errorf("reading session cookie: %w", common.ErrUnauthorized)
	}

	result, err := ac.auth.Refresh(ctx, cookie.Value, c.RealIP(), c.Request().UserAgent())
	if errors.Is(err, common.ErrUnauthorized) {
		c.SetCookie(session.ClearCookie(ac.secure))
		return fmt.Errorf("refreshing session: %w", err)
	}
	if err != nil {
		return fmt.Errorf("refreshing session: %w", err)
	}

	ac.attachUser(c, result.UserID)
	c.SetCookie(session.NewCookie(result.CookieValue, ac.maxAge, ac.secure))
	return c.JSON(http.StatusOK, dto.AccessToken{
		AccessToken: result.AccessToken,
		ExpiresAt:   result.ExpiresAt,
	})
}

// Logout revokes the provider session and clears the cookie. Sign-out
// is idempotent. A missing cookie still answers 204.
func (ac *AuthController) Logout(c *echo.Context) error {
	ctx := c.Request().Context()

	if cookie, err := c.Request().Cookie(common.SessionCookieName); err == nil {
		userID := ac.auth.Logout(ctx, cookie.Value)
		ac.attachUser(c, userID)
	}

	c.SetCookie(session.ClearCookie(ac.secure))
	return c.NoContent(http.StatusNoContent)
}

// respondSession seals a started session into the cookie and answers
// with the access token the web app holds in memory.
func (ac *AuthController) respondSession(c *echo.Context, started services.Session) error {
	ac.attachUser(c, started.UserID)
	c.SetCookie(session.NewCookie(started.CookieValue, ac.maxAge, ac.secure))
	return c.JSON(http.StatusOK, dto.AccessToken{
		AccessToken: started.AccessToken,
		ExpiresAt:   started.ExpiresAt,
	})
}

// attachUser puts the user id on the request context so the access log
// line for this request carries it.
func (ac *AuthController) attachUser(c *echo.Context, userID string) {
	if userID == "" {
		return
	}
	ctx := logging.ContextWithUserID(c.Request().Context(), userID)
	c.SetRequest(c.Request().WithContext(ctx))
}

// callbackErrorCode maps a failed code exchange to the error code the
// sign-in page understands and writes the single log line for it.
func callbackErrorCode(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, common.ErrInvalidState):
		logging.Warn(ctx, "login callback rejected", common.LogKeyError, err.Error())
		return "invalid_state"
	case errors.Is(err, common.ErrProviderUnavailable):
		logging.Warn(ctx, "login callback failed at provider", common.LogKeyError, err.Error())
		return "provider_unavailable"
	case errors.Is(err, common.ErrUnauthorized):
		logging.Warn(ctx, "login callback refused", common.LogKeyError, err.Error())
		return "flow_incomplete"
	default:
		logging.Error(ctx, "login callback failed unexpectedly", common.LogKeyError, fmt.Sprintf("%+v", err))
		return "flow_incomplete"
	}
}
