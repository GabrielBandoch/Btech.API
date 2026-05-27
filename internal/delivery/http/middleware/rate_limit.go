package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
)

type ipLimiter struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu         sync.Mutex
	ips        map[string]*ipLimiter
	rate       float64 // tokens added per second
	capacity   float64 // maximum burst capacity
}

// NewRateLimiter instantiates an IP-based token bucket rate limiter.
func NewRateLimiter(rate float64, capacity float64) *RateLimiter {
	rl := &RateLimiter{
		ips:      make(map[string]*ipLimiter),
		rate:     rate,
		capacity: capacity,
	}
	// Start background cleanup routine every 10 minutes
	rl.startCleanup(10 * time.Minute)
	return rl
}

// Limit returns a Chi compatible middleware.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		rl.mu.Lock()
		limiter, exists := rl.ips[ip]
		now := time.Now()

		if !exists {
			limiter = &ipLimiter{
				tokens:     rl.capacity,
				lastRefill: now,
			}
			rl.ips[ip] = limiter
		} else {
			// Refill tokens based on time elapsed
			elapsed := now.Sub(limiter.lastRefill).Seconds()
			limiter.tokens += elapsed * rl.rate
			if limiter.tokens > rl.capacity {
				limiter.tokens = rl.capacity
			}
			limiter.lastRefill = now
		}

		if limiter.tokens >= 1.0 {
			limiter.tokens -= 1.0
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
		} else {
			rl.mu.Unlock()
			response.Error(w, http.StatusTooManyRequests, "too many requests - please try again later")
		}
	})
}

// startCleanup periodically removes stale IP entries to prevent memory leaks.
func (rl *RateLimiter) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for ip, limiter := range rl.ips {
				// Remove entry if it has been idle and bucket is fully refilled
				if now.Sub(limiter.lastRefill) > 1*time.Hour && limiter.tokens >= rl.capacity {
					delete(rl.ips, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
}

// getClientIP extracts client IP address, handling proxy headers safely.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
