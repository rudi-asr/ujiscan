package api

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter — simple in-memory token bucket per client IP.
// Proteksi API publik (tunnel) dari spam/brute force.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int           // request per window
	window   time.Duration // window duration
	burst    int           // burst allowance
}

type bucket struct {
	tokens    int
	lastRefil time.Time
}

// NewRateLimiter membuat rate limiter.
// limit: maks request per window. burst: token cadangan awal.
func NewRateLimiter(limit int, window time.Duration, burst int) *RateLimiter {
	if limit <= 0 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	if burst <= 0 {
		burst = limit
	}
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    limit,
		window:  window,
		burst:   burst,
	}
}

// allow mengecek apakah request dari IP diizinkan.
func (rl *RateLimiter) allow(ip string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: rl.burst, lastRefil: now}
		rl.buckets[ip] = b
	}

	// Refill token berdasarkan waktu yang lewat
	elapsed := now.Sub(b.lastRefil)
	refill := int(elapsed / (rl.window / time.Duration(rl.rate)))
	if refill > 0 {
		b.tokens = min(rl.burst, b.tokens+refill)
		b.lastRefil = now
	}

	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// Cleanup menghapus bucket lama (hindari memory leak). Panggil berkala.
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, b := range rl.buckets {
		if time.Since(b.lastRefil) > maxAge {
			delete(rl.buckets, ip)
		}
	}
}

// clientIP mengekstrak IP client (forwarded-aware untuk proxy).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// ambil IP pertama (paling dekat client)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	return r.RemoteAddr
}

// RateLimitMiddleware membungkus handler dengan rate limiting per IP.
func (rl *RateLimiter) RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.allow(ip) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate limit exceeded — terlalu banyak request, coba lagi nanti"}`, http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// min helper (Go < 1.21 compat).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}