package middleware

import (
	"fmt"
	"net/http"
	"time"

	rdb "codeberg.org/azzet/azzetbe/internal/redis"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

// RateLimit limits requests per client IP within the given window.
func RateLimit(redis *rdb.Redis, prefix string, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimitByIP(redis, prefix, maxRequests, window)
}

// RateLimitByIP limits requests per client IP within the given window.
func RateLimitByIP(redis *rdb.Redis, prefix string, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetClientIP(r)
			key := fmt.Sprintf("ratelimit:%s:%s", prefix, ip)

			count, err := redis.Incr(r.Context(), key).Result()
			if err != nil {
				shared.InternalError(w, r, "ratelimit", "failed to check rate limit")
				return
			}
			if count == 1 {
				_ = redis.Expire(r.Context(), key, window).Err()
			}
			if count > int64(maxRequests) {
				shared.Error(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "ratelimit", "too many requests, try again later")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
