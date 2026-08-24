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
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Static File Server for uploaded files
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

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

		// Protected User Routes (Requires valid JWT Access Token)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg))

			r.Post("/auth/logout", userHandler.Logout)
			r.Get("/auth/profile", userHandler.GetProfile)
			r.Put("/auth/profile", userHandler.UpdateProfile)

			// File Upload Endpoint
			r.Post("/uploads", uploadHandler.UploadFile)

			// Admin-Only Routes (RBAC Protected)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))

				r.Get("/users", userHandler.ListUsers)
				r.Delete("/users/{id}", userHandler.DeleteUser)
			})
		})
	})

	return r
}
