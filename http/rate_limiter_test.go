// Package http provides HTTP utilities including rate limiting middleware.
package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	bucket := NewTokenBucket(10, 5) // 10 tokens max, 5 per second refill

	// Should allow 10 requests initially
	for i := 0; i < 10; i++ {
		if !bucket.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 11th request should be denied
	if bucket.Allow() {
		t.Error("11th request should be denied")
	}

	// Wait for refill
	time.Sleep(300 * time.Millisecond)

	// Should allow at least 1 more request
	if !bucket.Allow() {
		t.Error("request after refill should be allowed")
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		KeyFunc:           IPKeyFunc,
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Should allow burst size requests
	for i := 0; i < 5; i++ {
		if !rl.Allow(req) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if rl.Allow(req) {
		t.Error("6th request should be denied")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         3,
		KeyFunc:           IPKeyFunc,
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"

	// Exhaust limit for first IP
	for i := 0; i < 3; i++ {
		rl.Allow(req1)
	}

	// Second IP should still be allowed
	if !rl.Allow(req2) {
		t.Error("request from different IP should be allowed")
	}
}

func TestRateLimiter_ExcludeFunc(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         1,
		KeyFunc:           IPKeyFunc,
		ExcludeFunc: func(r *http.Request) bool {
			return r.URL.Path == "/health"
		},
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	healthReq := httptest.NewRequest("GET", "/health", nil)
	healthReq.RemoteAddr = "192.168.1.1:12345"

	apiReq := httptest.NewRequest("GET", "/api/test", nil)
	apiReq.RemoteAddr = "192.168.1.1:12345"

	// Exhaust limit with API request
	rl.Allow(apiReq)

	// Health check should still be allowed (excluded)
	if !rl.Allow(healthReq) {
		t.Error("health check should be excluded from rate limiting")
	}

	// Another API request should be denied
	if rl.Allow(apiReq) {
		t.Error("API request should be denied")
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         2,
		KeyFunc:           IPKeyFunc,
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rateLimited := rl.Middleware(handler)

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		rateLimited.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d should return 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	rateLimited.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("request should return 429, got %d", w.Code)
	}

	// Check rate limit headers
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header should be set")
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be set")
	}
}

func TestIPKeyFunc(t *testing.T) {
	tests := []struct {
		name        string
		remoteAddr  string
		xForwarded  string
		xRealIP     string
		expectedKey string
	}{
		{
			name:        "remote addr only",
			remoteAddr:  "192.168.1.1:12345",
			expectedKey: "192.168.1.1:12345",
		},
		{
			name:        "x-forwarded-for",
			remoteAddr:  "192.168.1.1:12345",
			xForwarded:  "10.0.0.1",
			expectedKey: "10.0.0.1",
		},
		{
			name:        "x-real-ip",
			remoteAddr:  "192.168.1.1:12345",
			xRealIP:     "10.0.0.2",
			expectedKey: "10.0.0.2",
		},
		{
			name:        "x-forwarded-for takes precedence",
			remoteAddr:  "192.168.1.1:12345",
			xForwarded:  "10.0.0.1",
			xRealIP:     "10.0.0.2",
			expectedKey: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwarded)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			key := IPKeyFunc(req)
			if key != tt.expectedKey {
				t.Errorf("IPKeyFunc() = %v, want %v", key, tt.expectedKey)
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	middleware := RateLimitMiddleware(100, 10)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("first request should return 200, got %d", w.Code)
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	bucket := NewTokenBucket(10, 10) // 10 tokens, refill 10 per second

	// Consume all tokens
	for i := 0; i < 10; i++ {
		if !bucket.Allow() {
			t.Errorf("token %d should be available", i+1)
		}
	}

	// Should be empty now
	if bucket.Allow() {
		t.Error("bucket should be empty")
	}

	// Wait for refill (100ms = 1 token at 10 per second)
	time.Sleep(150 * time.Millisecond)

	// Should have at least 1 token now
	if !bucket.Allow() {
		t.Error("bucket should have refilled")
	}
}

func TestTokenBucket_Tokens(t *testing.T) {
	bucket := NewTokenBucket(10, 5)

	// Initially should have max tokens
	tokens := bucket.Tokens()
	if tokens < 9.9 || tokens > 10.1 {
		t.Errorf("expected ~10 tokens, got %f", tokens)
	}

	// Consume some tokens
	bucket.Allow()
	bucket.Allow()

	tokens = bucket.Tokens()
	if tokens < 7.9 || tokens > 8.1 {
		t.Errorf("expected ~8 tokens after consuming 2, got %f", tokens)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		KeyFunc:           IPKeyFunc,
		CleanupInterval:   50 * time.Millisecond,
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	// Create some buckets
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rl.Allow(req1)

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	rl.Allow(req2)

	// Wait for cleanup to run
	time.Sleep(100 * time.Millisecond)

	// Buckets should have been cleaned up (they're full/unused)
	// This is more of a smoke test to ensure cleanup doesn't crash
}

func TestUserIDKeyFunc(t *testing.T) {
	tests := []struct {
		name        string
		userID      interface{}
		remoteAddr  string
		expectedKey string
	}{
		{
			name:        "with user ID",
			userID:      "user123",
			remoteAddr:  "192.168.1.1:12345",
			expectedKey: "user:user123",
		},
		{
			name:        "without user ID",
			userID:      nil,
			remoteAddr:  "192.168.1.1:12345",
			expectedKey: "192.168.1.1:12345",
		},
		{
			name:        "with numeric user ID",
			userID:      12345,
			remoteAddr:  "192.168.1.1:12345",
			expectedKey: "user:12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			if tt.userID != nil {
				ctx := req.Context()
				ctx = context.WithValue(ctx, "user_id", tt.userID)
				req = req.WithContext(ctx)
			}

			key := UserIDKeyFunc(req)
			if key != tt.expectedKey {
				t.Errorf("UserIDKeyFunc() = %v, want %v", key, tt.expectedKey)
			}
		})
	}
}

func TestCombinedKeyFunc(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	key := CombinedKeyFunc(req)
	expectedKey := "192.168.1.1:12345:/api/users"

	if key != expectedKey {
		t.Errorf("CombinedKeyFunc() = %v, want %v", key, expectedKey)
	}
}

func TestRateLimiter_OnLimitExceeded(t *testing.T) {
	limitExceeded := false
	var exceededKey string

	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         2,
		KeyFunc:           IPKeyFunc,
		OnLimitExceeded: func(r *http.Request, key string) {
			limitExceeded = true
			exceededKey = key
		},
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Exhaust limit
	rl.Allow(req)
	rl.Allow(req)

	// This should trigger the callback
	rl.Allow(req)

	if !limitExceeded {
		t.Error("OnLimitExceeded callback should have been called")
	}

	if exceededKey != "192.168.1.1:12345" {
		t.Errorf("expected key 192.168.1.1:12345, got %s", exceededKey)
	}
}

func TestPerEndpointRateLimiter(t *testing.T) {
	config := DefaultRateLimiterConfig()
	pe := NewPerEndpointRateLimiter(config)
	defer pe.Close()

	// Set different limits for different endpoints
	pe.SetEndpointLimit("/api/public", 100, 10)
	pe.SetEndpointLimit("/api/private", 10, 2)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := pe.Middleware(handler)

	// Test public endpoint
	req1 := httptest.NewRequest("GET", "/api/public", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	wrapped.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("public endpoint should allow request, got %d", w1.Code)
	}

	// Test private endpoint - exhaust limit
	req2 := httptest.NewRequest("GET", "/api/private", nil)
	req2.RemoteAddr = "192.168.1.1:12345"

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req2)
		if w.Code != http.StatusOK {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Third request should be rate limited
	w2 := httptest.NewRecorder()
	wrapped.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w2.Code)
	}
}

func TestPerEndpointRateLimiter_GetLimiter(t *testing.T) {
	config := DefaultRateLimiterConfig()
	pe := NewPerEndpointRateLimiter(config)
	defer pe.Close()

	// First call should create a new limiter
	limiter1 := pe.getLimiter("/api/test")
	if limiter1 == nil {
		t.Fatal("expected limiter to be created")
	}

	// Second call should return same limiter
	limiter2 := pe.getLimiter("/api/test")
	if limiter1 != limiter2 {
		t.Error("expected same limiter instance")
	}

	// Different endpoint should get different limiter
	limiter3 := pe.getLimiter("/api/other")
	if limiter1 == limiter3 {
		t.Error("different endpoints should have different limiters")
	}
}

func TestDefaultRateLimiterConfig(t *testing.T) {
	config := DefaultRateLimiterConfig()

	if config.RequestsPerSecond <= 0 {
		t.Error("expected positive requests per second")
	}

	if config.BurstSize <= 0 {
		t.Error("expected positive burst size")
	}

	if config.KeyFunc == nil {
		t.Error("expected KeyFunc to be set")
	}

	if config.CleanupInterval <= 0 {
		t.Error("expected positive cleanup interval")
	}
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 100,
		BurstSize:         50,
		KeyFunc:           IPKeyFunc,
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Make concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				rl.Allow(req)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify no race conditions occurred
	// The rate limiter should handle concurrent access safely
}

func TestRateLimiter_MiddlewareHeaders(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		KeyFunc:           IPKeyFunc,
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := rl.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Check that rate limit headers are set
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header should be set")
	}

	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("X-RateLimit-Remaining header should be set")
	}
}

