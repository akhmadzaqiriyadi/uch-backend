package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// StructuredLogger logs HTTP requests with structured JSON attributes and proper log levels
func StructuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				duration := time.Since(start)
				status := ww.Status()
				reqID := middleware.GetReqID(r.Context())

				attrs := []any{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
					slog.String("req_id", reqID),
					slog.String("user_agent", r.UserAgent()),
					slog.String("remote_addr", r.RemoteAddr),
				}

				switch {
				case status >= 500:
					logger.Error("HTTP Server Error", attrs...)
				case status >= 400:
					logger.Warn("HTTP Client Error", attrs...)
				default:
					logger.Info("HTTP Request Handled", attrs...)
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
