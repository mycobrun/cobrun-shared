// Package database provides database client utilities.
package database

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (0 means no retries).
	MaxRetries int
	// InitialDelay is the initial delay before the first retry.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
	// Multiplier is the factor by which delay increases after each retry.
	Multiplier float64
	// Jitter is the maximum random jitter to add (as a percentage of delay, 0-1).
	Jitter float64
}

// DefaultRetryConfig returns sensible production defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.2, // 20% jitter
	}
}

// RetryableFunc is a function that can be retried.
type RetryableFunc func() error

// Retry executes a function with exponential backoff retry.
func Retry(ctx context.Context, config RetryConfig, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if the error is retryable
		if !isRetryable(err) {
			return err
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(config, attempt)

		// Wait or check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", config.MaxRetries, lastErr)
}

// calculateDelay calculates the delay for a given attempt with jitter.
func calculateDelay(config RetryConfig, attempt int) time.Duration {
	// Exponential backoff: initialDelay * multiplier^attempt
	delay := float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter (random percentage of the delay)
	if config.Jitter > 0 {
		jitterAmount := delay * config.Jitter * rand.Float64()
		// Randomly add or subtract jitter
		if rand.Float64() < 0.5 {
			delay -= jitterAmount
		} else {
			delay += jitterAmount
		}
	}

	return time.Duration(delay)
}

// isRetryable determines if an error is retryable.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for context errors (not retryable)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for Azure SDK errors
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		// Retry on server errors and rate limiting
		switch respErr.StatusCode {
		case 408, 429, 500, 502, 503, 504:
			return true
		case 400, 401, 403, 404, 409:
			// Client errors are not retryable
			return false
		}
	}

	// Check for common retryable error patterns
	errStr := err.Error()
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"connection timed out",
		"timeout",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"i/o timeout",
		"no such host",
		"network is unreachable",
	}

	for _, pattern := range retryablePatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	// Check for wrapped retryable errors
	var unwrapped error = err
	for unwrapped != nil {
		if isNetworkError(unwrapped) {
			return true
		}
		unwrapped = errors.Unwrap(unwrapped)
	}

	// Default to retryable for unknown errors (safer for transient issues)
	return true
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))

	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		sLower[i] = c
	}

	for i := range substr {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		substrLower[i] = c
	}

	return indexOf(string(sLower), string(substrLower)) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// isNetworkError checks if the error is a network-related error.
func isNetworkError(err error) bool {
	// Check for common network error types
	errType := fmt.Sprintf("%T", err)
	networkTypes := []string{
		"net.OpError",
		"net.DNSError",
		"syscall.Errno",
		"*url.Error",
	}

	for _, t := range networkTypes {
		if errType == t {
			return true
		}
	}

	return false
}

// RetryWithResult executes a function that returns a value with retry.
func RetryWithResult[T any](ctx context.Context, config RetryConfig, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		var err error
		result, err = fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !isRetryable(err) {
			return result, err
		}

		if attempt == config.MaxRetries {
			break
		}

		delay := calculateDelay(config, attempt)

		select {
		case <-ctx.Done():
			return result, fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return result, fmt.Errorf("max retries (%d) exceeded: %w", config.MaxRetries, lastErr)
}

// RetryCosmosOperation wraps a Cosmos DB operation with retry logic.
func RetryCosmosOperation(ctx context.Context, fn RetryableFunc) error {
	return Retry(ctx, DefaultRetryConfig(), fn)
}

// RetrySQLOperation wraps a SQL operation with retry logic.
func RetrySQLOperation(ctx context.Context, fn RetryableFunc) error {
	config := DefaultRetryConfig()
	// SQL operations may need longer delays due to connection pooling
	config.InitialDelay = 200 * time.Millisecond
	return Retry(ctx, config, fn)
}

// RetryRedisOperation wraps a Redis operation with retry logic.
func RetryRedisOperation(ctx context.Context, fn RetryableFunc) error {
	config := DefaultRetryConfig()
	// Redis is typically faster, so shorter delays
	config.InitialDelay = 50 * time.Millisecond
	config.MaxDelay = 2 * time.Second
	return Retry(ctx, config, fn)
}
