package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"gozaq/internal/domain"
)

type PushSubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewPushSubscriptionRepository(db *pgxpool.Pool) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{db: db}
}

func (r *PushSubscriptionRepository) Upsert(ctx context.Context, sub *domain.PushSubscription) error {
	query := `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, user_agent, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (endpoint) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth,
			user_agent = EXCLUDED.user_agent,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now().UTC()
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		sub.ID,
		sub.UserID,
		sub.Endpoint,
		sub.P256dh,
		sub.Auth,
		sub.UserAgent,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert push subscription: %w", err)
	}
	return nil
}

func (r *PushSubscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.PushSubscription, error) {
	query := `
		SELECT id, user_id, endpoint, p256dh, auth, COALESCE(user_agent, ''), created_at, updated_at
		FROM push_subscriptions
		WHERE user_id = $1
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get push subscriptions for user: %w", err)
	}
	defer rows.Close()

	var subs []domain.PushSubscription
	for rows.Next() {
		var s domain.PushSubscription
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Endpoint,
			&s.P256dh,
			&s.Auth,
			&s.UserAgent,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan push subscription: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *PushSubscriptionRepository) GetByRole(ctx context.Context, role string) ([]domain.PushSubscription, error) {
	query := `
		SELECT ps.id, ps.user_id, ps.endpoint, ps.p256dh, ps.auth, COALESCE(ps.user_agent, ''), ps.created_at, ps.updated_at
		FROM push_subscriptions ps
		JOIN users u ON u.id = ps.user_id
		WHERE u.role = $1 AND u.deleted_at IS NULL
	`
	rows, err := r.db.Query(ctx, query, role)
	if err != nil {
		return nil, fmt.Errorf("failed to get push subscriptions for role %s: %w", role, err)
	}
	defer rows.Close()

	var subs []domain.PushSubscription
	for rows.Next() {
		var s domain.PushSubscription
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Endpoint,
			&s.P256dh,
			&s.Auth,
			&s.UserAgent,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan push subscription: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *PushSubscriptionRepository) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	query := `DELETE FROM push_subscriptions WHERE endpoint = $1`
	_, err := r.db.Exec(ctx, query, endpoint)
	if err != nil {
		return fmt.Errorf("failed to delete push subscription: %w", err)
	}
	return nil
}
