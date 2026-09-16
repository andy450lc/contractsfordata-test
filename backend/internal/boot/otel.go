package boot

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/config"
)

// NewTracerProvider wires OTel tracing. Without an OTLP endpoint, spans
// are still created but not exported. Shuts down with the fx lifecycle.
func NewTracerProvider(lc fx.Lifecycle, cfg config.AppConfig) (*sdktrace.TracerProvider, error) {
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("sow-backend"),
		semconv.DeploymentEnvironmentNameKey.String(cfg.Stage),
	))
	if err != nil {
		return nil, fmt.Errorf("building otel resource: %w", err)
	}

	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if cfg.OTELExporterOTLPEndpoint != "" {
		exporter, err := otlptracehttp.New(context.Background(),
			otlptracehttp.WithEndpointURL(cfg.OTELExporterOTLPEndpoint),
		)
		if err != nil {
			return nil, fmt.Errorf("creating otlp exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(opts...)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if err := tp.Shutdown(ctx); err != nil {
				return fmt.Errorf("shutting down tracer provider: %w", err)
			}
			return nil
		},
	})
	return tp, nil
}

// InstallOTelGlobals publishes the tracer provider and the propagator
// as the process defaults. Code without an injected provider, such as
// a queued job, traces through the globals.
func InstallOTelGlobals(tp *sdktrace.TracerProvider, propagator propagation.TextMapPropagator) {
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagator)
}

// NewPropagator builds the W3C traceparent and baggage propagator.
func NewPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
