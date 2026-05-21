package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type DBPoolStats interface {
	AcquiredConns() int32
	IdleConns() int32
	TotalConns() int32
	MaxConns() int32
	AcquireCount() int64
	CanceledAcquireCount() int64
	EmptyAcquireCount() int64
}

type routerMetrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

func newRouterMetrics(dbStats func() DBPoolStats) *routerMetrics {
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "learning_garden",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "route", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "learning_garden",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request latency in seconds.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"method", "route", "status"})
	inFlight := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "learning_garden",
		Subsystem: "http",
		Name:      "requests_in_flight",
		Help:      "Current number of in-flight HTTP requests.",
	})

	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(requests, duration, inFlight)
	registerDBPoolMetrics(registry, dbStats)

	return &routerMetrics{
		registry: registry,
		requests: requests,
		duration: duration,
		inFlight: inFlight,
	}
}

func (m *routerMetrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *routerMetrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		next.ServeHTTP(recorder, r)

		status := strconv.Itoa(recorder.statusCode())
		route := routePattern(r)
		m.requests.WithLabelValues(r.Method, route, status).Inc()
		m.duration.WithLabelValues(r.Method, route, status).Observe(time.Since(start).Seconds())
	})
}

func routePattern(r *http.Request) string {
	if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
		if pattern := routeContext.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unknown"
}

func registerDBPoolMetrics(registry *prometheus.Registry, dbStats func() DBPoolStats) {
	if dbStats == nil {
		return
	}
	registerDBGauge(registry, dbStats, "acquired_conns", "Number of currently acquired database connections.", func(stats DBPoolStats) float64 {
		return float64(stats.AcquiredConns())
	})
	registerDBGauge(registry, dbStats, "idle_conns", "Number of idle database connections.", func(stats DBPoolStats) float64 {
		return float64(stats.IdleConns())
	})
	registerDBGauge(registry, dbStats, "total_conns", "Total number of database connections.", func(stats DBPoolStats) float64 {
		return float64(stats.TotalConns())
	})
	registerDBGauge(registry, dbStats, "max_conns", "Configured maximum number of database connections.", func(stats DBPoolStats) float64 {
		return float64(stats.MaxConns())
	})
	registerDBGauge(registry, dbStats, "acquire_count", "Cumulative database connection acquire count.", func(stats DBPoolStats) float64 {
		return float64(stats.AcquireCount())
	})
	registerDBGauge(registry, dbStats, "canceled_acquire_count", "Cumulative canceled database connection acquire count.", func(stats DBPoolStats) float64 {
		return float64(stats.CanceledAcquireCount())
	})
	registerDBGauge(registry, dbStats, "empty_acquire_count", "Cumulative acquire count when the database pool was empty.", func(stats DBPoolStats) float64 {
		return float64(stats.EmptyAcquireCount())
	})
}

func registerDBGauge(registry *prometheus.Registry, dbStats func() DBPoolStats, name string, help string, value func(DBPoolStats) float64) {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "learning_garden",
		Subsystem: "db_pool",
		Name:      name,
		Help:      help,
	}, func() float64 {
		stats := dbStats()
		if stats == nil {
			return 0
		}
		return value(stats)
	}))
}
