package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed by endpoint and status code.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

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
