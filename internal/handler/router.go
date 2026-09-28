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
	auditHandler *AuditHandler,
	roomHandler *RoomHandler,
	bookingHandler *BookingHandler,
	wsHandler *WebSocketHandler,
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

	// 7. Dynamic CORS Configuration
	allowCredentials := len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "*"
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Request-ID", "X-Response-Time"},
		AllowCredentials: allowCredentials,
		MaxAge:           300,
	}))

	// Storage Proxy & Static File Server for uploaded files
	r.Get("/uploads/*", uploadHandler.ServeFile)
	r.Head("/uploads/*", uploadHandler.ServeFile)
	r.Get("/api/v1/upload/file/*", uploadHandler.ServeFile)
	r.Head("/api/v1/upload/file/*", uploadHandler.ServeFile)

	// Realtime WebSocket endpoint for instant booking notifications
	r.Get("/ws", wsHandler.HandleWS)

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
		r.Get("/ws", wsHandler.HandleWS)

		// Public Auth routes (Anti Brute-Force Rate Limiting)
		r.Group(func(r chi.Router) {
			r.Use(authLimiter.Limit())

			r.Post("/auth/register", userHandler.Register)
			r.Post("/auth/login", userHandler.Login)
			r.Post("/auth/refresh", userHandler.RefreshToken)
			r.Post("/auth/verify-email", userHandler.VerifyEmail)
			r.Post("/auth/resend-verification", userHandler.ResendVerification)
			r.Post("/auth/forgot-password", userHandler.ForgotPassword)
			r.Post("/auth/reset-password", userHandler.ResetPassword)
		})

		// Public Rooms catalog
		r.Get("/rooms", roomHandler.ListRooms)
		r.Get("/rooms/{id}", roomHandler.GetRoom)

		// Protected Routes (Requires valid JWT Access Token)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg))

			r.Post("/auth/logout", userHandler.Logout)
			r.Get("/auth/profile", userHandler.GetProfile)
			r.Put("/auth/profile", userHandler.UpdateProfile)
			r.Put("/auth/password", userHandler.ChangePassword)

			// User Booking Flow (Wajib Login & Terproteksi)
			r.Post("/bookings", bookingHandler.CreateBooking)
			r.Get("/my-bookings", bookingHandler.ListMyBookings)
			r.Post("/bookings/{id}/cancel", bookingHandler.CancelBooking)
			r.Post("/bookings/self-checkin", bookingHandler.SelfCheckIn)

			// Admin & Manager Bookings Management
			r.With(middleware.RequireAnyPermission("bookings:read", "bookings:manage")).
				Get("/bookings", bookingHandler.ListAllBookings)
			r.With(middleware.RequireAnyPermission("bookings:read", "bookings:manage")).
				Get("/bookings/{id}", bookingHandler.GetBooking)
			r.With(middleware.RequirePermission("bookings:manage")).
				Patch("/bookings/{id}/status", bookingHandler.UpdateBookingStatus)
			r.With(middleware.RequirePermission("bookings:manage")).
				Post("/bookings/checkin", bookingHandler.AdminCheckIn)
			r.With(middleware.RequirePermission("bookings:delete")).
				Delete("/bookings/{id}", bookingHandler.DeleteBooking)

			// Admin Rooms Management (CRUD Ruangan Hub)
			r.With(middleware.RequirePermission("rooms:create")).
				Post("/rooms", roomHandler.CreateRoom)
			r.With(middleware.RequirePermission("rooms:update")).
				Put("/rooms/{id}", roomHandler.UpdateRoom)
			r.With(middleware.RequirePermission("rooms:delete")).
				Delete("/rooms/{id}", roomHandler.DeleteRoom)

			// File Upload Endpoint (Protected by PBAC: uploads:create)
			r.With(middleware.RequirePermission("uploads:create")).
				Post("/uploads", uploadHandler.UploadFile)
			r.With(middleware.RequirePermission("uploads:create")).
				Post("/upload", uploadHandler.UploadFile)

			// User Management (Protected by Granular Permissions: users:read, users:create & users:delete)
			r.With(middleware.RequirePermission("users:read")).
				Get("/users", userHandler.ListUsers)

			r.With(middleware.RequireAnyPermission("users:create", "roles:manage")).
				Post("/users", userHandler.Register)

			r.With(middleware.RequireAnyPermission("users:update", "roles:manage")).
				Put("/users/{id}", userHandler.UpdateUser)

			r.With(middleware.RequirePermission("users:delete")).
				Delete("/users/{id}", userHandler.DeleteUser)

			r.With(middleware.RequirePermission("roles:manage")).
				Put("/users/{id}/role", userHandler.UpdateUserRole)

			// Audit Logs (Protected by PBAC: audit:read)
			r.With(middleware.RequirePermission("audit:read")).
				Get("/audit-logs", auditHandler.ListAuditLogs)

			// Dynamic RBAC & PBAC Management (Admin Only / roles:manage)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAnyPermission("roles:read", "roles:manage"))

				r.Get("/roles", rbacHandler.ListRoles)
				r.Get("/permissions", rbacHandler.ListPermissions)
			})

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission("roles:manage"))

				r.Post("/roles", rbacHandler.CreateRole)
				r.Delete("/roles/{id}", rbacHandler.DeleteRole)
				r.Post("/roles/{role_id}/permissions/{permission_id}", rbacHandler.AssignPermission)
				r.Delete("/roles/{role_id}/permissions/{permission_id}", rbacHandler.RevokePermission)
			})
		})
	})

	return r
}
