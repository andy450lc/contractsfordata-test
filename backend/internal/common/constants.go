// Package common holds context keys, constants, and sentinel
// errors shared across layers.
package common

// ContextKey is the type for request-context keys owned by this app.
type ContextKey string

const (
	// SessionCookieName is the cookie holding the sealed refresh session.
	SessionCookieName = "sow_session"
)

const (
	// EchoContextKeyValidatedDTO is where ValidateRequest stores the typed,
	// validated request struct on the echo context.
	EchoContextKeyValidatedDTO = "validated_dto"
)

const (
	// ContextKeyUserID is where the auth middleware stores the
	// authenticated user's id on the request context.
	ContextKeyUserID ContextKey = "user_id"
)

const (
	HeaderRequestID      = "X-Request-ID"
	HeaderIdempotencyKey = "Idempotency-Key"
)

const (
	LogKeyRequestID = "request_id"
	LogKeyTraceID   = "trace_id"
	LogKeySpanID    = "span_id"
	LogKeyUserID    = "user_id"
	LogKeyError     = "error"
)
