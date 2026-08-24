package storage

import (
	"fmt"
	"log/slog"
	"strings"

	"gozaq/config"
)

// NewStorage initializes Local or S3 storage driver based on configuration
func NewStorage(cfg *config.Config) (Storage, error) {
	driver := strings.ToLower(cfg.Storage.Driver)

	switch driver {
	case "s3", "r2", "minio":
		slog.Info("Cloud Object Storage initialized",
			slog.String("driver", driver),
			slog.String("bucket", cfg.Storage.S3Bucket),
			slog.String("region", cfg.Storage.S3Region),
		)
		return NewS3Storage(cfg)
	default:
		baseURL := fmt.Sprintf("%s/uploads", strings.TrimRight(cfg.App.BaseURL, "/"))
		slog.Info("Local file storage initialized",
			slog.String("dir", cfg.Storage.LocalDir),
			slog.String("base_url", baseURL),
		)
		return NewLocalStorage(cfg.Storage.LocalDir, baseURL, cfg.Storage.MaxFileSize), nil
	}
}
