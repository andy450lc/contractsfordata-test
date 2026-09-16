package middleware

import (
	"log/slog"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/logging"
)

// operationalPaths skip access logs, request metrics, and rate limiting.
var operationalPaths = map[string]bool{
	"/livez":   true,
	"/readyz":  true,
	"/metrics": true,
}

// SkipOperational reports whether the request hits an operational path.
func SkipOperational(c *echo.Context) bool {
	return operationalPaths[c.Request().URL.Path]
}

// RequestContext seeds the request context with the request id and a
// request-scoped logger. Honors an incoming X-Request-ID. Generates an
// id when the header is absent.
func RequestContext(logger *slog.Logger) echo.MiddlewareFunc {
	return echomw.RequestIDWithConfig(echomw.RequestIDConfig{
		RequestIDHandler: func(c *echo.Context, requestID string) {
			ctx := c.Request().Context()

			ctx = logging.ContextWithRequestID(ctx, requestID)
			ctx = logging.ContextWithLogger(ctx, logger)
			c.SetRequest(c.Request().WithContext(ctx))
		},
	})
}

// AccessLogger emits one structured line per request, leveled by outcome:
// 5xx → Error, 4xx → Warn, else Info. Operational endpoints are skipped.
func AccessLogger() echo.MiddlewareFunc {
	return echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
		Skipper:      SkipOperational,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogMethod:    true,
		LogURI:       true,
		LogRoutePath: true,
		LogStatus:    true,
		LogValuesFunc: func(c *echo.Context, v echomw.RequestLoggerValues) error {
			ctx := c.Request().Context()

			// The raw handler error arrives here before the central
			// handler writes the response. The error determines the status.
			status := v.Status
			if v.Error != nil {
				status = common.HTTPStatus(v.Error)
				if status == 0 {
					status = 500
				}
			}

			// Error content is logged by the central error handler. This
			// line is the request record only.
			args := []any{
				"method", v.Method,
				"uri", v.URI,
				"route", v.RoutePath,
				"status", status,
				"latency_ms", v.Latency.Milliseconds(),
				"remote_ip", v.RemoteIP,
			}

			switch {
			case status >= 500:
				logging.Error(ctx, "request", args...)
			case status >= 400:
				logging.Warn(ctx, "request", args...)
			default:
				logging.Info(ctx, "request", args...)
			}
			return nil
		},
	})
}
