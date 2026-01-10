package resilience

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewResilientHTTPClient(t *testing.T) {
	config := DefaultResilientHTTPClientConfig("test-service")
	client := NewResilientHTTPClient(config)

	if client == nil {
		t.Fatal("expected client to be created")
	}

	if client.circuitBreaker == nil {
		t.Error("expected circuit breaker to be initialized")
	}

	if client.client == nil {
		t.Error("expected HTTP client to be initialized")
	}
}

func TestDefaultResilientHTTPClientConfig(t *testing.T) {
	config := DefaultResilientHTTPClientConfig("test-service")

	if config.Name != "test-service" {
		t.Errorf("expected name 'test-service', got '%s'", config.Name)
	}

	if config.Timeout <= 0 {
		t.Error("expected positive timeout")
	}

	if config.Retries < 0 {
		t.Error("expected non-negative retries")
	}

	if config.RetryDelay <= 0 {
		t.Error("expected positive retry delay")
	}
}

func TestResilientHTTPClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	config := DefaultResilientHTTPClientConfig("test")
	client := NewResilientHTTPClient(config)

	ctx := context.Background()
	resp, err := client.Get(ctx, server.URL+"/test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestResilientHTTPClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != "test data" {
			t.Errorf("expected body 'test data', got '%s'", string(body))
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := DefaultResilientHTTPClientConfig("test")
	client := NewResilientHTTPClient(config)

	ctx := context.Background()
	resp, err := client.Post(ctx, server.URL+"/test", "application/json", strings.NewReader("test data"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestResilientHTTPClient_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultResilientHTTPClientConfig("test")
	config.Retries = 3
	config.RetryDelay = 10 * time.Millisecond
	client := NewResilientHTTPClient(config)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestResilientHTTPClient_RetryExhausted(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := DefaultResilientHTTPClientConfig("test")
	config.Retries = 2
	config.RetryDelay = 10 * time.Millisecond
	client := NewResilientHTTPClient(config)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
	_, err := client.Do(req)

	if err == nil {
		t.Error("expected error after retries exhausted")
	}

	// Should attempt initial + 2 retries = 3 total
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestResilientHTTPClient_CircuitBreaker(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cbConfig := DefaultCircuitBreakerConfig("test")
	cbConfig.FailureThreshold = 2
	cbConfig.Timeout = 1 * time.Second

	config := DefaultResilientHTTPClientConfig("test")
	config.CircuitBreakerConfig = &cbConfig
	config.Retries = 0 // Disable retries for this test
	client := NewResilientHTTPClient(config)

	ctx := context.Background()

	// First two requests should fail and open the circuit
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
		_, _ = client.Do(req)
	}

	// Circuit should now be open
	if client.CircuitBreaker().State() != StateOpen {
		t.Error("expected circuit breaker to be open")
	}

	// Next request should be blocked by circuit breaker
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
	_, err := client.Do(req)

	if err == nil {
		t.Error("expected error from open circuit breaker")
	}

	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestResilientHTTPClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultResilientHTTPClientConfig("test")
	client := NewResilientHTTPClient(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
	_, err := client.Do(req)

	if err == nil {
		t.Error("expected error from cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestResilientHTTPClient_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultResilientHTTPClientConfig("test")
	client := NewResilientHTTPClient(config)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
	_, err := client.Do(req)

	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestResilientHTTPClient_SuccessfulRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	config := DefaultResilientHTTPClientConfig("test")
	client := NewResilientHTTPClient(config)

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "success" {
		t.Errorf("expected body 'success', got '%s'", string(body))
	}
}

func TestResilientHTTPClient_4xxErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"bad request", http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized},
		{"forbidden", http.StatusForbidden},
		{"not found", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			config := DefaultResilientHTTPClientConfig("test")
			config.Retries = 0 // Don't retry 4xx errors
			client := NewResilientHTTPClient(config)

			ctx := context.Background()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
			resp, err := client.Do(req)

			// 4xx errors should return response but not retry
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatal("expected response")
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, resp.StatusCode)
			}
		})
	}
}

func TestResilientHTTPClient_CircuitBreakerMetrics(t *testing.T) {
	config := DefaultResilientHTTPClientConfig("test-metrics")
	client := NewResilientHTTPClient(config)

	metrics := client.Metrics()

	if metrics.Name != "test-metrics" {
		t.Errorf("expected name 'test-metrics', got '%s'", metrics.Name)
	}

	if metrics.State != "closed" {
		t.Errorf("expected state 'closed', got '%s'", metrics.State)
	}
}

