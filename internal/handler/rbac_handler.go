package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"gozaq/internal/domain"
	"gozaq/internal/handler/middleware"
	"gozaq/pkg/response"
)

type RBACHandler struct {
	repo      domain.RBACRepository
	auditRepo domain.AuditRepository
}

func NewRBACHandler(repo domain.RBACRepository, auditRepo domain.AuditRepository) *RBACHandler {
	return &RBACHandler{
		repo:      repo,
		auditRepo: auditRepo,
	}
}

func (h *RBACHandler) logAudit(ctx context.Context, action, entityID string, details any) {
	if h.auditRepo == nil {
		return
	}
	var uid *uuid.UUID
	if id, err := middleware.GetUserID(ctx); err == nil {
		uid = &id
	}
	_ = h.auditRepo.Create(ctx, &domain.AuditLog{
		ID:        uuid.New(),
		UserID:    uid,
		Action:    action,
		Entity:    "roles",
		EntityID:  entityID,
		Details:   details,
		CreatedAt: time.Now().UTC(),
	})
}

func (h *RBACHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.repo.ListRoles(r.Context())
	if err != nil {
		response.InternalServerError(w, "Failed to retrieve roles", err.Error())
		return
	}
	response.OK(w, "Roles retrieved successfully", roles)
}

func (h *RBACHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.repo.ListPermissions(r.Context())
	if err != nil {
		response.InternalServerError(w, "Failed to retrieve permissions", err.Error())
		return
	}
	response.OK(w, "Permissions retrieved successfully", perms)
}

func (h *RBACHandler) AssignPermission(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "role_id")
	permID := chi.URLParam(r, "permission_id")

	if roleID == "" || permID == "" {
		response.BadRequest(w, "role_id and permission_id are required", nil)
		return
	}

	if err := h.repo.AssignPermissionToRole(r.Context(), roleID, permID); err != nil {
		response.InternalServerError(w, "Failed to assign permission", err.Error())
		return
	}

	h.logAudit(r.Context(), "permission.assigned", roleID, map[string]string{
		"role_id":       roleID,
		"permission_id": permID,
	})

	response.OK(w, "Permission assigned to role successfully", nil)
}

func (h *RBACHandler) RevokePermission(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "role_id")
	permID := chi.URLParam(r, "permission_id")

	if roleID == "" || permID == "" {
		response.BadRequest(w, "role_id and permission_id are required", nil)
		return
	}

	if err := h.repo.RevokePermissionFromRole(r.Context(), roleID, permID); err != nil {
		response.InternalServerError(w, "Failed to revoke permission", err.Error())
		return
	}

	h.logAudit(r.Context(), "permission.revoked", roleID, map[string]string{
		"role_id":       roleID,
		"permission_id": permID,
	})

	response.OK(w, "Permission revoked from role successfully", nil)
}

func (h *RBACHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON payload", err.Error())
		return
	}

	if req.ID == "" || req.Name == "" {
		response.BadRequest(w, "Role ID and Name are required", nil)
		return
	}

	if err := h.repo.CreateRole(r.Context(), req.ID, req.Name, req.Description); err != nil {
		response.InternalServerError(w, "Failed to create role", err.Error())
		return
	}

	h.logAudit(r.Context(), "role.created", req.ID, req)

	response.Created(w, "Role created successfully", req)
}

func (h *RBACHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	if roleID == "" {
		response.BadRequest(w, "Role ID is required", nil)
		return
	}

	if roleID == "admin" || roleID == "user" {
		response.BadRequest(w, "System default roles cannot be deleted", nil)
		return
	}

	if err := h.repo.DeleteRole(r.Context(), roleID); err != nil {
		response.InternalServerError(w, "Failed to delete role", err.Error())
		return
	}

	h.logAudit(r.Context(), "role.deleted", roleID, map[string]string{"role_id": roleID})

	response.OK(w, "Role deleted successfully", nil)
}
