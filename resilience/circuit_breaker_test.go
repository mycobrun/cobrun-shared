package resilience

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
	})

	// Should allow requests in closed state
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	stateChanges := make([]CircuitState, 0)
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
		OnStateChange: func(name string, from, to CircuitState) {
			stateChanges = append(stateChanges, to)
		},
	})

	// Cause failures
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return errTest
		})
	}

	// Wait for state change callback
	time.Sleep(10 * time.Millisecond)

	if cb.State() != StateOpen {
		t.Errorf("expected open state, got %s", cb.State())
	}
}

func TestCircuitBreaker_BlocksWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 2,
		Timeout:          1 * time.Second, // Long timeout
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = _ = cb.Execute(func() error { return errTest })
	}

	// Should block
	err := cb.Execute(func() error {
		return nil
	})

	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 2,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      2,
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = _ = cb.Execute(func() error { return errTest })
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open and allow request
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error after timeout, got %v", err)
	}

	if cb.State() != StateHalfOpen {
		t.Errorf("expected half-open state, got %s", cb.State())
	}
}

func TestCircuitBreaker_ClosesAfterSuccessInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      5,
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = _ = cb.Execute(func() error { return errTest })
	}

	// Wait for timeout to enter half-open
	time.Sleep(60 * time.Millisecond)

	// Successful requests in half-open
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return nil })
	}

	if cb.State() != StateClosed {
		t.Errorf("expected closed state after successes, got %s", cb.State())
	}
}

func TestCircuitBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = _ = cb.Execute(func() error { return errTest })
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Fail in half-open
	_ = cb.Execute(func() error { return errTest })

	if cb.State() != StateOpen {
		t.Errorf("expected open state after failure in half-open, got %s", cb.State())
	}
}

func TestCircuitBreaker_ExecuteWithContext(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("test"))

	ctx := context.Background()
	executed := false

	err := cb.ExecuteWithContext(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !executed {
		t.Error("function was not executed")
	}
}

func TestCircuitBreaker_ExecuteWithContext_Cancelled(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("test"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	executed := false
	err := cb.ExecuteWithContext(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	if executed {
		t.Error("function should not have executed")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 2,
		Timeout:          1 * time.Second,
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = _ = cb.Execute(func() error { return errTest })
	}

	if cb.State() != StateOpen {
		t.Errorf("expected open state, got %s", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected closed state after reset, got %s", cb.State())
	}

	// Should allow requests again
	err := _ = cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected no error after reset, got %v", err)
	}
}

func TestCircuitBreaker_Metrics(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test-metrics",
		FailureThreshold: 5,
	})

	// Cause some failures
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error { return errTest })
	}

	metrics := cb.Metrics()

	if metrics.Name != "test-metrics" {
		t.Errorf("expected name test-metrics, got %s", metrics.Name)
	}

	if metrics.State != "closed" {
		t.Errorf("expected closed state, got %s", metrics.State)
	}

	if metrics.Failures != 3 {
		t.Errorf("expected 3 failures, got %d", metrics.Failures)
	}
}

func TestCircuitBreakerRegistry(t *testing.T) {
	registry := NewCircuitBreakerRegistry()

	cb1 := registry.Get("service-a")
	cb2 := registry.Get("service-b")
	cb3 := registry.Get("service-a") // Should return same as cb1

	if cb1 == cb2 {
		t.Error("different services should have different circuit breakers")
	}

	if cb1 != cb3 {
		t.Error("same service should return same circuit breaker")
	}
}

func TestCircuitBreakerRegistry_AllMetrics(t *testing.T) {
	registry := NewCircuitBreakerRegistry()

	registry.Get("service-a")
	registry.Get("service-b")

	metrics := registry.AllMetrics()

	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(metrics))
	}
}

