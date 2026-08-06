package web

import (
	"context"
	"net/http"
	"sync"

	"github.com/lewtec/rotulador/internal/domain"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const requestCacheKey contextKey = "request_cache"

// RequestCache holds cached data for a single HTTP request
type RequestCache struct {
	mu     sync.RWMutex
	images []*domain.Image
}

// NewRequestCache creates a new request cache
func NewRequestCache() *RequestCache {
	return &RequestCache{}
}

// GetImages returns cached images if available
func (rc *RequestCache) GetImages() ([]*domain.Image, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.images != nil {
		return rc.images, true
	}
	return nil, false
}

// SetImages caches the images list
func (rc *RequestCache) SetImages(images []*domain.Image) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.images = images
}

// WithRequestCache adds a request cache to the context
func WithRequestCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestCacheKey, NewRequestCache())
}

// GetRequestCache retrieves the request cache from context
func GetRequestCache(ctx context.Context) *RequestCache {
	if cache, ok := ctx.Value(requestCacheKey).(*RequestCache); ok {
		return cache
	}
	return nil
}

// requestCacheMiddleware adds a request cache to the context for each request
func requestCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithRequestCache(r.Context())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
