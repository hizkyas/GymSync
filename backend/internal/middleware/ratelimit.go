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
	return SlidingWindowLimiter(rdb, RateLimitRequests, "rate_limit", RateLimitWindow)
}

// CheckInRateLimiter is a stricter rate limiter specifically for the check-in endpoint.
// Limits to 10 requests per minute per IP to prevent QR token brute-force.
func CheckInRateLimiter(rdb *redis.Client) func(http.Handler) http.Handler {
	return SlidingWindowLimiter(rdb, 10, "checkin_rate", RateLimitWindow)
}

// RateLimitRedis is the interface that abstracts Redis operations used by
// the sliding-window rate limiters. This allows unit testing without a live
// Redis connection.
type RateLimitRedis interface {
	Pipeline() redis.Pipeliner
}

// SlidingWindowLimiter returns a testable, IP-keyed sliding window middleware
// backed by any RateLimitRedis implementation. limit is the max requests
// allowed per window; keyPrefix distinguishes different endpoint limiters.
func SlidingWindowLimiter(rdb RateLimitRedis, limit int64, keyPrefix string, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			key := fmt.Sprintf("%s:%s", keyPrefix, ip)

			ctx := context.Background()
			now := time.Now()
			windowStart := now.Add(-window)

			pipe := rdb.Pipeline()
			pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))
			pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixMilli()), Member: now.UnixNano()})
			countCmd := pipe.ZCard(ctx, key)
			pipe.Expire(ctx, key, window)

			if _, err := pipe.Exec(ctx); err != nil {
				// On Redis failure, allow the request through (fail open)
				next.ServeHTTP(w, r)
				return
			}

			count := countCmd.Val()
			remaining := limit - count
			if remaining < 0 {
				remaining = 0
			}

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(window).Unix()))

			if count > limit {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded, please slow down"}`))
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
