package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"gozaq/config"
	"gozaq/internal/domain"
	"gozaq/internal/repository/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	// Parse command line flags
	envFlag := flag.String("env", cfg.App.Env, "Target environment mode: 'dev' or 'prod'")
	adminEmailFlag := flag.String("email", cfg.Seed.AdminEmail, "Admin email address")
	adminPassFlag := flag.String("password", cfg.Seed.AdminPassword, "Admin password")
	forceFakeFlag := flag.Bool("force-fake-users", false, "Force generating 20 fake users even in production mode")
	flag.Parse()

	targetEnv := strings.ToLower(*envFlag)
	if targetEnv == "development" {
		targetEnv = "dev"
	}
	if targetEnv == "production" {
		targetEnv = "prod"
	}

	logger.Info("🌱 Initializing Database Seeder",
		slog.String("target_env", targetEnv),
		slog.String("admin_email", *adminEmailFlag),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("Failed to connect to database for seeding", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	userRepo := postgres.NewUserRepository(dbPool)
	now := time.Now().UTC()

	// 1. Seed Initial Admin Account
	adminPassword := *adminPassFlag
	if len(adminPassword) < 8 {
		logger.Error("Admin password must be at least 8 characters for security")
		os.Exit(1)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Failed to hash admin password", slog.String("error", err.Error()))
		os.Exit(1)
	}

	adminUser := &domain.User{
		ID:         uuid.New(),
		Name:       "System Administrator",
		Email:      *adminEmailFlag,
		Password:   string(hashedPassword),
		Role:       "admin",
		IsVerified: true,
		VerifiedAt: &now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := userRepo.Create(ctx, adminUser); err != nil {
		logger.Warn("Admin user already exists in database (skipped creation)", slog.String("email", adminUser.Email))
	} else {
		if targetEnv == "dev" {
			logger.Info("👑 [DEV] Admin user seeded successfully",
				slog.String("email", adminUser.Email),
				slog.String("role", adminUser.Role),
				slog.String("password", adminPassword),
			)
		} else {
			logger.Info("👑 [PROD] Root Admin seeded successfully (Password hidden for security)",
				slog.String("email", adminUser.Email),
				slog.String("role", adminUser.Role),
			)
		}
	}

	// 2. Production Security Gate: Prevent seeding fake data into production
	if targetEnv == "prod" && !*forceFakeFlag {
		logger.Info("🔒 Production mode active: Fake user generation strictly skipped for data safety.")
		logger.Info("🎉 Initial production seeding completed successfully!")
		return
	}

	// 3. Seed Realistic Sample Users using gofakeit (Dev Mode Only)
	logger.Info("🎭 Generating 20 realistic fake users with gofakeit...")
	_ = gofakeit.Seed(time.Now().UnixNano())
	defaultUserPass, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)

	totalUsers := 20
	createdCount := 0

	for i := 0; i < totalUsers; i++ {
		createdAt := gofakeit.DateRange(time.Now().AddDate(0, -3, 0), time.Now()).UTC()
		u := &domain.User{
			ID:         uuid.New(),
			Name:       gofakeit.Name(),
			Email:      gofakeit.Email(),
			Password:   string(defaultUserPass),
			Role:       "user",
			IsVerified: true,
			VerifiedAt: &createdAt,
			CreatedAt:  createdAt,
			UpdatedAt:  createdAt,
		}

		if err := userRepo.Create(ctx, u); err != nil {
			logger.Warn("Duplicate email generated, skipping", slog.String("email", u.Email))
		} else {
			createdCount++
		}
	}

	logger.Info(fmt.Sprintf("🎉 Seeding completed! Seeded %d realistic fake users (Password: Password123!)", createdCount))
}
