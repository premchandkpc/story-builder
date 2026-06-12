package api

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	"github.com/premchand/story-builder/internal/cache"
)

type contextKey string

const (
	ctxTenantID contextKey = "tenant_id"
)

func RateLimitMiddleware(limiter *cache.SlidingWindowRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := tenantFromRequest(r)
			key := fmt.Sprintf("http:api:%s", tenantID)
			allowed, err := limiter.Allow(r.Context(), key)
			if err != nil {
				http.Error(w, "rate limit check failed", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			ctx := context.WithValue(r.Context(), ctxTenantID, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GenerationRateLimitMiddleware(limiter *cache.SlidingWindowRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := fmt.Sprintf("node:generate:%s", r.RemoteAddr)
			allowed, err := limiter.Allow(r.Context(), key)
			if err != nil {
				http.Error(w, "rate limit check failed", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "generation rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func tenantFromRequest(r *http.Request) string {
	if id := r.Header.Get("X-Tenant-ID"); id != "" {
		return id
	}
	if id := r.Header.Get("Authorization"); id != "" {
		return fmt.Sprintf("token:%s", hashString(id))
	}
	parts := strings.SplitN(r.RemoteAddr, ":", 2)
	if len(parts) > 0 {
		return fmt.Sprintf("ip:%s", parts[0])
	}
	return "anonymous"
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:8])
}
