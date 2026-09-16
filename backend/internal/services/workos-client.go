package services

import (
	"context"
	"fmt"

	"github.com/workos/workos-go/v10"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/config"
)

// WorkOSClient is the slice of the identity provider the services use.
type WorkOSClient interface {
	AuthorizationURL(state string, provider string) string
	AuthenticateWithCode(ctx context.Context, code string, ip string, userAgent string) (*workos.AuthenticateResponse, error)
	AuthenticateWithRefreshToken(ctx context.Context, refreshToken string, ip string, userAgent string) (*workos.AuthenticateResponse, error)
	RevokeSession(ctx context.Context, sessionID string) error
	GetUser(ctx context.Context, id string) (*workos.User, error)

	AuthenticateWithPassword(ctx context.Context, email string, password string, ip string, userAgent string) (*workos.AuthenticateResponse, error)
	CreateUser(ctx context.Context, email string, firstName string, lastName string, password string) (*workos.UserCreateResponse, error)
	AuthenticateWithEmailVerification(ctx context.Context, pendingToken string, code string, ip string, userAgent string) (*workos.AuthenticateResponse, error)
	SendVerificationEmail(ctx context.Context, userID string) error
	FindUserByEmail(ctx context.Context, email string) (*workos.User, error)
	CreatePasswordReset(ctx context.Context, email string) (*workos.PasswordReset, error)
	ConfirmPasswordReset(ctx context.Context, token string, newPassword string) (*workos.ResetPasswordResponse, error)
}

type sdkWorkOSClient struct {
	users       *workos.UserManagementService
	redirectURI string
}

// NewSDKWorkOSClient wraps the WorkOS SDK behind the WorkOSClient
// interface.
func NewSDKWorkOSClient(users *workos.UserManagementService, cfg config.AppConfig) *sdkWorkOSClient {
	return &sdkWorkOSClient{users: users, redirectURI: cfg.WorkOSRedirectURI}
}

// AuthorizationURL builds the social sign-in URL for a provider name.
// google maps to GoogleOAuth, the only provider the login endpoint
// accepts. The browser lands on the provider's own consent screen,
// with no WorkOS screen hint.
func (c *sdkWorkOSClient) AuthorizationURL(state string, provider string) string {
	authProvider := workos.UserManagementAuthenticationProviderGoogleOAuth

	return c.users.GetAuthorizationURL(&workos.UserManagementGetAuthorizationURLParams{
		Provider:    &authProvider,
		RedirectURI: c.redirectURI,
		State:       &state,
	})
}

