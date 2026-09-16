// Package logging is the only place the app logs from. All helpers take
// ctx first. Correlation ids ride in via the context.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"

	"github.com/pixels-two/sow/backend/internal/common"
)

type ctxKey int

const (
	loggerKey ctxKey = iota
	attrsKey
	requestIDKey
)

// New builds the root JSON logger writing to stdout. Each line carries
// the file and line of the call site.
func New(stage string) *slog.Logger {
	return NewWithWriter(stage, os.Stdout)
}

// NewWithWriter is New with an explicit sink.
func NewWithWriter(stage string, w io.Writer) *slog.Logger {
	level := slog.LevelInfo
	if stage == "dev" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	return slog.New(handler)
}

// ContextWithLogger stashes a (typically request-scoped) logger in ctx.
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// ContextWithRequestID stores the raw request id and attaches it
// as a log attribute.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	return ContextWithAttrs(ctx, slog.String(common.LogKeyRequestID, requestID))
}

// RequestIDFromContext returns the request id stored in ctx, or "".
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// ContextWithUserID attaches the authenticated user id. Every subsequent
// log line in the request carries it.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return ContextWithAttrs(ctx, slog.String(common.LogKeyUserID, userID))
}

// ContextWithAttrs attaches attributes to ctx. Every subsequent log
// line in the request carries them.
func ContextWithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing, _ := ctx.Value(attrsKey).([]slog.Attr)

	combined := make([]slog.Attr, 0, len(existing)+len(attrs))
	combined = append(combined, existing...)
	combined = append(combined, attrs...)
	return context.WithValue(ctx, attrsKey, combined)
}

// FromContext recovers the request-scoped logger (or the default) enriched
// with OTel trace/span ids and any attached domain attrs.
func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)
	if !ok {
		logger = slog.Default()
	}

	if span := trace.SpanContextFromContext(ctx); span.IsValid() {
		logger = logger.With(
			slog.String(common.LogKeyTraceID, span.TraceID().String()),
			slog.String(common.LogKeySpanID, span.SpanID().String()),
		)
	}

	if attrs, ok := ctx.Value(attrsKey).([]slog.Attr); ok {
		for _, attr := range attrs {
			logger = logger.With(attr)
		}
	}
	return logger
}

// Structured key-value logging helpers.

func Debug(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).DebugContext(ctx, msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).InfoContext(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).WarnContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).ErrorContext(ctx, msg, args...)
}
