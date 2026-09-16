package middleware

import (
	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Tracing starts a server span per request and honors an incoming W3C
// traceparent. Operational endpoints are skipped.
func Tracing(tp trace.TracerProvider, propagator propagation.TextMapPropagator) echo.MiddlewareFunc {
	tracer := tp.Tracer("backend/http")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if SkipOperational(c) {
				return next(c)
			}

			req := c.Request()
			ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			spanName := req.Method
			if c.Path() != "" {
				spanName = req.Method + " " + c.Path()
			}
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.request.method", req.Method),
					attribute.String("url.path", req.URL.Path),
				),
			)
			defer span.End()

			c.SetRequest(req.WithContext(ctx))

			err := next(c)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "request failed")
			}
			return err
		}
	}
}
