package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"gozaq/internal/domain"
	"gozaq/pkg/response"
)

type RBACHandler struct {
	repo domain.RBACRepository
}

func NewRBACHandler(repo domain.RBACRepository) *RBACHandler {
	return &RBACHandler{repo: repo}
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

	response.OK(w, "Permission revoked from role successfully", nil)
}
