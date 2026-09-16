package boot

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/logging"
)

// InitSentry enables Sentry in deployed stages with a DSN and is a no-op
// otherwise. Flushes pending events on shutdown.
func InitSentry(lc fx.Lifecycle, cfg config.AppConfig) error {
	if !cfg.Deployed() || cfg.SentryDSN == "" {
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.SentryDSN,
		Environment: cfg.Stage,
	})
	if err != nil {
		return fmt.Errorf("initializing sentry: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			sentry.Flush(2 * time.Second)
			return nil
		},
	})
	return nil
}

// notifySentry reports an error to Sentry tagged with the request and
// trace ids. No-op when Sentry is not initialized.
func notifySentry(ctx context.Context, err error) {
	hub := sentry.CurrentHub()
	if hub.Client() == nil {
		return
	}

	hub.WithScope(func(scope *sentry.Scope) {
		if requestID := logging.RequestIDFromContext(ctx); requestID != "" {
			scope.SetTag("request_id", requestID)
		}
		if span := trace.SpanContextFromContext(ctx); span.IsValid() {
			scope.SetTag("trace_id", span.TraceID().String())
		}
		hub.CaptureException(err)
	})
}
