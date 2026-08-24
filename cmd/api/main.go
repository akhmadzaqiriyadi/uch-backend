package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gozaq/config"
	"gozaq/internal/handler"
	"gozaq/internal/repository/postgres"
	"gozaq/internal/service"
	"gozaq/pkg/cache"
	"gozaq/pkg/database"
	"gozaq/pkg/mailer"
	"gozaq/pkg/storage"
	"gozaq/pkg/worker"
)

func main() {
	// Initialize Structured Logger (JSON format)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load Config
	cfg := config.Load()
	logger.Info("Starting application",
		slog.String("app_name", cfg.App.Name),
		slog.String("env", cfg.App.Env),
		slog.String("port", cfg.App.Port),
	)

	// 1. Database Connection Pool (PostgreSQL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgxConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		logger.Error("Failed to parse database configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	pgxConfig.MaxConns = int32(cfg.Database.MaxOpenConns)
	pgxConfig.MinConns = int32(cfg.Database.MinIdleConns)
	pgxConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	pgxConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime

	dbPool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		logger.Error("Unable to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		logger.Warn("Database ping failed, continuing anyway (check if DB is ready)", slog.String("error", err.Error()))
	} else {
		logger.Info("Database connection pool established successfully")

		// Automatically apply any pending SQL migrations on startup
		if err := database.AutoMigrate(ctx, dbPool, "./migrations"); err != nil {
			logger.Error("Failed to apply automatic database migrations", slog.String("error", err.Error()))
		}
	}

	// 2. Redis Cache Connection (with Noop fallback if unavailable)
	var cacheClient cache.Cache
	redisCache, err := cache.NewRedisCache(cfg.Redis.URL)
	if err != nil {
		logger.Warn("Redis unavailable, falling back to Noop in-memory cache", slog.String("error", err.Error()))
		cacheClient = cache.NewNoopCache()
	} else {
		logger.Info("Redis cache connection established successfully")
		cacheClient = redisCache
		defer func() {
			_ = redisCache.Close()
		}()
	}

	// 3. Async Background Worker Pool (5 workers, 100 queue capacity)
	workerPool := worker.NewPool(5, 100)
	logger.Info("Async worker pool initialized (5 concurrent workers)")

	// 4. Supporting Infrastructure: Mailer & Local Storage
	appMailer := mailer.NewLogMailer()
	fileStorage := storage.NewLocalStorage("./uploads", fmt.Sprintf("http://localhost:%s/uploads", cfg.App.Port), 5<<20) // 5MB limit

	// 5. Dependency Injection (Clean Architecture Wiring)
	userRepo := postgres.NewUserRepository(dbPool)
	rbacRepo := postgres.NewRBACRepository(dbPool)
	userService := service.NewUserService(userRepo, rbacRepo, cacheClient, workerPool, appMailer, cfg)
	userHandler := handler.NewUserHandler(userService)
	docsHandler := handler.NewDocsHandler()
	healthHandler := handler.NewHealthHandler(cfg, dbPool, cacheClient)
	uploadHandler := handler.NewUploadHandler(fileStorage)
	rbacHandler := handler.NewRBACHandler(rbacRepo)

	// 6. Router Setup
	router := handler.NewRouter(cfg, logger, userHandler, docsHandler, healthHandler, uploadHandler, rbacHandler)

	// 7. HTTP Server Configuration
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
	}

	// Start server in background goroutine
	go func() {
		logger.Info(fmt.Sprintf("Server is listening on port %s", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed to start", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Graceful Shutdown on SIGINT or SIGTERM (Kubernetes/Docker standard)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
	}

	// Gracefully drain worker pool
	logger.Info("Stopping background worker pool...")
	workerPool.Shutdown()

	logger.Info("Server exited cleanly")
}
