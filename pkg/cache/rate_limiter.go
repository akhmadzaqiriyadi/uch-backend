package cache

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"gozaq/pkg/response"
)

// RedisRateLimiter implements a distributed sliding-window rate limiter powered by Redis
type RedisRateLimiter struct {
	client *redis.Client
	limit  int64
	window time.Duration
}

func NewRedisRateLimiter(client *redis.Client, limit int64, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

// Allow checks if the given identifier (IP/UserID) is within the allowed limit
func (r *RedisRateLimiter) Allow(ctx context.Context, identifier string) (bool, int64, error) {
	if r.client == nil {
		return true, r.limit, nil
	}

	key := fmt.Sprintf("ratelimit:%s", identifier)
	now := time.Now().UnixNano()
	clearBefore := now - r.window.Nanoseconds()

	// Sliding window using Redis Sorted Set (ZSET)
	pipe := r.client.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(clearBefore, 10))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, r.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		// Fallback to allowing request if Redis fails
		return true, r.limit, err
	}

	currentCount := countCmd.Val()
	remaining := r.limit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	return currentCount <= r.limit, remaining, nil
}

// Limit returns a middleware that enforces Redis-based distributed rate limiting
func (r *RedisRateLimiter) Limit() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ip := getClientIP(req)
			allowed, remaining, _ := r.Allow(req.Context(), ip)

			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(r.limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

			if !allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(r.window.Seconds()), 10))
				response.TooManyRequests(w, "Distributed rate limit exceeded. Please slow down.")
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
