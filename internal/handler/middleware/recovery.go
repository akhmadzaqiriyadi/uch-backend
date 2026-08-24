package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"gozaq/pkg/response"
)

// Recoverer recovers from panics, logs stack trace with slog, and returns 500 JSON
func Recoverer(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					if rvr == http.ErrAbortHandler {
						panic(rvr)
					}

					stack := string(debug.Stack())
					reqID := chiMiddleware.GetReqID(r.Context())

					logger.Error("panic_recovered",
						slog.String("panic", fmt.Sprintf("%v", rvr)),
						slog.String("req_id", reqID),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
						slog.String("stack", stack),
					)

					response.InternalServerError(w, "An unexpected internal server error occurred", nil)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
