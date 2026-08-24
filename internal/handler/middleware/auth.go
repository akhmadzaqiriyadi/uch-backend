package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"gozaq/config"
	"gozaq/internal/domain"
	"gozaq/pkg/response"
)

type contextKey string

const (
	UserIDKey   contextKey = "userID"
	UserRoleKey contextKey = "userRole"
)

func Auth(cfg *config.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "Authorization header is missing")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Unauthorized(w, "Invalid authorization format. Format must be 'Bearer <token>'")
				return
			}

			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(cfg.JWT.Secret), nil
			})

			if err != nil || !token.Valid {
				response.Unauthorized(w, "Invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				response.Unauthorized(w, "Invalid token claims")
				return
			}

			userIDStr, ok := claims["user_id"].(string)
			if !ok {
				response.Unauthorized(w, "User ID not found in token")
				return
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				response.Unauthorized(w, "Invalid user ID format in token")
				return
			}

			role, _ := claims["role"].(string)
			if role == "" {
				role = "user"
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UserRoleKey, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks if the authenticated user has one of the allowed roles
func RequireRole(allowedRoles ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentRole := GetUserRole(r.Context())

			hasRole := false
			for _, role := range allowedRoles {
				if currentRole == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				response.Forbidden(w, "Forbidden: insufficient role permissions", map[string]any{
					"required_roles": allowedRoles,
					"your_role":      currentRole,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetUserID(ctx context.Context) (uuid.UUID, error) {
	val := ctx.Value(UserIDKey)
	if val == nil {
		return uuid.Nil, domain.ErrUnauthorized
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return userID, nil
}

func GetUserRole(ctx context.Context) string {
	val := ctx.Value(UserRoleKey)
	if val == nil {
		return ""
	}
	role, ok := val.(string)
	if !ok {
		return ""
	}
	return role
}
