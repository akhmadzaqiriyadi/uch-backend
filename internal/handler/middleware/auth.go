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
	UserIDKey          contextKey = "userID"
	UserRoleKey        contextKey = "userRole"
	UserPermissionsKey contextKey = "userPermissions"
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

			var permissions []string
			if rawPerms, exists := claims["permissions"].([]any); exists {
				for _, p := range rawPerms {
					if pStr, ok := p.(string); ok {
						permissions = append(permissions, pStr)
					}
				}
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UserRoleKey, role)
			ctx = context.WithValue(ctx, UserPermissionsKey, permissions)

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

// RequirePermission checks if the authenticated user has ALL required granular permissions
func RequirePermission(requiredPermissions ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Super admin bypasses all permission checks
			if GetUserRole(r.Context()) == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			userPerms := GetUserPermissions(r.Context())
			permMap := make(map[string]bool)
			for _, p := range userPerms {
				permMap[p] = true
			}

			for _, required := range requiredPermissions {
				if !permMap[required] {
					response.Forbidden(w, fmt.Sprintf("Forbidden: missing required permission '%s'", required), map[string]any{
						"required_permissions": requiredPermissions,
						"your_permissions":     userPerms,
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission checks if the authenticated user has AT LEAST ONE of the listed permissions
func RequireAnyPermission(permissions ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if GetUserRole(r.Context()) == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			userPerms := GetUserPermissions(r.Context())
			permMap := make(map[string]bool)
			for _, p := range userPerms {
				permMap[p] = true
			}

			for _, p := range permissions {
				if permMap[p] {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Forbidden(w, "Forbidden: missing required permission", map[string]any{
				"one_of_permissions": permissions,
				"your_permissions":   userPerms,
			})
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

func GetUserPermissions(ctx context.Context) []string {
	val := ctx.Value(UserPermissionsKey)
	if val == nil {
		return []string{}
	}
	perms, ok := val.([]string)
	if !ok {
		return []string{}
	}
	return perms
}
