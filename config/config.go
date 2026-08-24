package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

type RedisConfig struct {
	URL string
}

type AppConfig struct {
	Name         string
	Port         string
	Env          string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DBConfig struct {
	URL             string
	MaxOpenConns    int
	MinIdleConns    int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type JWTConfig struct {
	Secret        string
	ExpireMinutes int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, loading from environment variables")
	}

	return &Config{
		App: AppConfig{
			Name:         getEnv("APP_NAME", "gozaq-api"),
			Port:         getEnv("APP_PORT", "8080"),
			Env:          getEnv("APP_ENV", "development"),
			ReadTimeout:  time.Duration(getEnvInt("APP_READ_TIMEOUT", 10)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("APP_WRITE_TIMEOUT", 10)) * time.Second,
			IdleTimeout:  time.Duration(getEnvInt("APP_IDLE_TIMEOUT", 60)) * time.Second,
		},
		Database: DBConfig{
			URL:             getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/gozaq_db?sslmode=disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MinIdleConns:    getEnvInt("DB_MIN_IDLE_CONNS", 5),
			MaxConnLifetime: time.Duration(getEnvInt("DB_MAX_CONN_LIFETIME", 60)) * time.Minute,
			MaxConnIdleTime: time.Duration(getEnvInt("DB_MAX_CONN_IDLE_TIME", 15)) * time.Minute,
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "super-secret-jwt-key-change-in-production"),
			ExpireMinutes: getEnvInt("JWT_EXPIRE_MINUTES", 1440), // 24 hours
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
