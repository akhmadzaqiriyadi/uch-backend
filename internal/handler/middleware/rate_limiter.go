package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"gozaq/pkg/response"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*client
	rate    rate.Limit
	burst   int
}

// NewRateLimiter creates a new in-memory rate limiter
// r: number of allowed requests per second (RPS)
// b: burst capacity (maximum surge)
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*client),
		rate:    r,
		burst:   b,
	}

	// Background cleanup of stale IP entries every 3 minutes
	go rl.cleanupStaleClients(3 * time.Minute)

	return rl
}

func (rl *RateLimiter) Limit() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)

			limiter := rl.getClientLimiter(ip)
			if !limiter.Allow() {
				w.Header().Set("Retry-After", "1")
				response.TooManyRequests(w, "Too many requests. Please slow down.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) getClientLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, exists := rl.clients[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.clients[ip] = &client{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	c.lastSeen = time.Now()
	return c.limiter
}

func (rl *RateLimiter) cleanupStaleClients(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		rl.mu.Lock()
		for ip, c := range rl.clients {
			if time.Since(c.lastSeen) > 5*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func getIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
