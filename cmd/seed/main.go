package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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
	logger.Info("🌱 Starting database seeding with gofakeit realistic data...")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("Failed to connect to database for seeding", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	userRepo := postgres.NewUserRepository(dbPool)

	// 1. Seed Default Admin User
	adminPassword, _ := bcrypt.GenerateFromPassword([]byte("Admin123!"), bcrypt.DefaultCost)
	now := time.Now().UTC()

	adminUser := &domain.User{
		ID:        uuid.New(),
		Name:      "System Administrator",
		Email:     "admin@gozaq.com",
		Password:  string(adminPassword),
		Role:      "admin",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := userRepo.Create(ctx, adminUser); err != nil {
		logger.Warn("Admin user already exists or skipped", slog.String("email", adminUser.Email))
	} else {
		logger.Info("👑 Admin user seeded successfully",
			slog.String("email", adminUser.Email),
			slog.String("role", adminUser.Role),
			slog.String("password", "Admin123!"),
		)
	}

	// 2. Seed Realistic Sample Users using gofakeit
	_ = gofakeit.Seed(time.Now().UnixNano())
	defaultPassword, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)

	totalUsers := 20
	createdCount := 0

	for i := 0; i < totalUsers; i++ {
		createdAt := gofakeit.DateRange(time.Now().AddDate(0, -3, 0), time.Now()).UTC()
		u := &domain.User{
			ID:        uuid.New(),
			Name:      gofakeit.Name(),
			Email:     gofakeit.Email(),
			Password:  string(defaultPassword),
			Role:      "user",
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}

		if err := userRepo.Create(ctx, u); err != nil {
			logger.Warn("Duplicate email generated, skipping", slog.String("email", u.Email))
		} else {
			createdCount++
		}
	}

	logger.Info(fmt.Sprintf("🎉 Seeding completed! Seeded %d realistic fake users (Password: Password123!)", createdCount))
}