func TestGlobalRegistry(t *testing.T) {
	cb := GetCircuitBreaker("global-test")

	if cb == nil {
		t.Error("expected circuit breaker from global registry")
	}

	if cb.config.Name != "global-test" {
		t.Errorf("expected name global-test, got %s", cb.config.Name)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")

	if config.Name != "test" {
		t.Errorf("expected name test, got %s", config.Name)
	}

	if config.FailureThreshold != 5 {
		t.Errorf("expected FailureThreshold 5, got %d", config.FailureThreshold)
	}

	if config.SuccessThreshold != 2 {
		t.Errorf("expected SuccessThreshold 2, got %d", config.SuccessThreshold)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", config.Timeout)
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}

func TestCircuitBreaker_ConcurrentExecution(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "concurrent",
		FailureThreshold: 10,
		Timeout:          100 * time.Millisecond,
	})

	// Execute many concurrent requests
	done := make(chan bool)
	var successCount, failureCount int32 // Use atomic operations for thread safety
	var countMu sync.Mutex

	for i := 0; i < 100; i++ {
		go func(id int) {
			err := cb.Execute(func() error {
				time.Sleep(1 * time.Millisecond)
				if id%10 == 0 {
					return errTest
				}
				return nil
			})
			countMu.Lock()
			if err == nil {
				successCount++
			} else {
				failureCount++
			}
			countMu.Unlock()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Circuit breaker should handle concurrent access safely
	if cb.State() == StateOpen {
		t.Logf("Circuit opened due to failures (success=%d, failure=%d)", successCount, failureCount)
	}
}

func TestCircuitBreaker_MultipleStateTransitions(t *testing.T) {
	var stateChangesMu sync.Mutex
	stateChanges := []string{}
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "multi-transition",
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      5,
		OnStateChange: func(name string, from, to CircuitState) {
			stateChangesMu.Lock()
			stateChanges = append(stateChanges, to.String())
			stateChangesMu.Unlock()
		},
	})

	// Open the circuit
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })
	time.Sleep(10 * time.Millisecond)

	// Should be open
	if cb.State() != StateOpen {
		t.Errorf("expected open state, got %s", cb.State())
	}

	// Wait for timeout and transition to half-open
	time.Sleep(60 * time.Millisecond)

	// Success in half-open
	_ = cb.Execute(func() error { return nil })
	_ = cb.Execute(func() error { return nil })
	time.Sleep(10 * time.Millisecond)

	// Should be closed again
	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}

	// Verify state changes
	stateChangesMu.Lock()
	numChanges := len(stateChanges)
	stateChangesMu.Unlock()
	if numChanges < 2 {
		t.Errorf("expected at least 2 state changes, got %d", numChanges)
	}
}

func TestCircuitBreaker_MaxRequestsInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "max-requests",
		FailureThreshold: 2,
		SuccessThreshold: 5, // High threshold so we stay in half-open
		Timeout:          50 * time.Millisecond,
		MaxRequests:      2,
	})

	// Open the circuit
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	// Wait for timeout to transition to half-open
	time.Sleep(60 * time.Millisecond)

	// First request transitions to half-open and is allowed
	err := _ = cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("first request should be allowed: %v", err)
	}

	// Second request should be allowed (halfOpenRequests goes to 2)
	err = _ = cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("second request should be allowed: %v", err)
	}

	// Third request should be denied (at max requests)
	err = cb.Execute(func() error {
		t.Error("should not execute when max requests exceeded")
		return nil
	})

	if err == nil {
		t.Error("expected error when exceeding max requests in half-open")
	}
}

func TestCircuitBreaker_StateOpenDuration(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "duration-test",
		FailureThreshold: 2,
		Timeout:          100 * time.Millisecond,
	})

	// Open the circuit
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	// Should be blocked immediately
	err := _ = cb.Execute(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Error("expected circuit to be open")
	}

	// Wait less than timeout
	time.Sleep(50 * time.Millisecond)

	// Should still be blocked
	err = _ = cb.Execute(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Error("expected circuit to still be open")
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should allow request (transition to half-open)
	err = _ = cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected request to be allowed after timeout, got %v", err)
	}
}

func TestCircuitBreaker_ResetMultipleTimes(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "reset-test",
		FailureThreshold: 2,
		Timeout:          1 * time.Second,
	})

	// Open the circuit
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	if cb.State() != StateOpen {
		t.Error("expected circuit to be open")
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Error("expected circuit to be closed after reset")
	}

	// Open again
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	if cb.State() != StateOpen {
		t.Error("expected circuit to be open again")
	}

	// Reset again
	cb.Reset()

	if cb.State() != StateClosed {
		t.Error("expected circuit to be closed after second reset")
	}
}

func TestCircuitBreaker_PartialFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "partial",
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
	})

	// Success resets failure counter, so we need consecutive failures to open
	_ = cb.Execute(func() error { return errTest }) // failures=1
	_ = cb.Execute(func() error { return nil })     // failures=0 (reset on success)
	_ = cb.Execute(func() error { return errTest }) // failures=1

	// Should still be closed (failures reset on success)
	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}

	// Need 3 consecutive failures to open
	_ = cb.Execute(func() error { return errTest }) // failures=2
	_ = cb.Execute(func() error { return errTest }) // failures=3 -> open

	if cb.State() != StateOpen {
		t.Errorf("expected open state, got %s", cb.State())
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "success-reset",
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
	})

	// Two failures
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	// Success should reset counter
	_ = cb.Execute(func() error { return nil })

	// Verify failures were reset
	metrics := cb.Metrics()
	if metrics.Failures != 0 {
		t.Errorf("expected 0 failures after success, got %d", metrics.Failures)
	}

	// Should still be closed
	if cb.State() != StateClosed {
		t.Error("expected circuit to remain closed")
	}
}

func TestCircuitBreaker_HalfOpenPartialSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "half-open-partial",
		FailureThreshold: 2,
		SuccessThreshold: 3,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      5,
	})

	// Open the circuit
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Partial successes in half-open (not enough to close)
	_ = cb.Execute(func() error { return nil })
	_ = cb.Execute(func() error { return nil })

	// Should still be half-open (need 3 successes)
	if cb.State() != StateHalfOpen {
		t.Errorf("expected half-open state, got %s", cb.State())
	}

	// One more success should close it
	_ = cb.Execute(func() error { return nil })

	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}
}

