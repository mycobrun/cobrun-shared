// Package http provides HTTP utilities including a resilient HTTP client.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ResilientClientConfig holds configuration for the resilient HTTP client.
type ResilientClientConfig struct {
	// BaseURL is the base URL for all requests.
	BaseURL string
	// Timeout is the request timeout.
	Timeout time.Duration
	// CircuitBreakerConfig configures the circuit breaker.
	CircuitBreakerConfig CircuitBreakerConfig
	// RetryConfig configures retry behavior.
	RetryConfig RetryConfig
}

// RetryConfig configures retry behavior for the HTTP client.
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// DefaultResilientClientConfig returns sensible production defaults.
func DefaultResilientClientConfig(serviceName, baseURL string) ResilientClientConfig {
	return ResilientClientConfig{
		BaseURL: baseURL,
		Timeout: 30 * time.Second,
		CircuitBreakerConfig: CircuitBreakerConfig{
			Name:                    serviceName,
			FailureThreshold:        5,
			SuccessThreshold:        2,
			Timeout:                 30 * time.Second,
			MaxConcurrentInHalfOpen: 1,
		},
		RetryConfig: RetryConfig{
			MaxRetries:   3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     2 * time.Second,
		},
	}
}

// ResilientClient is an HTTP client with circuit breaker and retry capabilities.
type ResilientClient struct {
	config         ResilientClientConfig
	httpClient     *http.Client
	circuitBreaker *CircuitBreaker
	tracer         trace.Tracer
}

// NewResilientClient creates a new resilient HTTP client.
func NewResilientClient(config ResilientClientConfig) *ResilientClient {
	return &ResilientClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		circuitBreaker: NewCircuitBreaker(config.CircuitBreakerConfig),
		tracer:         otel.Tracer(config.CircuitBreakerConfig.Name),
	}
}

// Request represents an HTTP request to be made.
type Request struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// HTTPResponse represents an HTTP response from the resilient client.
type HTTPResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do executes an HTTP request with circuit breaker and retry protection.
func (c *ResilientClient) Do(ctx context.Context, req Request) (*HTTPResponse, error) {
	var response *HTTPResponse
	var lastErr error

	err := c.circuitBreaker.Execute(ctx, func() error {
		response, lastErr = c.doWithRetry(ctx, req)
		return lastErr
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// doWithRetry executes a request with retry logic.
func (c *ResilientClient) doWithRetry(ctx context.Context, req Request) (*HTTPResponse, error) {
	var response *HTTPResponse
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryConfig.MaxRetries; attempt++ {
		response, lastErr = c.doRequest(ctx, req)
		if lastErr == nil {
			return response, nil
		}

		// Check if error is retryable
		if !c.isRetryable(lastErr, response) {
			return response, lastErr
		}

		// Don't sleep after last attempt
		if attempt == c.config.RetryConfig.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay := c.calculateDelay(attempt)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return response, lastErr
}

// doRequest executes a single HTTP request with tracing.
func (c *ResilientClient) doRequest(ctx context.Context, req Request) (*HTTPResponse, error) {
	url := c.config.BaseURL + req.Path

	// Start span for the HTTP request
	spanName := fmt.Sprintf("HTTP %s %s", req.Method, req.Path)
	ctx, span := c.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", url),
			attribute.String("service.name", c.config.CircuitBreakerConfig.Name),
		),
	)
	defer span.End()

	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to marshal request body")
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Inject trace context into outgoing request headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(httpReq.Header))

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "request failed")
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read response body")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Record response status
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	response := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}

	// Return error for non-2xx status codes
	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       body,
		}
	}

	span.SetStatus(codes.Ok, "")
	return response, nil
}

// isRetryable determines if a request should be retried.
func (c *ResilientClient) isRetryable(err error, resp *HTTPResponse) bool {
	if err == nil {
		return false
	}

	// Network errors are retryable
	if resp == nil {
		return true
	}

	// Retry on server errors and rate limiting
	switch resp.StatusCode {
	case 408, 429, 500, 502, 503, 504:
		return true
	}

	return false
}

// calculateDelay calculates retry delay with exponential backoff.
func (c *ResilientClient) calculateDelay(attempt int) time.Duration {
	delay := c.config.RetryConfig.InitialDelay * (1 << attempt)
	if delay > c.config.RetryConfig.MaxDelay {
		delay = c.config.RetryConfig.MaxDelay
	}
	return delay
}

// HTTPError represents an HTTP error response.
type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, string(e.Body))
}

// Get performs a GET request.
func (c *ResilientClient) Get(ctx context.Context, path string, headers map[string]string) (*HTTPResponse, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodGet,
		Path:    path,
		Headers: headers,
	})
}

// Post performs a POST request.
func (c *ResilientClient) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*HTTPResponse, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodPost,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// Put performs a PUT request.
func (c *ResilientClient) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*HTTPResponse, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodPut,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// Patch performs a PATCH request.
func (c *ResilientClient) Patch(ctx context.Context, path string, body interface{}, headers map[string]string) (*HTTPResponse, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodPatch,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// Delete performs a DELETE request.
func (c *ResilientClient) Delete(ctx context.Context, path string, headers map[string]string) (*HTTPResponse, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodDelete,
		Path:    path,
		Headers: headers,
	})
}

// GetJSON performs a GET request and unmarshals the response.
func (c *ResilientClient) GetJSON(ctx context.Context, path string, result interface{}) error {
	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp.Body, result)
}

// PostJSON performs a POST request and unmarshals the response.
func (c *ResilientClient) PostJSON(ctx context.Context, path string, body, result interface{}) error {
	resp, err := c.Post(ctx, path, body, nil)
	if err != nil {
		return err
	}
	if result != nil && len(resp.Body) > 0 {
		return json.Unmarshal(resp.Body, result)
	}
	return nil
}

// CircuitState returns the current state of the circuit breaker.
func (c *ResilientClient) CircuitState() CircuitState {
	return c.circuitBreaker.State()
}

// Metrics returns the circuit breaker metrics.
func (c *ResilientClient) Metrics() CircuitBreakerMetrics {
	return c.circuitBreaker.Metrics()
}
