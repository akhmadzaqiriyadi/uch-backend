package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PushSubscription struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PushSubscriptionRepository interface {
	Upsert(ctx context.Context, sub *PushSubscription) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]PushSubscription, error)
	GetByRole(ctx context.Context, role string) ([]PushSubscription, error)
	DeleteByEndpoint(ctx context.Context, endpoint string) error
}