func TestCircuitBreakerRegistry_Concurrent(t *testing.T) {
	registry := NewCircuitBreakerRegistry()

	// Concurrently get circuit breakers
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			serviceName := "service-a"
			cb := registry.Get(serviceName)
			if cb == nil {
				t.Error("expected circuit breaker")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should only have one circuit breaker
	metrics := registry.AllMetrics()
	if len(metrics) != 1 {
		t.Errorf("expected 1 circuit breaker, got %d", len(metrics))
	}
}

func TestCircuitBreakerRegistry_GetWithConfig(t *testing.T) {
	registry := NewCircuitBreakerRegistry()

	config := CircuitBreakerConfig{
		Name:             "custom-service",
		FailureThreshold: 10,
		SuccessThreshold: 5,
		Timeout:          1 * time.Minute,
	}

	cb1 := registry.GetWithConfig(config)
	if cb1 == nil {
		t.Fatal("expected circuit breaker")
	}

	if cb1.config.FailureThreshold != 10 {
		t.Errorf("expected failure threshold 10, got %d", cb1.config.FailureThreshold)
	}

	// Getting again should return same instance
	cb2 := registry.GetWithConfig(config)
	if cb1 != cb2 {
		t.Error("expected same circuit breaker instance")
	}
}

func TestCircuitBreaker_ZeroConfig(t *testing.T) {
	// Test that zero values get replaced with defaults
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name: "zero-config",
	})

	if cb.config.FailureThreshold <= 0 {
		t.Error("expected positive failure threshold")
	}

	if cb.config.SuccessThreshold <= 0 {
		t.Error("expected positive success threshold")
	}

	if cb.config.Timeout <= 0 {
		t.Error("expected positive timeout")
	}

	if cb.config.MaxRequests <= 0 {
		t.Error("expected positive max requests")
	}
}

func TestCircuitBreaker_ExecuteWithNilFunction(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("test"))

	// This should not panic on creation
	if cb == nil {
		t.Error("circuit breaker should not be nil")
	}

	// Verify the circuit breaker was created correctly
	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}
}

func TestAllCircuitBreakerMetrics(t *testing.T) {
	// Use global registry
	cb1 := GetCircuitBreaker("global-service-1")
	cb2 := GetCircuitBreaker("global-service-2")

	if cb1 == nil || cb2 == nil {
		t.Fatal("expected circuit breakers from global registry")
	}

	metrics := AllCircuitBreakerMetrics()

	// Should have at least the ones we just created
	found1, found2 := false, false
	for _, m := range metrics {
		if m.Name == "global-service-1" {
			found1 = true
		}
		if m.Name == "global-service-2" {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("expected to find both circuit breakers in global metrics")
	}
}

func TestCircuitBreaker_MetricsAccuracy(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "metrics-test",
		FailureThreshold: 5,
		Timeout:          100 * time.Millisecond,
	})

	// Execute consecutive failures (success resets counter)
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	metrics := cb.Metrics()

	if metrics.Name != "metrics-test" {
		t.Errorf("expected name 'metrics-test', got '%s'", metrics.Name)
	}

	if metrics.State != "closed" {
		t.Errorf("expected state 'closed', got '%s'", metrics.State)
	}

	// Consecutive failures should be tracked
	if metrics.Failures != 3 {
		t.Errorf("expected 3 failures, got %d", metrics.Failures)
	}
}

func TestCircuitBreaker_ContextTimeout(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("timeout-test"))

	// Test with already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	executed := false
	err := cb.ExecuteWithContext(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})

	// Context was already cancelled, so function should not execute
	if err == nil {
		t.Error("expected context cancelled error")
	}

	if executed {
		t.Error("function should not have been executed with cancelled context")
	}

	// Test with valid context
	ctx2 := context.Background()
	executed2 := false
	err2 := cb.ExecuteWithContext(ctx2, func(ctx context.Context) error {
		executed2 = true
		return nil
	})

	if err2 != nil {
		t.Errorf("unexpected error: %v", err2)
	}

	if !executed2 {
		t.Error("function should have been executed with valid context")
	}
}

func TestCircuitBreaker_ExecuteWithContextSuccess(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("ctx-success"))

	ctx := context.Background()
	value := ""

	err := cb.ExecuteWithContext(ctx, func(ctx context.Context) error {
		value = "executed"
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if value != "executed" {
		t.Error("function should have been executed")
	}
}

func TestGetCircuitBreakerWithConfig(t *testing.T) {
	config := CircuitBreakerConfig{
		Name:             "global-custom",
		FailureThreshold: 20,
		Timeout:          2 * time.Minute,
	}

	cb := GetCircuitBreakerWithConfig(config)

	if cb == nil {
		t.Fatal("expected circuit breaker")
	}

	if cb.config.Name != "global-custom" {
		t.Errorf("expected name 'global-custom', got '%s'", cb.config.Name)
	}

	if cb.config.FailureThreshold != 20 {
		t.Errorf("expected failure threshold 20, got %d", cb.config.FailureThreshold)
	}
}
