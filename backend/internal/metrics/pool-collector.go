package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// poolStatsCollector exposes pgxpool saturation as gauges and counters.
type poolStatsCollector struct {
	pool *pgxpool.Pool

	acquiredConns   *prometheus.Desc
	idleConns       *prometheus.Desc
	maxConns        *prometheus.Desc
	totalConns      *prometheus.Desc
	acquireCount    *prometheus.Desc
	acquireDuration *prometheus.Desc
	emptyAcquires   *prometheus.Desc
}

func newPoolStatsCollector(pool *pgxpool.Pool) *poolStatsCollector {
	return &poolStatsCollector{
		pool:            pool,
		acquiredConns:   prometheus.NewDesc("pgxpool_acquired_conns", "Connections currently in use", nil, nil),
		idleConns:       prometheus.NewDesc("pgxpool_idle_conns", "Idle connections in the pool", nil, nil),
		maxConns:        prometheus.NewDesc("pgxpool_max_conns", "Maximum pool size", nil, nil),
		totalConns:      prometheus.NewDesc("pgxpool_total_conns", "Total connections in the pool", nil, nil),
		acquireCount:    prometheus.NewDesc("pgxpool_acquire_count_total", "Cumulative connection acquires", nil, nil),
		acquireDuration: prometheus.NewDesc("pgxpool_acquire_duration_seconds_total", "Cumulative time blocked waiting for a connection", nil, nil),
		emptyAcquires:   prometheus.NewDesc("pgxpool_empty_acquire_count_total", "Acquires that waited because the pool was empty", nil, nil),
	}
}

func (c *poolStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquiredConns
	ch <- c.idleConns
	ch <- c.maxConns
	ch <- c.totalConns
	ch <- c.acquireCount
	ch <- c.acquireDuration
	ch <- c.emptyAcquires
}

func (c *poolStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.pool.Stat()

	ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(stats.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(stats.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(stats.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(stats.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.CounterValue, float64(stats.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.acquireDuration, prometheus.CounterValue, stats.AcquireDuration().Seconds())
	ch <- prometheus.MustNewConstMetric(c.emptyAcquires, prometheus.CounterValue, float64(stats.EmptyAcquireCount()))
}
