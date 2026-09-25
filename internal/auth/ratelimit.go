package auth

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*tokenBucket
	rateHz   float64 // Requests per second
	capacity int     // Max tokens in bucket
}

// tokenBucket represents a token bucket for an IP address
type tokenBucket struct {
	tokens    float64
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
// rateHz: requests per second, capacity: max tokens per bucket
func NewRateLimiter(rateHz float64, capacity int) *RateLimiter {
	if rateHz <= 0 {
		rateHz = 100.0 // Default: 100 req/sec
	}
	if capacity <= 0 {
		capacity = 1000 // Default: 1000 tokens
	}

	return &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rateHz:   rateHz,
		capacity: capacity,
	}
}

// Allow checks if request should be allowed for the given IP
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[clientIP]
	if !exists {
		bucket = &tokenBucket{
			tokens:    float64(rl.capacity),
			lastReset: time.Now(),
		}
		rl.buckets[clientIP] = bucket
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastReset).Seconds()
	bucket.tokens += elapsed * rl.rateHz
	bucket.lastReset = now

	// Cap at capacity
	if bucket.tokens > float64(rl.capacity) {
		bucket.tokens = float64(rl.capacity)
	}

	// Check if we have tokens
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

// RateLimitMiddleware creates an HTTP middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := r.RemoteAddr
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				clientIP = xff
			}

			if !limiter.Allow(clientIP) {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
