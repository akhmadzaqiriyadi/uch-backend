package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"gozaq/config"
)

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=31536000")
}

func TestAuthMiddleware(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "super-secret-key",
		},
	}

	userID := uuid.New()
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":     userID.String(),
		"email":       "test@example.com",
		"role":        "admin",
		"permissions": []string{"users:read", "users:delete"},
		"exp":         time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := validToken.SignedString([]byte(cfg.JWT.Secret))
	require.NoError(t, err)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, err := GetUserID(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, userID, uid)
		assert.Equal(t, "admin", GetUserRole(r.Context()))
		assert.Equal(t, []string{"users:read", "users:delete"}, GetUserPermissions(r.Context()))
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Valid Token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		Auth(cfg)(nextHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Missing Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		Auth(cfg)(nextHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid Format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Token "+tokenString)
		w := httptest.NewRecorder()

		Auth(cfg)(nextHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid Secret / Signature", func(t *testing.T) {
		badToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": userID.String(),
		}).SignedString([]byte("wrong-secret"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+badToken)
		w := httptest.NewRecorder()

		Auth(cfg)(nextHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestPermissionsAndRoles(t *testing.T) {
	t.Run("RequireRole", func(t *testing.T) {
		guard := RequireRole("admin", "manager")
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Allowed
		req := httptest.NewRequest("GET", "/", nil)
		ctx := context.WithValue(req.Context(), UserRoleKey, "admin")
		w := httptest.NewRecorder()
		guard(next).ServeHTTP(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusOK, w.Code)

		// Denied
		reqDenied := httptest.NewRequest("GET", "/", nil)
		ctxDenied := context.WithValue(reqDenied.Context(), UserRoleKey, "user")
		wDenied := httptest.NewRecorder()
		guard(next).ServeHTTP(wDenied, reqDenied.WithContext(ctxDenied))
		assert.Equal(t, http.StatusForbidden, wDenied.Code)
	})

	t.Run("RequirePermission", func(t *testing.T) {
		guard := RequirePermission("users:delete")
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Allowed
		req := httptest.NewRequest("GET", "/", nil)
		ctx := context.WithValue(req.Context(), UserPermissionsKey, []string{"users:read", "users:delete"})
		w := httptest.NewRecorder()
		guard(next).ServeHTTP(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusOK, w.Code)

		// Denied
		reqDenied := httptest.NewRequest("GET", "/", nil)
		ctxDenied := context.WithValue(reqDenied.Context(), UserPermissionsKey, []string{"users:read"})
		wDenied := httptest.NewRecorder()
		guard(next).ServeHTTP(wDenied, reqDenied.WithContext(ctxDenied))
		assert.Equal(t, http.StatusForbidden, wDenied.Code)
	})

	t.Run("RequireAnyPermission", func(t *testing.T) {
		guard := RequireAnyPermission("roles:read", "roles:manage")
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Allowed
		req := httptest.NewRequest("GET", "/", nil)
		ctx := context.WithValue(req.Context(), UserPermissionsKey, []string{"roles:read"})
		w := httptest.NewRecorder()
		guard(next).ServeHTTP(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusOK, w.Code)

		// Denied
		reqDenied := httptest.NewRequest("GET", "/", nil)
		ctxDenied := context.WithValue(reqDenied.Context(), UserPermissionsKey, []string{"users:read"})
		wDenied := httptest.NewRecorder()
		guard(next).ServeHTTP(wDenied, reqDenied.WithContext(ctxDenied))
		assert.Equal(t, http.StatusForbidden, wDenied.Code)
	})
}

func TestRecoverer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	panickingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went critically wrong")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	Recoverer(logger)(panickingHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(1), 1)
	handler := limiter.Limit()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"

	// First request succeeds
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Immediate second request gets rate limited
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}
