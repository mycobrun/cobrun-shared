// Package http provides HTTP utilities including a resilient HTTP client.
package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewResilientClient(t *testing.T) {
	config := DefaultResilientClientConfig("test-service", "http://localhost:8080")
	client := NewResilientClient(config)

	if client == nil {
		t.Fatal("expected client to be created")
	}

	if client.circuitBreaker == nil {
		t.Error("expected circuit breaker to be initialized")
	}

	if client.httpClient == nil {
		t.Error("expected HTTP client to be initialized")
	}
}

func TestDefaultResilientClientConfig(t *testing.T) {
	config := DefaultResilientClientConfig("test-service", "http://localhost:8080")

	if config.BaseURL != "http://localhost:8080" {
		t.Errorf("expected BaseURL http://localhost:8080, got %s", config.BaseURL)
	}

	if config.Timeout <= 0 {
		t.Error("expected positive timeout")
	}

	if config.RetryConfig.MaxRetries <= 0 {
		t.Error("expected positive max retries")
	}

	if config.CircuitBreakerConfig.Name != "test-service" {
		t.Errorf("expected service name test-service, got %s", config.CircuitBreakerConfig.Name)
	}
}

func TestResilientClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx := context.Background()
	resp, err := client.Get(ctx, "/test", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestResilientClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx := context.Background()
	body := map[string]string{"name": "test"}
	resp, err := client.Post(ctx, "/test", body, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestResilientClient_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx := context.Background()
	body := map[string]string{"name": "updated"}
	resp, err := client.Put(ctx, "/test/123", body, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestResilientClient_Patch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx := context.Background()
	body := map[string]string{"status": "active"}
	resp, err := client.Patch(ctx, "/test/123", body, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestResilientClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx := context.Background()
	resp, err := client.Delete(ctx, "/test/123", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestResilientClient_GetJSON(t *testing.T) {
	type TestData struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TestData{ID: 123, Name: "test"})
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx := context.Background()
	var result TestData
	err := client.GetJSON(ctx, "/test", &result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 123 {
		t.Errorf("expected ID 123, got %d", result.ID)
	}

	if result.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", result.Name)
	}
}

func TestResilientClient_PostJSON(t *testing.T) {
	type RequestData struct {
		Name string `json:"name"`
	}

	type ResponseData struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RequestData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ResponseData{ID: 456, Name: req.Name})
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx := context.Background()
	body := RequestData{Name: "test"}
	var result ResponseData
	err := client.PostJSON(ctx, "/test", body, &result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 456 {
		t.Errorf("expected ID 456, got %d", result.ID)
	}

	if result.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", result.Name)
	}
}

func TestResilientClient_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	config.RetryConfig.MaxRetries = 3
	config.RetryConfig.InitialDelay = 10 * time.Millisecond
	client := NewResilientClient(config)

	ctx := context.Background()
	resp, err := client.Get(ctx, "/test", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestResilientClient_RetryExhausted(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	config.RetryConfig.MaxRetries = 2
	config.RetryConfig.InitialDelay = 10 * time.Millisecond
	client := NewResilientClient(config)

	ctx := context.Background()
	_, err := client.Get(ctx, "/test", nil)

	if err == nil {
		t.Error("expected error after retries exhausted")
	}

	// Should attempt initial + 2 retries = 3 total
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestResilientClient_CircuitBreaker(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	config.CircuitBreakerConfig.FailureThreshold = 2
	config.RetryConfig.MaxRetries = 0 // Disable retries for this test
	client := NewResilientClient(config)

	ctx := context.Background()

	// First two requests should fail and open the circuit
	for i := 0; i < 2; i++ {
		_, _ = client.Get(ctx, "/test", nil)
	}

	// Circuit should now be open
	if client.circuitBreaker.State() != StateOpen {
		t.Error("expected circuit breaker to be open")
	}

	// Next request should be blocked by circuit breaker
	_, err := client.Get(ctx, "/test", nil)
	if err == nil {
		t.Error("expected error from open circuit breaker")
	}
}

func TestResilientClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Get(ctx, "/test", nil)

	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestResilientClient_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Error("expected custom header to be set")
		}
		if r.Header.Get("Authorization") != "Bearer token123" {
			t.Error("expected authorization header to be set")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx := context.Background()
	headers := map[string]string{
		"X-Custom-Header": "custom-value",
		"Authorization":   "Bearer token123",
	}

	_, err := client.Get(ctx, "/test", headers)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResilientClient_4xxErrors(t *testing.T) {
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

			config := DefaultResilientClientConfig("test", server.URL)
			config.RetryConfig.MaxRetries = 0 // Don't retry 4xx errors
			client := NewResilientClient(config)

			ctx := context.Background()
			_, err := client.Get(ctx, "/test", nil)

			if err == nil {
				t.Error("expected error for 4xx status code")
			}

			httpErr, ok := err.(*HTTPError)
			if !ok {
				t.Errorf("expected HTTPError, got %T", err)
			} else if httpErr.StatusCode != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, httpErr.StatusCode)
			}
		})
	}
}