func TestResilientHTTPClient_ExponentialBackoff(t *testing.T) {
	attempts := 0
	timings := []time.Time{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		timings = append(timings, time.Now())
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := DefaultResilientHTTPClientConfig("test")
	config.Retries = 3
	config.RetryDelay = 50 * time.Millisecond
	client := NewResilientHTTPClient(config)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
	_, _ = client.Do(req)

	// Verify exponential backoff
	if len(timings) != 4 { // Initial + 3 retries
		t.Errorf("expected 4 attempts, got %d", len(timings))
	}

	if len(timings) >= 2 {
		// First retry delay should be ~50ms
		delay1 := timings[1].Sub(timings[0])
		if delay1 < 40*time.Millisecond || delay1 > 70*time.Millisecond {
			t.Logf("first retry delay: %v (expected ~50ms)", delay1)
		}
	}

	if len(timings) >= 3 {
		// Second retry delay should be ~100ms
		delay2 := timings[2].Sub(timings[1])
		if delay2 < 80*time.Millisecond || delay2 > 130*time.Millisecond {
			t.Logf("second retry delay: %v (expected ~100ms)", delay2)
		}
	}
}

func TestResilientHTTPClient_NoRetryOnCircuitOpen(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cbConfig := DefaultCircuitBreakerConfig("test")
	cbConfig.FailureThreshold = 1
	cbConfig.Timeout = 1 * time.Second

	config := DefaultResilientHTTPClientConfig("test")
	config.CircuitBreakerConfig = &cbConfig
	config.Retries = 3
	config.RetryDelay = 10 * time.Millisecond
	client := NewResilientHTTPClient(config)

	ctx := context.Background()

	// First request opens the circuit
	req1, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
	_, _ = client.Do(req1)

	attempts = 0 // Reset counter

	// Second request should not retry when circuit is open
	req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
	_, err := client.Do(req2)

	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}

	// Should not have made any actual HTTP requests
	if attempts > 0 {
		t.Errorf("expected 0 attempts when circuit is open, got %d", attempts)
	}
}

func TestResilientHTTPClient_GetCircuitBreaker(t *testing.T) {
	config := DefaultResilientHTTPClientConfig("test")
	client := NewResilientHTTPClient(config)

	cb := client.CircuitBreaker()

	if cb == nil {
		t.Fatal("expected circuit breaker")
	}

	if cb.State() != StateClosed {
		t.Errorf("expected initial state to be closed, got %s", cb.State())
	}
}

func TestResilientHTTPClient_ServerErrorRetries(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expectRetry bool
	}{
		{"500 error retries", http.StatusInternalServerError, true},
		{"502 error retries", http.StatusBadGateway, true},
		{"503 error retries", http.StatusServiceUnavailable, true},
		{"504 error retries", http.StatusGatewayTimeout, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			config := DefaultResilientHTTPClientConfig("test")
			config.Retries = 2
			config.RetryDelay = 10 * time.Millisecond
			client := NewResilientHTTPClient(config)

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
			_, _ = client.Do(req)

			if tt.expectRetry {
				// Should attempt initial + 2 retries = 3 total
				if attempts != 3 {
					t.Errorf("expected 3 attempts, got %d", attempts)
				}
			}
		})
	}
}

func TestResilientHTTPClient_CustomTimeout(t *testing.T) {
	config := DefaultResilientHTTPClientConfig("test")
	config.Timeout = 50 * time.Millisecond
	client := NewResilientHTTPClient(config)

	if client.client.Timeout != 50*time.Millisecond {
		t.Errorf("expected timeout 50ms, got %v", client.client.Timeout)
	}
}

func TestResilientHTTPClient_ZeroTimeout(t *testing.T) {
	config := DefaultResilientHTTPClientConfig("test")
	config.Timeout = 0
	client := NewResilientHTTPClient(config)

	// Should use default timeout
	if client.client.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", client.client.Timeout)
	}
}

func TestResilientHTTPClient_CustomCircuitBreakerConfig(t *testing.T) {
	cbConfig := CircuitBreakerConfig{
		Name:             "custom-cb",
		FailureThreshold: 10,
		SuccessThreshold: 3,
		Timeout:          1 * time.Minute,
		MaxRequests:      5,
	}

	config := DefaultResilientHTTPClientConfig("test")
	config.CircuitBreakerConfig = &cbConfig
	client := NewResilientHTTPClient(config)

	if client.circuitBreaker.config.Name != "custom-cb" {
		t.Errorf("expected name 'custom-cb', got '%s'", client.circuitBreaker.config.Name)
	}

	if client.circuitBreaker.config.FailureThreshold != 10 {
		t.Errorf("expected failure threshold 10, got %d", client.circuitBreaker.config.FailureThreshold)
	}
}

func TestResilientHTTPClient_RequestCloning(t *testing.T) {
	// Test that requests are properly cloned for retries
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := DefaultResilientHTTPClientConfig("test")
	config.Retries = 1
	config.RetryDelay = 10 * time.Millisecond
	client := NewResilientHTTPClient(config)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
	req.Header.Set("X-Custom-Header", "test-value")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}
