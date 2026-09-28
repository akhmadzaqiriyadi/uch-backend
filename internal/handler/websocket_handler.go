package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"gozaq/config"
	"gozaq/internal/service"
	"gozaq/pkg/realtime"
)

type WebSocketHandler struct {
	hub *realtime.Hub
	cfg *config.Config
}

func NewWebSocketHandler(hub *realtime.Hub, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
		cfg: cfg,
	}
}

func (h *WebSocketHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	var userID uuid.UUID
	role := "guest"

	if tokenStr != "" {
		token, err := jwt.ParseWithClaims(tokenStr, &service.JWTClaims{}, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(h.cfg.JWT.Secret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(*service.JWTClaims); ok {
				userID = claims.UserID
				role = claims.Role
			}
		}
	}

	if userID == uuid.Nil {
		userID = uuid.New()
	}

	_ = realtime.ServeWs(h.hub, w, r, userID, role)
}
