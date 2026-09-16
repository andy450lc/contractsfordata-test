// Package config holds env-tagged configuration structs.
// No logic lives here beyond validation the struct tags cannot express.
package config

import "errors"

type AppConfig struct {
	Stage string `env:"STAGE" envDefault:"dev" validate:"oneof=dev staging prod"`
	Port  int    `env:"PORT" envDefault:"8080" validate:"gte=0,lte=65535"`

	DatabaseURL    string `env:"DATABASE_URL,required,notEmpty" validate:"required"`
	DBPoolMaxConns int32  `env:"DB_POOL_MAX_CONNS" envDefault:"10" validate:"gte=1"`
	DBPoolMinConns int32  `env:"DB_POOL_MIN_CONNS" envDefault:"2" validate:"gte=0"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:5173,http://127.0.0.1:5173,http://localhost:4321,http://127.0.0.1:4321"`

	RateLimitRPS   float64 `env:"RATE_LIMIT_RPS" envDefault:"20" validate:"gt=0"`
	RateLimitBurst int     `env:"RATE_LIMIT_BURST" envDefault:"40" validate:"gte=1"`

	AuthRateLimitRPS   float64 `env:"AUTH_RATE_LIMIT_RPS" envDefault:"2" validate:"gt=0"`
	AuthRateLimitBurst int     `env:"AUTH_RATE_LIMIT_BURST" envDefault:"10" validate:"gte=1"`

	TemplateDeliveryRateLimitRPS   float64 `env:"TEMPLATE_DELIVERY_RATE_LIMIT_RPS" envDefault:"1" validate:"gt=0"`
	TemplateDeliveryRateLimitBurst int     `env:"TEMPLATE_DELIVERY_RATE_LIMIT_BURST" envDefault:"5" validate:"gte=1"`

	ShutdownGraceSeconds int `env:"SHUTDOWN_GRACE_SECONDS" envDefault:"10" validate:"gte=1"`

	WorkOSAPIKey         string `env:"WORKOS_API_KEY,required,notEmpty" validate:"required"`
	WorkOSClientID       string `env:"WORKOS_CLIENT_ID,required,notEmpty" validate:"required"`
	WorkOSWebhookSecret  string `env:"WORKOS_WEBHOOK_SECRET,required,notEmpty" validate:"required"`
	WorkOSRedirectURI    string `env:"WORKOS_REDIRECT_URI,required,notEmpty" validate:"required,url"`
	WorkOSJWKSURL        string `env:"WORKOS_JWKS_URL"`
	WorkOSBaseURL        string `env:"WORKOS_BASE_URL"`
	WorkOSTimeoutSeconds int    `env:"WORKOS_TIMEOUT_SECONDS" envDefault:"15" validate:"gte=1"`

	WebAppURL            string `env:"WEB_APP_URL,required,notEmpty" validate:"required,url"`
	SessionCookieKey     string `env:"SESSION_COOKIE_KEY,required,notEmpty" validate:"required"`
	SessionMaxAgeSeconds int    `env:"SESSION_MAX_AGE_SECONDS" envDefault:"604800" validate:"gte=1"`

	SentryDSN string `env:"SENTRY_DSN" validate:"required_if=Stage prod"`

	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`

	ResendAPIKey string `env:"RESEND_API_KEY"`
	EmailFrom    string `env:"EMAIL_FROM" validate:"omitempty,email"`
}

// Deployed reports whether the app runs in staging or prod.
func (c AppConfig) Deployed() bool {
	return c.Stage == "staging" || c.Stage == "prod"
}

// Validate checks rules struct tags cannot express. Call it after
// struct-tag validation passes.
func (c AppConfig) Validate() error {
	if c.Deployed() && (c.ResendAPIKey == "" || c.EmailFrom == "") {
		return errors.New("RESEND_API_KEY and EMAIL_FROM are required when deployed")
	}
	return nil
}
