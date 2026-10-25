package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type PostRateLimiter struct {
	rdb   *redis.Client
	limit int
}

func NewPostRateLimiter(rdb *redis.Client, postPerMinute int) *PostRateLimiter {
	return &PostRateLimiter{rdb: rdb, limit: postPerMinute}
}

func (l *PostRateLimiter) Middleware(next http.Handler) http.Handler {
	if l.limit <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		ip := clientIP(r)
		bucket := time.Now().Unix() / 60
		key := fmt.Sprintf("ratelimit:post:%s:%d", ip, bucket)

		n, err := l.rdb.Incr(r.Context(), key).Result()
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		if n == 1 {
			_ = l.rdb.Expire(r.Context(), key, 2*time.Minute).Err()
		}
		if n > int64(l.limit) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// AllowPost is used from tests or direct handler checks.
func (l *PostRateLimiter) AllowPost(ctx context.Context, ip string) bool {
	if l.limit <= 0 {
		return true
	}
	bucket := time.Now().Unix() / 60
	key := fmt.Sprintf("ratelimit:post:%s:%d", ip, bucket)
	n, err := l.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true
	}
	if n == 1 {
		_ = l.rdb.Expire(ctx, key, 2*time.Minute).Err()
	}
	return n <= int64(l.limit)
}
