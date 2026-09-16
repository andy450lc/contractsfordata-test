package boot

import (
	"fmt"
	"net/http"
	"time"

	"github.com/workos/workos-go/v10"
	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/middleware"
	"github.com/pixels-two/sow/backend/internal/session"
)

// NewWorkOSClient builds the WorkOS API client from config. A base URL
// override points it at a test double. The client's timeout comes from
// WORKOS_TIMEOUT_SECONDS. It bounds every call, including one held by
// a stale keep-alive connection.
func NewWorkOSClient(cfg config.AppConfig) *workos.Client {
	timeout := time.Duration(cfg.WorkOSTimeoutSeconds) * time.Second

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.IdleConnTimeout = 30 * time.Second
	transport.ResponseHeaderTimeout = timeout
	transport.ForceAttemptHTTP2 = true

	opts := []workos.ClientOption{
		workos.WithClientID(cfg.WorkOSClientID),
		workos.WithMaxRetries(0),
		workos.WithHTTPClient(&http.Client{
			Timeout:   timeout,
			Transport: transport,
		}),
	}
	if cfg.WorkOSBaseURL != "" {
		opts = append(opts, workos.WithBaseURL(cfg.WorkOSBaseURL))
	}
	return workos.NewClient(cfg.WorkOSAPIKey, opts...)
}

// NewWorkOSUserManagement exposes the client's User Management service.
func NewWorkOSUserManagement(client *workos.Client) *workos.UserManagementService {
	return client.UserManagement()
}

// NewWebhookVerifier builds the verifier for inbound WorkOS webhooks.
func NewWebhookVerifier(cfg config.AppConfig) *workos.WebhookVerifier {
	return workos.NewWebhookVerifier(cfg.WorkOSWebhookSecret)
}

// NewSessionKeyring parses SESSION_COOKIE_KEY. A malformed value fails
// boot and names the variable.
func NewSessionKeyring(cfg config.AppConfig) (*session.Keyring, error) {
	ring, err := session.ParseKeyring(cfg.SessionCookieKey)
	if err != nil {
		return nil, fmt.Errorf("parsing SESSION_COOKIE_KEY: %w", err)
	}
	return ring, nil
}

// NewAuthMiddleware builds the token-validating middleware. The JWKS URL
// comes from config when set. An empty value derives the WorkOS JWKS URL
// for the configured client. Stops the JWKS refresh on OnStop.
func NewAuthMiddleware(cfg config.AppConfig, lc fx.Lifecycle) *middleware.AuthMiddleware {
	jwksURL := cfg.WorkOSJWKSURL
	if jwksURL == "" {
		jwksURL = "https://api.workos.com/sso/jwks/" + cfg.WorkOSClientID
	}

	am := middleware.NewAuthMiddleware(jwksURL)
	lc.Append(fx.StopHook(am.Close))
	return am
}
