package middleware

import "net/http"

// SecurityHeaders (Helmet equivalent for Go) sets essential HTTP security headers
func SecurityHeaders() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent Clickjacking (disallow embedding in iframes)
			w.Header().Set("X-Frame-Options", "DENY")

			// Control referrer information sent in requests
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Cross-site scripting (XSS) filter
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// HTTP Strict Transport Security (HSTS) - enforce HTTPS in production
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

			// Restrict browser features and APIs
			w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")

			next.ServeHTTP(w, r)
		})
	}
}
