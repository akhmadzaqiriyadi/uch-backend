package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gozaq_http_requests_total",
			Help: "Total number of HTTP requests processed by endpoint and status code.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gozaq_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	dbPoolTotalConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gozaq_db_pool_total_conns",
			Help: "Total number of active and idle connections in the PostgreSQL connection pool.",
		},
	)

	dbPoolAcquiredConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gozaq_db_pool_acquired_conns",
			Help: "Number of currently acquired/in-use connections in the PostgreSQL connection pool.",
		},
	)

	dbPoolIdleConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gozaq_db_pool_idle_conns",
			Help: "Number of idle connections currently in the PostgreSQL connection pool.",
		},
	)
)

// RecordDBPoolMetrics updates Prometheus gauges with current PostgreSQL pool stats
func RecordDBPoolMetrics(pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	stat := pool.Stat()
	dbPoolTotalConns.Set(float64(stat.TotalConns()))
	dbPoolAcquiredConns.Set(float64(stat.AcquiredConns()))
	dbPoolIdleConns.Set(float64(stat.IdleConns()))
}

// PrometheusMetrics records HTTP metrics for Prometheus scraping
func PrometheusMetrics() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip recording metrics for /metrics and /healthz
			if r.URL.Path == "/metrics" || r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				duration := time.Since(start).Seconds()
				statusStr := strconv.Itoa(ww.Status())

				httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
				httpRequestDuration.WithLabelValues(r.Method, r.URL.Path, statusStr).Observe(duration)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
