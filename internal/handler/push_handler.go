package handler

import (
	"encoding/json"
	"net/http"

	"gozaq/internal/handler/middleware"
	"gozaq/pkg/response"
	"gozaq/pkg/webpush"
)

type PushHandler struct {
	pushService *webpush.Service
}

func NewPushHandler(pushService *webpush.Service) *PushHandler {
	return &PushHandler{pushService: pushService}
}

func (h *PushHandler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	response.OK(w, "VAPID public key retrieved successfully", map[string]string{
		"vapid_public_key": h.pushService.VAPIDPublicKey(),
	})
}

type SubscribePushRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	var req SubscribePushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON payload", err.Error())
		return
	}

	if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		response.BadRequest(w, "Endpoint, p256dh, and auth keys are required", nil)
		return
	}

	userAgent := r.UserAgent()
	if err := h.pushService.Subscribe(r.Context(), userID, req.Endpoint, req.Keys.P256dh, req.Keys.Auth, userAgent); err != nil {
		response.InternalServerError(w, "Failed to register push subscription", err.Error())
		return
	}

	response.OK(w, "Push subscription registered successfully", map[string]bool{"subscribed": true})
}

type UnsubscribePushRequest struct {
	Endpoint string `json:"endpoint"`
}

func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var req UnsubscribePushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON payload", err.Error())
		return
	}

	if req.Endpoint == "" {
		response.BadRequest(w, "Endpoint is required", nil)
		return
	}

	if err := h.pushService.Unsubscribe(r.Context(), req.Endpoint); err != nil {
		response.InternalServerError(w, "Failed to remove push subscription", err.Error())
		return
	}

	response.OK(w, "Push subscription removed successfully", map[string]bool{"unsubscribed": true})
}
