package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"gozaq/pkg/pagination"
)

type AuditLog struct {
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Action    string     `json:"action"`
	Entity    string     `json:"entity"`
	EntityID  string     `json:"entity_id,omitempty"`
	IPAddress string     `json:"ip_address,omitempty"`
	UserAgent string     `json:"user_agent,omitempty"`
	Details   any        `json:"details,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type PaginatedAuditLogsResponse struct {
	Logs []AuditLog      `json:"logs"`
	Meta pagination.Meta `json:"meta"`
}

type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, p pagination.Params, action string) ([]AuditLog, int, error)
}
