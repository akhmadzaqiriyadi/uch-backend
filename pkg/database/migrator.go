package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AutoMigrate applies pending SQL migrations from the migrations directory
func AutoMigrate(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	// 1. Create schema_migrations table if not exists
	initQuery := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := pool.Exec(ctx, initQuery); err != nil {
		return fmt.Errorf("failed to initialize schema_migrations table: %w", err)
	}

	// 2. Read migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("Migrations directory not found, skipping auto-migration", slog.String("dir", migrationsDir))
			return nil
		}
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var upFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}
	sort.Strings(upFiles)

	// 3. Apply pending migrations
	appliedCount := 0
	for _, fileName := range upFiles {
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`
		if err := pool.QueryRow(ctx, checkQuery, fileName).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", fileName, err)
		}

		if exists {
			continue
		}

		slog.Info("🔄 Applying database migration", slog.String("file", fileName))

		filePath := filepath.Join(migrationsDir, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", fileName, err)
		}

		err = WithinTransaction(ctx, pool, func(txCtx context.Context) error {
			tx := GetDBTX(txCtx, pool)
			if _, execErr := tx.Exec(txCtx, string(content)); execErr != nil {
				return fmt.Errorf("failed executing migration %s: %w", fileName, execErr)
			}

			recordQuery := `INSERT INTO schema_migrations (version) VALUES ($1)`
			if _, recErr := tx.Exec(txCtx, recordQuery, fileName); recErr != nil {
				return fmt.Errorf("failed recording migration %s: %w", fileName, recErr)
			}
			return nil
		})

		if err != nil {
			return err
		}

		appliedCount++
		slog.Info("✅ Applied database migration", slog.String("file", fileName))
	}

	if appliedCount == 0 {
		slog.Info("Database schema is up to date (no pending migrations)")
	} else {
		slog.Info(fmt.Sprintf("🎉 Successfully applied %d new database migration(s)", appliedCount))
	}

	return nil
}