func TestResilientClient_CircuitState(t *testing.T) {
	config := DefaultResilientClientConfig("test", "http://localhost:8080")
	client := NewResilientClient(config)

	state := client.CircuitState()
	if state != StateClosed {
		t.Errorf("expected initial state to be closed, got %s", state.String())
	}
}

func TestResilientClient_Metrics(t *testing.T) {
	config := DefaultResilientClientConfig("test-metrics", "http://localhost:8080")
	client := NewResilientClient(config)

	metrics := client.Metrics()

	if metrics.Name != "test-metrics" {
		t.Errorf("expected name 'test-metrics', got '%s'", metrics.Name)
	}

	if metrics.State != StateClosed {
		t.Errorf("expected state closed, got %v", metrics.State)
	}
}

func TestHTTPError_Error(t *testing.T) {
	err := &HTTPError{
		StatusCode: 404,
		Body:       []byte("not found"),
	}

	expected := "HTTP 404: not found"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestResilientClient_CalculateDelay(t *testing.T) {
	config := DefaultResilientClientConfig("test", "http://localhost:8080")
	config.RetryConfig.InitialDelay = 100 * time.Millisecond
	config.RetryConfig.MaxDelay = 2 * time.Second
	client := NewResilientClient(config)

	tests := []struct {
		attempt     int
		expectDelay time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1600 * time.Millisecond},
		{5, 2 * time.Second}, // Should be capped at MaxDelay
		{10, 2 * time.Second}, // Should be capped at MaxDelay
	}

	for _, tt := range tests {
		delay := client.calculateDelay(tt.attempt)
		if delay != tt.expectDelay {
			t.Errorf("attempt %d: expected delay %v, got %v", tt.attempt, tt.expectDelay, delay)
		}
	}
}

func TestResilientClient_IsRetryable(t *testing.T) {
	config := DefaultResilientClientConfig("test", "http://localhost:8080")
	client := NewResilientClient(config)

	tests := []struct {
		name       string
		statusCode int
		err        error
		resp       *ClientResponse
		expect     bool
	}{
		{
			name:       "network error is retryable",
			err:        context.DeadlineExceeded,
			resp:       nil,
			expect:     true,
		},
		{
			name:       "500 error is retryable",
			statusCode: 500,
			err:        &HTTPError{StatusCode: 500},
			resp:       &ClientResponse{StatusCode: 500},
			expect:     true,
		},
		{
			name:       "503 error is retryable",
			statusCode: 503,
			err:        &HTTPError{StatusCode: 503},
			resp:       &ClientResponse{StatusCode: 503},
			expect:     true,
		},
		{
			name:       "429 error is retryable",
			statusCode: 429,
			err:        &HTTPError{StatusCode: 429},
			resp:       &ClientResponse{StatusCode: 429},
			expect:     true,
		},
		{
			name:       "404 error is not retryable",
			statusCode: 404,
			err:        &HTTPError{StatusCode: 404},
			resp:       &ClientResponse{StatusCode: 404},
			expect:     false,
		},
		{
			name:   "no error is not retryable",
			err:    nil,
			resp:   &ClientResponse{StatusCode: 200},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isRetryable(tt.err, tt.resp)
			if result != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, result)
			}
		})
	}
}

func TestResilientClient_RequestWithBody(t *testing.T) {
	receivedBody := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		receivedBody = body["message"]
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultResilientClientConfig("test", server.URL)
	client := NewResilientClient(config)

	ctx := context.Background()
	body := map[string]string{"message": "hello world"}
	_, err := client.Post(ctx, "/test", body, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody != "hello world" {
		t.Errorf("expected body 'hello world', got '%s'", receivedBody)
	}
}
