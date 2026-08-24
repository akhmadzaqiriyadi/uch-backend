package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"gozaq/internal/domain"
	"gozaq/pkg/pagination"
)

type mockAuditRepo struct {
	createFunc func(ctx context.Context, log *domain.AuditLog) error
	listFunc   func(ctx context.Context, p pagination.Params, action string) ([]domain.AuditLog, int, error)
}

func (m *mockAuditRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, log)
	}
	return nil
}

func (m *mockAuditRepo) List(ctx context.Context, p pagination.Params, action string) ([]domain.AuditLog, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, p, action)
	}
	uid := uuid.New()
	return []domain.AuditLog{
		{
			ID:        uuid.New(),
			UserID:    &uid,
			Action:    "user.login",
			Entity:    "users",
			CreatedAt: time.Now(),
		},
	}, 1, nil
}

func TestAuditHandler_ListAuditLogs(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &mockAuditRepo{}
		h := NewAuditHandler(repo)

		req := httptest.NewRequest("GET", "/audit-logs?page=1&per_page=10&action=user.login", nil)
		w := httptest.NewRecorder()

		h.ListAuditLogs(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Audit logs retrieved successfully")
	})

	t.Run("Error", func(t *testing.T) {
		repo := &mockAuditRepo{
			listFunc: func(ctx context.Context, p pagination.Params, action string) ([]domain.AuditLog, int, error) {
				return nil, 0, errors.New("database failure")
			},
		}
		h := NewAuditHandler(repo)

		req := httptest.NewRequest("GET", "/audit-logs", nil)
		w := httptest.NewRecorder()

		h.ListAuditLogs(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