// AuthenticateWithCode exchanges an authorization code for tokens.
func (c *sdkWorkOSClient) AuthenticateWithCode(ctx context.Context, code string, ip string, userAgent string) (*workos.AuthenticateResponse, error) {
	resp, err := c.users.AuthenticateWithCode(ctx, &workos.UserManagementAuthenticateWithCodeParams{
		Code:      code,
		IPAddress: optional(ip),
		UserAgent: optional(userAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}
	return resp, nil
}

// AuthenticateWithRefreshToken rotates a refresh token for new tokens.
func (c *sdkWorkOSClient) AuthenticateWithRefreshToken(ctx context.Context, refreshToken string, ip string, userAgent string) (*workos.AuthenticateResponse, error) {
	resp, err := c.users.AuthenticateWithRefreshToken(ctx, &workos.UserManagementAuthenticateWithRefreshTokenParams{
		RefreshToken: refreshToken,
		IPAddress:    optional(ip),
		UserAgent:    optional(userAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("refreshing provider session: %w", err)
	}
	return resp, nil
}

// RevokeSession ends a provider session.
func (c *sdkWorkOSClient) RevokeSession(ctx context.Context, sessionID string) error {
	err := c.users.RevokeSession(ctx, &workos.UserManagementRevokeSessionParams{SessionID: sessionID})
	if err != nil {
		return fmt.Errorf("revoking provider session: %w", err)
	}
	return nil
}

// GetUser fetches a user from the provider.
func (c *sdkWorkOSClient) GetUser(ctx context.Context, id string) (*workos.User, error) {
	user, err := c.users.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching provider user: %w", err)
	}
	return user, nil
}

// AuthenticateWithPassword exchanges an email and password for tokens.
// An unverified user comes back as an error carrying a pending
// authentication token and email verification id.
func (c *sdkWorkOSClient) AuthenticateWithPassword(ctx context.Context, email string, password string, ip string, userAgent string) (*workos.AuthenticateResponse, error) {
	resp, err := c.users.AuthenticateWithPassword(ctx, &workos.UserManagementAuthenticateWithPasswordParams{
		Email:     email,
		Password:  password,
		IPAddress: optional(ip),
		UserAgent: optional(userAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("authenticating with password: %w", err)
	}
	return resp, nil
}

// CreateUser creates an unverified user at the provider.
func (c *sdkWorkOSClient) CreateUser(ctx context.Context, email string, firstName string, lastName string, password string) (*workos.UserCreateResponse, error) {
	resp, err := c.users.Create(ctx, &workos.UserManagementCreateParams{
		Email:     email,
		FirstName: optional(firstName),
		LastName:  optional(lastName),
		Password:  workos.UserManagementPasswordPlaintext{Password: password},
	})
	if err != nil {
		return nil, fmt.Errorf("creating provider user: %w", err)
	}
	return resp, nil
}

// AuthenticateWithEmailVerification completes a pending sign-in with
// the code the provider emailed.
func (c *sdkWorkOSClient) AuthenticateWithEmailVerification(ctx context.Context, pendingToken string, code string, ip string, userAgent string) (*workos.AuthenticateResponse, error) {
	resp, err := c.users.AuthenticateWithEmailVerification(ctx, &workos.UserManagementAuthenticateWithEmailVerificationParams{
		Code:                       code,
		PendingAuthenticationToken: pendingToken,
		IPAddress:                  optional(ip),
		UserAgent:                  optional(userAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("authenticating with email verification: %w", err)
	}
	return resp, nil
}

// SendVerificationEmail asks the provider to email a fresh code.
func (c *sdkWorkOSClient) SendVerificationEmail(ctx context.Context, userID string) error {
	_, err := c.users.SendVerificationEmail(ctx, userID)
	if err != nil {
		return fmt.Errorf("sending verification email: %w", err)
	}
	return nil
}

// FindUserByEmail looks up a user by address. Returns ErrUserNotFound
// when the provider knows no user with that address.
func (c *sdkWorkOSClient) FindUserByEmail(ctx context.Context, email string) (*workos.User, error) {
	it := c.users.List(ctx, &workos.UserManagementListParams{Email: optional(email)})
	if !it.Next() {
		if err := it.Err(); err != nil {
			return nil, fmt.Errorf("listing provider users: %w", err)
		}
		return nil, common.ErrUserNotFound
	}
	return it.Current(), nil
}

// CreatePasswordReset issues a single-use reset token for an address.
func (c *sdkWorkOSClient) CreatePasswordReset(ctx context.Context, email string) (*workos.PasswordReset, error) {
	resp, err := c.users.ResetPassword(ctx, &workos.UserManagementResetPasswordParams{Email: email})
	if err != nil {
		return nil, fmt.Errorf("creating password reset: %w", err)
	}
	return resp, nil
}

// ConfirmPasswordReset consumes a reset token and sets the new
// password. The provider verifies the email as a side effect.
func (c *sdkWorkOSClient) ConfirmPasswordReset(ctx context.Context, token string, newPassword string) (*workos.ResetPasswordResponse, error) {
	resp, err := c.users.ConfirmPasswordReset(ctx, &workos.UserManagementConfirmPasswordResetParams{
		Token:       token,
		NewPassword: newPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("confirming password reset: %w", err)
	}
	return resp, nil
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
