package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"gozaq/internal/domain"
)

type mockRBACRepo struct {
	getPermsFunc  func(ctx context.Context, roleID string) ([]string, error)
	listRolesFunc func(ctx context.Context) ([]domain.RoleWithPermissions, error)
	listPermsFunc func(ctx context.Context) ([]domain.Permission, error)
	assignFunc    func(ctx context.Context, roleID, permID string) error
	revokeFunc    func(ctx context.Context, roleID, permID string) error
}

func (m *mockRBACRepo) GetPermissionsByRole(ctx context.Context, roleID string) ([]string, error) {
	if m.getPermsFunc != nil {
		return m.getPermsFunc(ctx, roleID)
	}
	return []string{"users:read"}, nil
}

func (m *mockRBACRepo) ListRoles(ctx context.Context) ([]domain.RoleWithPermissions, error) {
	if m.listRolesFunc != nil {
		return m.listRolesFunc(ctx)
	}
	return []domain.RoleWithPermissions{
		{
			Role: domain.Role{
				ID:        "admin",
				Name:      "Administrator",
				CreatedAt: time.Now(),
			},
			Permissions: []domain.Permission{
				{ID: "users:read", Name: "Read Users", Module: "users"},
			},
		},
	}, nil
}

func (m *mockRBACRepo) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	if m.listPermsFunc != nil {
		return m.listPermsFunc(ctx)
	}
	return []domain.Permission{
		{ID: "users:read", Name: "Read Users", Module: "users"},
	}, nil
}

func (m *mockRBACRepo) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	if m.assignFunc != nil {
		return m.assignFunc(ctx, roleID, permissionID)
	}
	return nil
}

func (m *mockRBACRepo) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, roleID, permissionID)
	}
	return nil
}

func TestRBACHandler(t *testing.T) {
	t.Run("ListRoles_Success", func(t *testing.T) {
		repo := &mockRBACRepo{}
		h := NewRBACHandler(repo)

		req := httptest.NewRequest("GET", "/roles", nil)
		w := httptest.NewRecorder()

		h.ListRoles(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Roles retrieved successfully")
	})

	t.Run("ListRoles_Error", func(t *testing.T) {
		repo := &mockRBACRepo{
			listRolesFunc: func(ctx context.Context) ([]domain.RoleWithPermissions, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewRBACHandler(repo)

		req := httptest.NewRequest("GET", "/roles", nil)
		w := httptest.NewRecorder()

		h.ListRoles(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("ListPermissions_Success", func(t *testing.T) {
		repo := &mockRBACRepo{}
		h := NewRBACHandler(repo)

		req := httptest.NewRequest("GET", "/permissions", nil)
		w := httptest.NewRecorder()

		h.ListPermissions(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Permissions retrieved successfully")
	})

	t.Run("AssignPermission_Success", func(t *testing.T) {
		repo := &mockRBACRepo{}
		h := NewRBACHandler(repo)

		r := chi.NewRouter()
		r.Post("/roles/{role_id}/permissions/{permission_id}", h.AssignPermission)

		req := httptest.NewRequest("POST", "/roles/admin/permissions/users:delete", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("RevokePermission_Success", func(t *testing.T) {
		repo := &mockRBACRepo{}
		h := NewRBACHandler(repo)

		r := chi.NewRouter()
		r.Delete("/roles/{role_id}/permissions/{permission_id}", h.RevokePermission)

		req := httptest.NewRequest("DELETE", "/roles/admin/permissions/users:delete", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