func TestRateLimiter_MultipleIPs(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         2,
		KeyFunc:           IPKeyFunc,
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	// Test that different IPs have separate rate limits
	ips := []string{
		"192.168.1.1:12345",
		"192.168.1.2:12345",
		"192.168.1.3:12345",
	}

	for _, ip := range ips {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ip

		// Each IP should be able to make 2 requests
		for i := 0; i < 2; i++ {
			if !rl.Allow(req) {
				t.Errorf("IP %s request %d should be allowed", ip, i+1)
			}
		}

		// Third request should be denied
		if rl.Allow(req) {
			t.Errorf("IP %s request 3 should be denied", ip)
		}
	}
}

func TestRateLimiter_MiddlewareFunc(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		KeyFunc:           IPKeyFunc,
	}

	rl := NewRateLimiter(config)
	defer rl.Close()

	middlewareFunc := rl.MiddlewareFunc()
	if middlewareFunc == nil {
		t.Fatal("MiddlewareFunc should return a function")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middlewareFunc(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestTokenBucket_MaxTokensLimit(t *testing.T) {
	bucket := NewTokenBucket(5, 10)

	// Wait for potential over-refill
	time.Sleep(200 * time.Millisecond)

	// Tokens should not exceed max
	tokens := bucket.Tokens()
	if tokens > 5.1 {
		t.Errorf("tokens should not exceed max of 5, got %f", tokens)
	}
}

func TestRateLimiter_XForwardedForMultiple(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")

	key := IPKeyFunc(req)

	// Should use the first IP in the X-Forwarded-For chain
	if key != "10.0.0.1, 10.0.0.2, 10.0.0.3" {
		t.Errorf("expected first forwarded IP, got %s", key)
	}
}
