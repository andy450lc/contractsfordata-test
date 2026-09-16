package common

import (
	"errors"
	"net/http"
)

// Domain sentinel errors, grouped by domain. Compare with errors.Is.
// Sentinels are expected errors. They carry no stacks.

// Authentication errors.
var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbiddenOrigin = errors.New("forbidden origin")
)

// User errors.
var (
	ErrUserNotFound = errors.New("user not found")
)

// Request and provider errors.
var (
	ErrInvalidState             = errors.New("invalid state")
	ErrProviderUnavailable      = errors.New("provider unavailable")
	ErrEmailDeliveryUnavailable = errors.New("email delivery unavailable")
	ErrGenerationFailed         = errors.New("document generation failed")
	ErrIdempotencyConflict      = errors.New("idempotency conflict")
)

// Credential and token errors.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidCode        = errors.New("invalid code")
	ErrInvalidResetToken  = errors.New("invalid reset token")
)

// WeakPasswordError carries the identity provider's rejection message
// for a password that fails its strength check. Maps to a 422 with a
// weak_password envelope.
type WeakPasswordError struct {
	Message string
}

func (e *WeakPasswordError) Error() string { return "weak password" }

func (e *WeakPasswordError) StatusCode() int { return http.StatusUnprocessableEntity }

// ValidationError carries structured field errors from request
// validation. Maps to a 400 with a validation_failed envelope.
type ValidationError struct {
	Details []ValidationDetail
}

type ValidationDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string { return "validation failed" }

func (e *ValidationError) StatusCode() int { return http.StatusBadRequest }

// httpStatusBySentinel maps each request/provider sentinel to its HTTP
// status, in precedence order.
var httpStatusBySentinel = []struct {
	err    error
	status int
}{
	{ErrUserNotFound, http.StatusNotFound},
	{ErrUnauthorized, http.StatusUnauthorized},
	{ErrInvalidState, http.StatusBadRequest},
	{ErrForbiddenOrigin, http.StatusForbidden},
	{ErrProviderUnavailable, http.StatusServiceUnavailable},
	{ErrEmailDeliveryUnavailable, http.StatusServiceUnavailable},
	{ErrGenerationFailed, http.StatusInternalServerError},
	{ErrIdempotencyConflict, http.StatusConflict},
	{ErrInvalidCredentials, http.StatusUnauthorized},
	{ErrInvalidCode, http.StatusBadRequest},
	{ErrInvalidResetToken, http.StatusBadRequest},
}

// HTTPStatus maps an error to its HTTP status. Returns 0 for
// errors with no mapping.
func HTTPStatus(err error) int {
	var statusCoder interface{ StatusCode() int }
	if errors.As(err, &statusCoder) {
		return statusCoder.StatusCode()
	}
	for _, entry := range httpStatusBySentinel {
		if errors.Is(err, entry.err) {
			return entry.status
		}
	}
	return 0
}

// errorCodeBySentinel maps each request/provider sentinel to its
// machine-readable code, in precedence order.
var errorCodeBySentinel = []struct {
	err  error
	code string
}{
	{ErrUserNotFound, "user_not_found"},
	{ErrInvalidState, "invalid_state"},
	{ErrForbiddenOrigin, "forbidden_origin"},
	{ErrProviderUnavailable, "provider_unavailable"},
	{ErrEmailDeliveryUnavailable, "email_delivery_unavailable"},
	{ErrGenerationFailed, "generation_failed"},
	{ErrIdempotencyConflict, "idempotency_conflict"},
	{ErrInvalidCredentials, "invalid_credentials"},
	{ErrInvalidCode, "invalid_code"},
	{ErrInvalidResetToken, "invalid_reset_token"},
}

// ErrorCode maps an error to its machine-readable code. Returns ""
// for errors without a specific code.
func ErrorCode(err error) string {
	var weakPasswordErr *WeakPasswordError
	if errors.As(err, &weakPasswordErr) {
		return "weak_password"
	}
	for _, entry := range errorCodeBySentinel {
		if errors.Is(err, entry.err) {
			return entry.code
		}
	}
	return ""
}
