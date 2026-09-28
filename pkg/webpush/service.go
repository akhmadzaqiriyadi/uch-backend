package webpush

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	webpush "github.com/SherClockHolmes/webpush-go"

	"gozaq/config"
	"gozaq/internal/domain"
)

type PushPayload struct {
	Title     string `json:"title"`
	Message   string `json:"message"`
	Body      string `json:"body"`
	URL       string `json:"url,omitempty"`
	BookingID string `json:"bookingId,omitempty"`
	Timestamp string `json:"timestamp"`
}

type Service struct {
	repo domain.PushSubscriptionRepository
	cfg  config.WebPushConfig
}

func NewService(repo domain.PushSubscriptionRepository, cfg config.WebPushConfig) *Service {
	return &Service{
		repo: repo,
		cfg:  cfg,
	}
}

func (s *Service) VAPIDPublicKey() string {
	return s.cfg.VAPIDPublicKey
}

func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth, userAgent string) error {
	sub := &domain.PushSubscription{
		ID:        uuid.New(),
		UserID:    userID,
		Endpoint:  endpoint,
		P256dh:    p256dh,
		Auth:      auth,
		UserAgent: userAgent,
	}
	return s.repo.Upsert(ctx, sub)
}

func (s *Service) Unsubscribe(ctx context.Context, endpoint string) error {
	return s.repo.DeleteByEndpoint(ctx, endpoint)
}

func (s *Service) SendToUser(ctx context.Context, userID uuid.UUID, payload PushPayload) {
	subs, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		slog.Error("Failed to fetch push subscriptions for user", slog.String("user_id", userID.String()), slog.String("error", err.Error()))
		return
	}
	if len(subs) == 0 {
		slog.Debug("No push subscriptions found for user", slog.String("user_id", userID.String()))
		return
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal push payload", slog.String("error", err.Error()))
		return
	}

	slog.Info("Sending Web Push notification to user",
		slog.String("user_id", userID.String()),
		slog.String("title", payload.Title),
		slog.Int("subscriptions_count", len(subs)),
	)

	for _, sub := range subs {
		go s.dispatchPush(sub, payloadBytes)
	}
}

func (s *Service) SendToRole(ctx context.Context, role string, payload PushPayload) {
	subs, err := s.repo.GetByRole(ctx, role)
	if err != nil {
		slog.Error("Failed to fetch push subscriptions for role", slog.String("role", role), slog.String("error", err.Error()))
		return
	}
	if len(subs) == 0 {
		slog.Debug("No push subscriptions found for role", slog.String("role", role))
		return
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal push payload", slog.String("error", err.Error()))
		return
	}

	slog.Info("Sending Web Push notification to role",
		slog.String("role", role),
		slog.String("title", payload.Title),
		slog.Int("subscriptions_count", len(subs)),
	)

	for _, sub := range subs {
		go s.dispatchPush(sub, payloadBytes)
	}
}

func (s *Service) dispatchPush(sub domain.PushSubscription, payload []byte) {
	wpSub := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	resp, err := webpush.SendNotification(payload, wpSub, &webpush.Options{
		Subscriber:      s.cfg.VAPIDSubject,
		VAPIDPublicKey:  s.cfg.VAPIDPublicKey,
		VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
		TTL:             86400, // 24 hours
		Urgency:         webpush.UrgencyHigh,
	})

	if err != nil {
		slog.Warn("Failed to send web push notification",
			slog.String("endpoint", sub.Endpoint),
			slog.String("error", err.Error()),
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		slog.Info("Subscription expired or unregistered, removing from database",
			slog.String("endpoint", sub.Endpoint),
			slog.Int("status_code", resp.StatusCode),
		)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.repo.DeleteByEndpoint(ctx, sub.Endpoint)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("Web Push notification delivered successfully",
			slog.String("user_id", sub.UserID.String()),
			slog.Int("status_code", resp.StatusCode),
		)
	} else {
		slog.Warn("Web Push returned non-2xx status",
			slog.String("endpoint", sub.Endpoint),
			slog.Int("status_code", resp.StatusCode),
		)
	}
}
