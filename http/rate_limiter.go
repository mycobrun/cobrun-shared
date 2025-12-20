// Package http provides HTTP utilities including rate limiting middleware.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiterConfig holds rate limiter configuration.
type RateLimiterConfig struct {
	// RequestsPerSecond is the number of requests allowed per second.
	RequestsPerSecond float64
	// BurstSize is the maximum burst size (bucket capacity).
	BurstSize int
	// KeyFunc extracts the rate limit key from the request (e.g., IP, user ID).
	KeyFunc func(r *http.Request) string
	// ExcludeFunc determines if a request should be excluded from rate limiting.
	ExcludeFunc func(r *http.Request) bool
	// OnLimitExceeded is called when the rate limit is exceeded.
	OnLimitExceeded func(r *http.Request, key string)
	// CleanupInterval is how often to clean up expired entries.
	CleanupInterval time.Duration
}

// DefaultRateLimiterConfig returns sensible production defaults.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
		KeyFunc:           IPKeyFunc,
		CleanupInterval:   time.Minute,
	}
}

// IPKeyFunc extracts the client IP address.
func IPKeyFunc(r *http.Request) string {
	// Check X-Forwarded-For header (behind load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// UserIDKeyFunc extracts the user ID from context.
func UserIDKeyFunc(r *http.Request) string {
	if userID := r.Context().Value("user_id"); userID != nil {
		return fmt.Sprintf("user:%v", userID)
	}
	return IPKeyFunc(r)
}

// CombinedKeyFunc combines IP and path for more granular limiting.
func CombinedKeyFunc(r *http.Request) string {
	return fmt.Sprintf("%s:%s", IPKeyFunc(r), r.URL.Path)
}

// TokenBucket implements the token bucket algorithm.
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket.
func NewTokenBucket(maxTokens float64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so.
func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	// Check if we have a token available
	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

// Tokens returns the current number of available tokens.
func (b *TokenBucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill first
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	tokens := b.tokens + elapsed*b.refillRate
	if tokens > b.maxTokens {
		tokens = b.maxTokens
	}

	return tokens
}

// RateLimiter implements rate limiting using token buckets.
type RateLimiter struct {
	config  RateLimiterConfig
	buckets sync.Map // map[string]*TokenBucket
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())

	rl := &RateLimiter{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		go rl.cleanupLoop()
	}

	return rl
}

// getBucket returns or creates a bucket for the given key.
func (rl *RateLimiter) getBucket(key string) *TokenBucket {
	if bucket, ok := rl.buckets.Load(key); ok {
		return bucket.(*TokenBucket)
	}

	bucket := NewTokenBucket(float64(rl.config.BurstSize), rl.config.RequestsPerSecond)
	actual, _ := rl.buckets.LoadOrStore(key, bucket)
	return actual.(*TokenBucket)
}

// Allow checks if a request should be allowed.
func (rl *RateLimiter) Allow(r *http.Request) bool {
	if rl.config.ExcludeFunc != nil && rl.config.ExcludeFunc(r) {
		return true
	}

	key := rl.config.KeyFunc(r)
	bucket := rl.getBucket(key)

	if !bucket.Allow() {
		if rl.config.OnLimitExceeded != nil {
			rl.config.OnLimitExceeded(r, key)
		}
		return false
	}

	return true
}

// cleanupLoop periodically removes old buckets.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.ctx.Done():
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

// cleanup removes buckets that are full (not recently used).
func (rl *RateLimiter) cleanup() {
	rl.buckets.Range(func(key, value interface{}) bool {
		bucket := value.(*TokenBucket)
		// If bucket is full, it hasn't been used recently
		if bucket.Tokens() >= float64(rl.config.BurstSize)*0.99 {
			rl.buckets.Delete(key)
		}
		return true
	})
}

// Close stops the rate limiter.
func (rl *RateLimiter) Close() {
	rl.cancel()
}

// Middleware returns an HTTP middleware that applies rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(r) {
			key := rl.config.KeyFunc(r)
			bucket := rl.getBucket(key)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.config.BurstSize))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatFloat(bucket.Tokens(), 'f', 0, 64))
			w.Header().Set("Retry-After", "1")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please slow down.",
			})
			return
		}

		// Set rate limit headers for successful requests
		key := rl.config.KeyFunc(r)
		bucket := rl.getBucket(key)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.config.BurstSize))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatFloat(bucket.Tokens(), 'f', 0, 64))

		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc returns a middleware function.
func (rl *RateLimiter) MiddlewareFunc() func(http.Handler) http.Handler {
	return rl.Middleware
}

// RateLimitMiddleware is a convenience function to create rate limiting middleware.
func RateLimitMiddleware(requestsPerSecond float64, burstSize int) func(http.Handler) http.Handler {
	config := DefaultRateLimiterConfig()
	config.RequestsPerSecond = requestsPerSecond
	config.BurstSize = burstSize

	rl := NewRateLimiter(config)
	return rl.MiddlewareFunc()
}

// PerEndpointRateLimiter applies different rate limits per endpoint.
type PerEndpointRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*RateLimiter
	defaults RateLimiterConfig
}

// NewPerEndpointRateLimiter creates a rate limiter with per-endpoint configuration.
func NewPerEndpointRateLimiter(defaults RateLimiterConfig) *PerEndpointRateLimiter {
	return &PerEndpointRateLimiter{
		limiters: make(map[string]*RateLimiter),
		defaults: defaults,
	}
}

// SetEndpointLimit sets a specific limit for an endpoint.
func (pe *PerEndpointRateLimiter) SetEndpointLimit(endpoint string, requestsPerSecond float64, burstSize int) {
	config := pe.defaults
	config.RequestsPerSecond = requestsPerSecond
	config.BurstSize = burstSize

	pe.mu.Lock()
	pe.limiters[endpoint] = NewRateLimiter(config)
	pe.mu.Unlock()
}

// getLimiter returns the limiter for an endpoint.
func (pe *PerEndpointRateLimiter) getLimiter(endpoint string) *RateLimiter {
	pe.mu.RLock()
	if limiter, ok := pe.limiters[endpoint]; ok {
		pe.mu.RUnlock()
		return limiter
	}
	pe.mu.RUnlock()

	// Create default limiter for endpoint
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if limiter, ok := pe.limiters[endpoint]; ok {
		return limiter
	}

	limiter := NewRateLimiter(pe.defaults)
	pe.limiters[endpoint] = limiter
	return limiter
}

// Middleware returns the rate limiting middleware.
func (pe *PerEndpointRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := pe.getLimiter(r.URL.Path)
		limiter.Middleware(next).ServeHTTP(w, r)
	})
}

// Close stops all rate limiters.
func (pe *PerEndpointRateLimiter) Close() {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	for _, limiter := range pe.limiters {
		limiter.Close()
	}
}
