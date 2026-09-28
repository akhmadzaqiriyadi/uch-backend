package handler

import (
	"fmt"
	"log/slog"
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
	authSuccess := false

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
				authSuccess = true
			}
		} else {
			slog.Warn("WebSocket token invalid or expired", slog.Any("error", err), slog.String("remote_addr", r.RemoteAddr))
		}
	}

	if userID == uuid.Nil {
		userID = uuid.New()
	}

	slog.Info("WebSocket upgrading connection",
		slog.String("user_id", userID.String()),
		slog.String("role", role),
		slog.Bool("authenticated", authSuccess),
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("user_agent", r.UserAgent()),
	)

	_ = realtime.ServeWs(h.hub, w, r, userID, role)
}
