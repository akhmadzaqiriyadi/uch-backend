package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

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
	logger.Info("🌱 Starting database seeding...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("Failed to connect to database for seeding", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	userRepo := postgres.NewUserRepository(dbPool)

	// 1. Seed Admin User
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
		logger.Warn("Admin user might already exist", slog.String("email", adminUser.Email), slog.String("error", err.Error()))
	} else {
		logger.Info("✅ Admin user seeded successfully",
			slog.String("email", adminUser.Email),
			slog.String("role", adminUser.Role),
			slog.String("password", "Admin123!"),
		)
	}

	// 2. Seed Sample Users
	sampleUsers := []struct {
		name     string
		email    string
		password string
	}{
		{"John Doe", "john@example.com", "Password123!"},
		{"Jane Smith", "jane@example.com", "Password123!"},
		{"Bob Johnson", "bob@example.com", "Password123!"},
		{"Alice Williams", "alice@example.com", "Password123!"},
	}

	for _, su := range sampleUsers {
		hashedPw, _ := bcrypt.GenerateFromPassword([]byte(su.password), bcrypt.DefaultCost)
		u := &domain.User{
			ID:        uuid.New(),
			Name:      su.name,
			Email:     su.email,
			Password:  string(hashedPw),
			Role:      "user",
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := userRepo.Create(ctx, u); err != nil {
			logger.Warn("Sample user might already exist", slog.String("email", u.Email))
		} else {
			logger.Info(fmt.Sprintf("✅ Sample user seeded: %s (%s)", u.Name, u.Email))
		}
	}

	logger.Info("🎉 Seeding completed successfully!")
}
