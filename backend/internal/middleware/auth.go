package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/logging"
)

// jwksRegisterTimeout bounds the initial JWKS registration and fetch.
const jwksRegisterTimeout = 10 * time.Second

// AuthMiddleware validates bearer access tokens against the identity
// provider's JWKS and attaches the authenticated user id to the request
// context.
type AuthMiddleware struct {
	jwksURL string

	mu         sync.Mutex
	cache      *jwk.Cache
	registered bool
	cancel     context.CancelFunc
}

// NewAuthMiddleware builds the middleware for the given JWKS URL. The
// JWKS is fetched on first use and cached with background refresh.
func NewAuthMiddleware(jwksURL string) *AuthMiddleware {
	return &AuthMiddleware{jwksURL: jwksURL}
}

// Close stops the background JWKS refresh.
func (m *AuthMiddleware) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// keySet returns the cached JWKS. The first call registers the URL and
// fetches it. A failed fetch is retried on the next call.
func (m *AuthMiddleware) keySet(ctx context.Context) (jwk.Set, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cache == nil {
		cacheCtx, cancel := context.WithCancel(context.Background())
		cache, err := jwk.NewCache(cacheCtx, httprc.NewClient())
		if err != nil {
			cancel()
			return nil, fmt.Errorf("creating JWKS cache: %w", err)
		}
		m.cache = cache
		m.cancel = cancel
	}

	if !m.registered {
		if err := m.register(ctx); err != nil {
			return nil, err
		}
		m.registered = true
	}

	set, err := m.cache.Lookup(ctx, m.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("looking up JWKS: %w", err)
	}
	return set, nil
}

// register adds the JWKS URL to the cache and waits for the first
// fetch. The wait runs on a context that ignores request cancellation,
// so an aborted request cannot leave the shared cache half registered.
// A URL left behind by an earlier failed attempt counts as registered.
func (m *AuthMiddleware) register(ctx context.Context) error {
	registerCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), jwksRegisterTimeout)
	defer cancel()

	err := m.cache.Register(registerCtx, m.jwksURL)
	if err != nil && !errors.Is(err, httprc.ErrResourceAlreadyExists()) {
		return fmt.Errorf("fetching JWKS: %w", err)
	}
	return nil
}

// Authenticate validates the request's bearer access token and adds the
// authenticated user id to the request context.
func (m *AuthMiddleware) Authenticate(c *echo.Context) error {
	ctx := c.Request().Context()

	raw, ok := bearerToken(c.Request().Header.Get("Authorization"))
	if !ok {
		return fmt.Errorf("reading bearer token: %w", common.ErrUnauthorized)
	}

	set, err := m.keySet(ctx)
	if err != nil {
		return fmt.Errorf("loading token verification keys: %w", err)
	}

	token, err := jwt.Parse([]byte(raw),
		jwt.WithKeySet(set),
		jwt.WithValidate(true),
		jwt.WithAcceptableSkew(30*time.Second),
	)
	if err != nil {
		return fmt.Errorf("validating access token: %w", common.ErrUnauthorized)
	}

	userID, ok := token.Subject()
	if !ok || userID == "" {
		return fmt.Errorf("reading token subject: %w", common.ErrUnauthorized)
	}

	ctx = context.WithValue(ctx, common.ContextKeyUserID, userID)
	ctx = logging.ContextWithUserID(ctx, userID)
	c.SetRequest(c.Request().WithContext(ctx))

	return nil
}

// RequireSession authenticates the request before invoking the next handler.
func (m *AuthMiddleware) RequireSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if err := m.Authenticate(c); err != nil {
			return err
		}
		return next(c)
	}
}

// UserID returns the authenticated user id stored by the auth middleware.
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(common.ContextKeyUserID).(string)
	return id, ok && id != ""
}

func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	return token, true
}
