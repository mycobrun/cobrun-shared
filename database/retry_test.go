package database

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", config.MaxRetries)
	}
	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("expected InitialDelay=100ms, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 5*time.Second {
		t.Errorf("expected MaxDelay=5s, got %v", config.MaxDelay)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("expected Multiplier=2.0, got %f", config.Multiplier)
	}
	if config.Jitter != 0.2 {
		t.Errorf("expected Jitter=0.2, got %f", config.Jitter)
	}
}

func TestRetry_Success(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return nil
	}

	err := Retry(ctx, config, fn)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.0, // No jitter for predictability
	}

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	err := Retry(ctx, config, fn)
	if err != nil {
		t.Errorf("expected no error after retries, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("persistent failure")
	}

	err := Retry(ctx, config, fn)
	if err == nil {
		t.Error("expected error after max retries")
	}
	if !errors.Is(err, errors.New("persistent failure")) {
		if err.Error() != "max retries (3) exceeded: persistent failure" {
			t.Errorf("unexpected error message: %v", err)
		}
	}
	if attempts != 4 { // Initial + 3 retries
		t.Errorf("expected 4 attempts, got %d", attempts)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := RetryConfig{
		MaxRetries:   5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		if attempts == 2 {
			cancel() // Cancel after second attempt
		}
		return errors.New("temporary failure")
	}

	err := Retry(ctx, config, fn)
	if err == nil {
		t.Error("expected error from context cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		if err.Error() != "retry cancelled: context canceled" {
			t.Errorf("expected context cancellation error, got %v", err)
		}
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return context.Canceled
	}

	err := Retry(ctx, config, fn)
	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (non-retryable), got %d", attempts)
	}
}

func TestCalculateDelay(t *testing.T) {
	tests := []struct {
		name    string
		config  RetryConfig
		attempt int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{
			name: "first retry",
			config: RetryConfig{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
				Jitter:       0.0,
			},
			attempt:  0,
			minDelay: 100 * time.Millisecond,
			maxDelay: 100 * time.Millisecond,
		},
		{
			name: "second retry",
			config: RetryConfig{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
				Jitter:       0.0,
			},
			attempt:  1,
			minDelay: 200 * time.Millisecond,
			maxDelay: 200 * time.Millisecond,
		},
		{
			name: "capped at max delay",
			config: RetryConfig{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     500 * time.Millisecond,
				Multiplier:   2.0,
				Jitter:       0.0,
			},
			attempt:  10,
			minDelay: 500 * time.Millisecond,
			maxDelay: 500 * time.Millisecond,
		},
		{
			name: "with jitter",
			config: RetryConfig{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
				Jitter:       0.2,
			},
			attempt:  0,
			minDelay: 80 * time.Millisecond,  // 100ms - 20% jitter
			maxDelay: 120 * time.Millisecond, // 100ms + 20% jitter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := calculateDelay(tt.config, tt.attempt)

			if tt.config.Jitter == 0 {
				if delay != tt.minDelay {
					t.Errorf("expected delay=%v, got %v", tt.minDelay, delay)
				}
			} else {
				if delay < tt.minDelay || delay > tt.maxDelay {
					t.Errorf("expected delay between %v and %v, got %v", tt.minDelay, tt.maxDelay, delay)
				}
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		retryable  bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "context canceled",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "context deadline exceeded",
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name:      "connection timeout",
			err:       errors.New("connection timed out"),
			retryable: true,
		},
		{
			name:      "connection refused",
			err:       errors.New("connection refused"),
			retryable: true,
		},
		{
			name:      "service unavailable",
			err:       errors.New("service unavailable"),
			retryable: true,
		},
		{
			name:      "too many requests",
			err:       errors.New("too many requests"),
			retryable: true,
		},
		{
			name:      "i/o timeout",
			err:       errors.New("i/o timeout"),
			retryable: true,
		},
		{
			name:      "temporary failure",
			err:       errors.New("temporary failure"),
			retryable: true,
		},
		{
			name:      "unknown error",
			err:       errors.New("unknown error"),
			retryable: true, // Default to retryable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryable(tt.err)
			if result != tt.retryable {
				t.Errorf("expected retryable=%v, got %v", tt.retryable, result)
			}
		})
	}
}

func TestIsRetryable_AzureErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		retryable  bool
	}{
		{name: "408 Request Timeout", statusCode: 408, retryable: true},
		{name: "429 Too Many Requests", statusCode: 429, retryable: true},
		{name: "500 Internal Server Error", statusCode: 500, retryable: true},
		{name: "502 Bad Gateway", statusCode: 502, retryable: true},
		{name: "503 Service Unavailable", statusCode: 503, retryable: true},
		{name: "504 Gateway Timeout", statusCode: 504, retryable: true},
		{name: "400 Bad Request", statusCode: 400, retryable: false},
		{name: "401 Unauthorized", statusCode: 401, retryable: false},
		{name: "403 Forbidden", statusCode: 403, retryable: false},
		{name: "404 Not Found", statusCode: 404, retryable: false},
		{name: "409 Conflict", statusCode: 409, retryable: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respErr := &azcore.ResponseError{
				StatusCode: tt.statusCode,
			}
			result := isRetryable(respErr)
			if result != tt.retryable {
				t.Errorf("for status %d, expected retryable=%v, got %v", tt.statusCode, tt.retryable, result)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"Connection Timed Out", "connection timed out", true},
		{"CONNECTION REFUSED", "connection refused", true},
		{"Service Unavailable", "service unavailable", true},
		{"Hello World", "world", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "bye", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s contains %s", tt.s, tt.substr), func(t *testing.T) {
			result := containsIgnoreCase(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestRetryWithResult_Success(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.0,
	}

	expected := "success"
	fn := func() (string, error) {
		return expected, nil
	}

	result, err := RetryWithResult(ctx, config, fn)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("expected result=%q, got %q", expected, result)
	}
}

func TestRetryWithResult_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.0,
	}

	attempts := 0
	expected := 42
	fn := func() (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("temporary failure")
		}
		return expected, nil
	}

	result, err := RetryWithResult(ctx, config, fn)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("expected result=%d, got %d", expected, result)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithResult_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.0,
	}

	fn := func() (string, error) {
		return "", errors.New("persistent failure")
	}

	result, err := RetryWithResult(ctx, config, fn)
	if err == nil {
		t.Error("expected error after max retries")
	}
	if result != "" {
		t.Errorf("expected empty result on error, got %q", result)
	}
}

func TestRetryCosmosOperation(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := RetryCosmosOperation(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetrySQLOperation(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := RetrySQLOperation(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryRedisOperation(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := RetryRedisOperation(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}
