package cache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopCache(t *testing.T) {
	c := NewNoopCache()
	ctx := context.Background()

	var dest string
	err := c.Get(ctx, "test-key", &dest)
	assert.ErrorIs(t, err, ErrCacheMiss)

	err = c.Set(ctx, "test-key", "value", time.Minute)
	assert.NoError(t, err)

	err = c.Delete(ctx, "test-key")
	assert.NoError(t, err)

	err = c.Close()
	assert.NoError(t, err)
}

func TestRedisLocker_NilClientFallback(t *testing.T) {
	locker := NewRedisLocker(nil)
	ctx := context.Background()

	token, err := locker.Acquire(ctx, "resource-key", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "noop-lock-token", token)

	err = locker.Release(ctx, "resource-key", token)
	assert.NoError(t, err)

	executed := false
	err = locker.WithLock(ctx, "resource-key", time.Minute, func(ctx context.Context) error {
		executed = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestRedisRateLimiter_NilClientFallback(t *testing.T) {
	limiter := NewRedisRateLimiter(nil, 10, time.Minute)
	ctx := context.Background()

	allowed, remaining, err := limiter.Allow(ctx, "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(10), remaining)

	// Test Middleware behavior with fallback
	handler := limiter.Limit()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
}

func TestGetClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	assert.Equal(t, "10.0.0.1", getClientIP(req))

	req.RemoteAddr = "invalid-address"
	assert.Equal(t, "invalid-address", getClientIP(req))
}
