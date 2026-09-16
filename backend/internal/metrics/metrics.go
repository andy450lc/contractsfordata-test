// Package metrics owns the Prometheus registry: the request duration
// histogram, pgxpool saturation gauges, and the Go runtime collectors.
package metrics

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/pixels-two/sow/backend/internal/common"
)

type Metrics struct {
	registry        *prometheus.Registry
	RequestDuration *prometheus.HistogramVec
}

// New builds a self-contained registry with runtime, pool, and request
// collectors. Multiple instances can coexist in one process.
func New(pool *pgxpool.Pool) (*Metrics, error) {
	registry := prometheus.NewRegistry()

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	for _, c := range []prometheus.Collector{
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		newPoolStatsCollector(pool),
		requestDuration,
	} {
		if err := registry.Register(c); err != nil {
			return nil, err //nolint:wrapcheck // registration fails only on programmer error at boot
		}
	}

	return &Metrics{
		registry:        registry,
		RequestDuration: requestDuration,
	}, nil
}

// Handler serves the scrape endpoint.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware records the request duration histogram. The path label is
// always the route template. The status label reflects the returned error.
func (m *Metrics) Middleware(skipper func(c *echo.Context) bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if skipper(c) {
				return next(c)
			}

			timer := prometheus.NewTimer(nil)
			err := next(c)

			path := c.Path()
			if path == "" {
				// Unmatched routes share one label to keep cardinality bounded.
				path = "unmatched"
			}

			_, status := echo.ResolveResponseStatus(c.Response(), nil)
			if err != nil {
				status = common.HTTPStatus(err)
				if status == 0 {
					status = http.StatusInternalServerError
				}
			}

			m.RequestDuration.
				WithLabelValues(c.Request().Method, path, strconv.Itoa(status)).
				Observe(timer.ObserveDuration().Seconds())
			return err
		}
	}
}
