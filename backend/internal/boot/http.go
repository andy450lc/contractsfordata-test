package boot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/logging"
	"github.com/pixels-two/sow/backend/internal/metrics"
	appmw "github.com/pixels-two/sow/backend/internal/middleware"
)

// Server wraps the echo instance and the net/http server serving it.
type Server struct {
	Echo *echo.Echo

	cfg     config.AppConfig
	httpSrv *http.Server
	ln      net.Listener
}

// Addr returns the bound listen address. Empty before OnStart has run.
func (s *Server) Addr() string {
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

// NewServer assembles the echo server and its middleware stack.
// Middleware order, outermost first: request context, tracing, access
// log, metrics, hardening, rate limiting, panic recovery.
func NewServer(
	cfg config.AppConfig,
	logger *slog.Logger,
	m *metrics.Metrics,
	tp *sdktrace.TracerProvider,
	propagator propagation.TextMapPropagator,
) *Server {
	e := echo.New()
	e.Logger = logger

	e.Use(appmw.RequestContext(logger))
	e.Use(appmw.Tracing(tp, propagator))
	e.Use(appmw.AccessLogger())
	e.Use(m.Middleware(appmw.SkipOperational))

	// These run inside the metrics middleware. Rejections such as 429
	// and 413 land in the histogram's status label.
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "", // deprecated header, left unset
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'none'",
		ReferrerPolicy:        "no-referrer",
	}))
	// The session endpoints read a cookie. Credentialed requests are
	// allowed from the origins the allowlist names.
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowCredentials: true,
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			common.HeaderRequestID,
			common.HeaderIdempotencyKey,
		},
		ExposeHeaders: []string{echo.HeaderContentDisposition, common.HeaderRequestID},
	}))
	e.Use(middleware.BodyLimit(2 * 1024 * 1024))
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: appmw.SkipOperational,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:  cfg.RateLimitRPS,
			Burst: cfg.RateLimitBurst,
		}),
	}))

	e.Use(middleware.Recover())

	e.HTTPErrorHandler = newHTTPErrorHandler()

	e.GET("/metrics", echo.WrapHandler(m.Handler()))

	return &Server{
		Echo: e,
		cfg:  cfg,
		httpSrv: &http.Server{
			Handler:           e,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// RegisterServerLifecycle binds the listener in OnStart and drains
// in-flight requests within the grace period in OnStop. Binding
// synchronously fails boot on a taken port. Repeated shutdowns are safe.
func RegisterServerLifecycle(lc fx.Lifecycle, s *Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
			if err != nil {
				return fmt.Errorf("binding listener on port %d: %w", s.cfg.Port, err)
			}
			s.ln = ln

			go func() {
				if err := s.httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logging.Error(context.Background(), "http server terminated unexpectedly",
						common.LogKeyError, err.Error(),
					)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			graceCtx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.ShutdownGraceSeconds)*time.Second)
			defer cancel()

			if err := s.httpSrv.Shutdown(graceCtx); err != nil {
				return fmt.Errorf("shutting down http server: %w", err)
			}
			return nil
		},
	})
}

// errorEnvelope is the error response body for all non-2xx responses.
type errorEnvelope struct {
	Error   string `json:"error"`
	Details any    `json:"details,omitempty"`
}

// newHTTPErrorHandler is the single place errors become responses and
// get logged. Known sentinels and status-coded errors map to their
// status with a one-line log. Anything else returns a generic 500 and
// logs the error with its stack.
func newHTTPErrorHandler() echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		resp, _ := echo.UnwrapResponse(c.Response())
		if resp != nil && resp.Committed {
			return
		}

		ctx := c.Request().Context()

		status, envelope, expected := mapError(err)
		if expected {
			logging.Warn(ctx, "request failed",
				common.LogKeyError, err.Error(),
				"status", status,
			)
		} else {
			logging.Error(ctx, "request failed unexpectedly",
				common.LogKeyError, fmt.Sprintf("%+v", err),
				"status", status,
			)
			notifySentry(ctx, err)
		}

		if writeErr := c.JSON(status, envelope); writeErr != nil {
			logging.Error(ctx, "writing error response",
				common.LogKeyError, writeErr.Error(),
			)
		}
	}
}

// mapError classifies an error into status, response body, and whether
// the error was expected.
func mapError(err error) (int, errorEnvelope, bool) {
	status := common.HTTPStatus(err)
	if status == 0 {
		return http.StatusInternalServerError, errorEnvelope{Error: "internal_server_error"}, false
	}

	envelope := errorEnvelope{Error: statusErrorCode(status)}
	if code := common.ErrorCode(err); code != "" {
		envelope.Error = code
	}

	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) && httpErr.Message != "" {
		envelope.Error = httpErr.Message
	}
	var validationErr *common.ValidationError
	if errors.As(err, &validationErr) {
		envelope.Error = "validation_failed"
		envelope.Details = validationErr.Details
	}
	var weakPasswordErr *common.WeakPasswordError
	if errors.As(err, &weakPasswordErr) {
		envelope.Details = map[string]string{"message": weakPasswordErr.Message}
	}
	return status, envelope, true
}

// statusErrorCode renders an HTTP status as a snake_case machine code
// such as "not_found" or "too_many_requests".
func statusErrorCode(status int) string {
	text := http.StatusText(status)
	if text == "" {
		return "error"
	}
	return strings.ToLower(strings.ReplaceAll(text, " ", "_"))
}
