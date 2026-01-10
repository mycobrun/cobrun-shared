package resilience

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ResilientHTTPClient wraps an HTTP client with circuit breaker protection.
type ResilientHTTPClient struct {
	client         *http.Client
	circuitBreaker *CircuitBreaker
	retries        int
	retryDelay     time.Duration
}

// ResilientHTTPClientConfig configures a resilient HTTP client.
type ResilientHTTPClientConfig struct {
	// Name for the circuit breaker.
	Name string

	// Timeout for HTTP requests.
	Timeout time.Duration

	// Retries is the number of retry attempts.
	Retries int

	// RetryDelay is the delay between retries.
	RetryDelay time.Duration

	// CircuitBreaker config (optional, uses defaults if nil).
	CircuitBreakerConfig *CircuitBreakerConfig
}

// DefaultResilientHTTPClientConfig returns sensible defaults.
func DefaultResilientHTTPClientConfig(name string) ResilientHTTPClientConfig {
	return ResilientHTTPClientConfig{
		Name:       name,
		Timeout:    30 * time.Second,
		Retries:    3,
		RetryDelay: 100 * time.Millisecond,
	}
}

// NewResilientHTTPClient creates a new resilient HTTP client.
func NewResilientHTTPClient(config ResilientHTTPClientConfig) *ResilientHTTPClient {
	var cbConfig CircuitBreakerConfig
	if config.CircuitBreakerConfig != nil {
		cbConfig = *config.CircuitBreakerConfig
	} else {
		cbConfig = DefaultCircuitBreakerConfig(config.Name)
	}

	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	return &ResilientHTTPClient{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		circuitBreaker: NewCircuitBreaker(cbConfig),
		retries:        config.Retries,
		retryDelay:     config.RetryDelay,
	}
}

// Do executes an HTTP request with circuit breaker and retry protection.
func (c *ResilientHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt)) // Exponential backoff
		}

		err := c.circuitBreaker.ExecuteWithContext(req.Context(), func(ctx context.Context) error {
			// Clone request for retry
			reqClone := req.Clone(ctx)

			var reqErr error
			resp, reqErr = c.client.Do(reqClone)
			if reqErr != nil {
				return reqErr
			}

			// Consider 5xx errors as failures for circuit breaker
			if resp.StatusCode >= 500 {
				// Drain and close body to allow connection reuse
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				return fmt.Errorf("server error: status %d", resp.StatusCode)
			}

			return nil
		})

		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on circuit open
		if err == ErrCircuitOpen {
			return nil, err
		}

		// Don't retry on context cancellation
		if err == context.Canceled || err == context.DeadlineExceeded {
			return nil, err
		}
	}

	return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

// Get performs an HTTP GET request.
func (c *ResilientHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post performs an HTTP POST request.
func (c *ResilientHTTPClient) Post(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// CircuitBreaker returns the underlying circuit breaker.
func (c *ResilientHTTPClient) CircuitBreaker() *CircuitBreaker {
	return c.circuitBreaker
}

// Metrics returns circuit breaker metrics.
func (c *ResilientHTTPClient) Metrics() CircuitBreakerMetrics {
	return c.circuitBreaker.Metrics()
}
