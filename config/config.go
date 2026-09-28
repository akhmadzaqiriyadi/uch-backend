package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	SMTP     SMTPConfig
	Storage  StorageConfig
	CORS     CORSConfig
	Seed     SeedConfig
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RedisConfig struct {
	Driver string // "redis" or "memory" / "noop"
	URL    string
}

type AppConfig struct {
	Name         string
	BaseURL      string
	Port         string
	Env          string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type SMTPConfig struct {
	Driver   string // "smtp" or "log" / "mock"
	Host     string
	Port     int
	User     string
	Password string
	From     string
	FromName string
}

type StorageConfig struct {
	Driver      string // "local" or "s3" / "r2" / "minio"
	LocalDir    string
	MaxFileSize int64
	S3Bucket    string
	S3Region    string
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool
	S3SSLVerify bool
	S3PublicURL string
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

type SeedConfig struct {
	AdminEmail    string
	AdminPassword string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, loading from environment variables")
	}

	port := getEnv("APP_PORT", "8080")

	// Parse CORS allowed origins (comma-separated, default: "*")
	corsRaw := getEnv("CORS_ALLOWED_ORIGINS", "*")
	var allowedOrigins []string
	for _, origin := range strings.Split(corsRaw, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			allowedOrigins = append(allowedOrigins, trimmed)
		}
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	return &Config{
		App: AppConfig{
			Name:         getEnv("APP_NAME", "gozaq-api"),
			BaseURL:      getEnv("APP_BASE_URL", "http://localhost:"+port),
			Port:         port,
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
			Driver: strings.ToLower(getEnv("CACHE_DRIVER", "redis")),
			URL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "super-secret-jwt-key-change-in-production"),
			ExpireMinutes: getEnvInt("JWT_EXPIRE_MINUTES", 1440), // 24 hours
		},
		SMTP: SMTPConfig{
			Driver:   strings.ToLower(getEnv("MAILER_DRIVER", "smtp")),
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnvInt("SMTP_PORT", 587),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "no-reply@gozaq.com"),
			FromName: getEnv("SMTP_FROM_NAME", "Gozaq Team"),
		},
		Storage: StorageConfig{
			Driver:      strings.ToLower(getEnv("STORAGE_DRIVER", "local")),
			LocalDir:    getEnv("STORAGE_LOCAL_DIR", "./uploads"),
			MaxFileSize: int64(getEnvInt("STORAGE_MAX_FILE_SIZE_MB", 5)) << 20,
			S3Bucket:    getEnv("S3_BUCKET", ""),
			S3Region:    getEnv("S3_REGION", "us-east-1"),
			S3Endpoint:  getEnv("S3_ENDPOINT", ""),
			S3AccessKey: getEnv("S3_ACCESS_KEY", ""),
			S3SecretKey: getEnv("S3_SECRET_KEY", ""),
			S3UseSSL:    getEnvBool("S3_USE_SSL", true),
			S3SSLVerify: getEnvBool("S3_SSL_VERIFY", false),
			S3PublicURL: getEnv("S3_PUBLIC_URL", ""),
		},
		CORS: CORSConfig{
			AllowedOrigins: allowedOrigins,
		},
		Seed: SeedConfig{
			AdminEmail:    getEnv("INITIAL_ADMIN_EMAIL", "admin@gozaq.com"),
			AdminPassword: getEnv("INITIAL_ADMIN_PASSWORD", "Admin123!"),
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

func getEnvBool(key string, defaultVal bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
