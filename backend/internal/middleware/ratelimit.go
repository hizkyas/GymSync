package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// RateLimitRequests is the max number of requests allowed per window.
	RateLimitRequests = 60
	// RateLimitWindow is the sliding window duration.
	RateLimitWindow = time.Minute
)

// RateLimiter returns a Redis-backed sliding window rate limiting middleware.
// It limits requests per IP address.
func RateLimiter(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			key := fmt.Sprintf("rate_limit:%s", ip)

			ctx := context.Background()
			now := time.Now()
			windowStart := now.Add(-RateLimitWindow)

			// Use a Redis pipeline for atomicity and performance
			pipe := rdb.Pipeline()

			// Remove timestamps outside the current window
			pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))
			// Add the current request
			pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixMilli()), Member: now.UnixNano()})
			// Count requests in the window
			countCmd := pipe.ZCard(ctx, key)
			// Reset TTL
			pipe.Expire(ctx, key, RateLimitWindow)

			if _, err := pipe.Exec(ctx); err != nil {
				// On Redis failure, allow the request through (fail open)
				next.ServeHTTP(w, r)
				return
			}

			count := countCmd.Val()

			// Set rate limit headers
			remaining := int64(RateLimitRequests) - count
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", RateLimitRequests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(RateLimitWindow).Unix()))

			if count > int64(RateLimitRequests) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(RateLimitWindow.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded, please slow down"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CheckInRateLimiter is a stricter rate limiter specifically for the check-in endpoint.
// Limits to 10 requests per minute per IP to prevent QR token brute-force.
func CheckInRateLimiter(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			key := fmt.Sprintf("checkin_rate:%s", ip)

			ctx := context.Background()
			now := time.Now()
			windowStart := now.Add(-RateLimitWindow)

			pipe := rdb.Pipeline()
			pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))
			pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixMilli()), Member: now.UnixNano()})
			countCmd := pipe.ZCard(ctx, key)
			pipe.Expire(ctx, key, RateLimitWindow)

			if _, err := pipe.Exec(ctx); err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if countCmd.Val() > 10 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"too many check-in attempts"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// realIP extracts the client's real IP address, respecting common proxy headers.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
