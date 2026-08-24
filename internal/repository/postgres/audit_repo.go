package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"gozaq/internal/domain"
	"gozaq/pkg/pagination"
)

type AuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, user_id, action, entity, entity_id, ip_address, user_agent, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	var detailsJSON []byte
	if log.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			detailsJSON = nil
		}
	}

	_, err := r.db.Exec(ctx, query,
		log.ID,
		log.UserID,
		log.Action,
		log.Entity,
		log.EntityID,
		log.IPAddress,
		log.UserAgent,
		detailsJSON,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

func (r *AuditRepository) List(ctx context.Context, p pagination.Params, action string) ([]domain.AuditLog, int, error) {
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE ($1 = '' OR action = $1)`
	var totalItems int
	if err := r.db.QueryRow(ctx, countQuery, action).Scan(&totalItems); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	orderDir := "DESC"
	if p.Order == "asc" {
		orderDir = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, action, entity, entity_id, ip_address, user_agent, details, created_at
		FROM audit_logs
		WHERE ($1 = '' OR action = $1)
		ORDER BY created_at %s
		LIMIT $2 OFFSET $3
	`, orderDir)

	rows, err := r.db.Query(ctx, query, action, p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var detailsJSON []byte
		if err := rows.Scan(
			&l.ID,
			&l.UserID,
			&l.Action,
			&l.Entity,
			&l.EntityID,
			&l.IPAddress,
			&l.UserAgent,
			&detailsJSON,
			&l.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &l.Details)
		}

		logs = append(logs, l)
	}

	if logs == nil {
		logs = []domain.AuditLog{}
	}

	return logs, totalItems, nil
}
