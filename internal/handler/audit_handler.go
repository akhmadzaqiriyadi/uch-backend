package handler

import (
	"net/http"

	"gozaq/internal/domain"
	"gozaq/pkg/pagination"
	"gozaq/pkg/response"
)

type AuditHandler struct {
	repo domain.AuditRepository
}

func NewAuditHandler(repo domain.AuditRepository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	params := pagination.FromRequest(r)
	action := r.URL.Query().Get("action")

	logs, totalItems, err := h.repo.List(r.Context(), params, action)
	if err != nil {
		response.InternalServerError(w, "Failed to retrieve audit logs", err.Error())
		return
	}

	meta := pagination.BuildMeta(params, totalItems)

	response.OK(w, "Audit logs retrieved successfully", domain.PaginatedAuditLogsResponse{
		Logs: logs,
		Meta: meta,
	})
}
