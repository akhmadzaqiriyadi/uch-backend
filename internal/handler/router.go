package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"gozaq/config"
	"gozaq/internal/handler/middleware"
)

func NewRouter(
	cfg *config.Config,
	logger *slog.Logger,
	userHandler *UserHandler,
	docsHandler *DocsHandler,
	healthHandler *HealthHandler,
	uploadHandler *UploadHandler,
	rbacHandler *RBACHandler,
) http.Handler {
	r := chi.NewRouter()

	// Rate limiters
	globalLimiter := middleware.NewRateLimiter(rate.Limit(50), 100) // 50 RPS, burst 100
	authLimiter := middleware.NewRateLimiter(rate.Limit(5), 10)     // 5 RPS, burst 10 (anti brute-force)

	// 1. Security Headers (Go's Helmet equivalent)
	r.Use(middleware.SecurityHeaders())

	// 2. Request Correlation & Context
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Timeout(30 * time.Second))

	// 3. Performance: Gzip Compression
	r.Use(chiMiddleware.Compress(5))

	// 4. Observability: Prometheus Metrics & Structured JSON Logger
	r.Use(middleware.PrometheusMetrics())
	r.Use(middleware.StructuredLogger(logger))

	// 5. Reliability: Structured Panic Recovery (returns 500 JSON)
	r.Use(middleware.Recoverer(logger))

	// 6. Security: Global Rate Limiting
	r.Use(globalLimiter.Limit())

	// 7. CORS Configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Request-ID", "X-Response-Time"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Static File Server for uploaded files
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// Go Runtime Profiler (/debug/pprof/* for CPU, heap, goroutine analysis)
	r.Mount("/debug", chiMiddleware.Profiler())

	// Comprehensive Health Check endpoint (Postgres, Redis, & System metrics)
	r.Get("/healthz", healthHandler.Check)

	// Prometheus Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Scalar API Reference Documentation
	r.Get("/openapi.json", docsHandler.ServeOpenAPISpec)
	r.Get("/docs", docsHandler.ServeScalar)
	r.Get("/docs/*", docsHandler.ServeScalar)

	// API v1 Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public Auth routes (Anti Brute-Force Rate Limiting)
		r.Group(func(r chi.Router) {
			r.Use(authLimiter.Limit())

			r.Post("/auth/register", userHandler.Register)
			r.Post("/auth/login", userHandler.Login)
			r.Post("/auth/refresh", userHandler.RefreshToken)
		})

		// Protected Routes (Requires valid JWT Access Token)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg))

			r.Post("/auth/logout", userHandler.Logout)
			r.Get("/auth/profile", userHandler.GetProfile)
			r.Put("/auth/profile", userHandler.UpdateProfile)

			// File Upload Endpoint (Protected by PBAC: uploads:create)
			r.With(middleware.RequirePermission("uploads:create")).
				Post("/uploads", uploadHandler.UploadFile)

			// User Management (Protected by Granular Permissions: users:read & users:delete)
			r.With(middleware.RequirePermission("users:read")).
				Get("/users", userHandler.ListUsers)

			r.With(middleware.RequirePermission("users:delete")).
				Delete("/users/{id}", userHandler.DeleteUser)

			// Dynamic RBAC & PBAC Management (Admin Only / roles:manage)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAnyPermission("roles:read", "roles:manage"))

				r.Get("/roles", rbacHandler.ListRoles)
				r.Get("/permissions", rbacHandler.ListPermissions)
			})

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission("roles:manage"))

				r.Post("/roles/{role_id}/permissions/{permission_id}", rbacHandler.AssignPermission)
				r.Delete("/roles/{role_id}/permissions/{permission_id}", rbacHandler.RevokePermission)
			})
		})
	})

	return r
}
