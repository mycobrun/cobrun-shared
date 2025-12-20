// Package http provides HTTP utilities including circuit breaker for resilient service calls.
package http

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// StateClosed allows requests to pass through normally.
	StateClosed CircuitState = iota
	// StateOpen rejects all requests immediately.
	StateOpen
	// StateHalfOpen allows a limited number of requests to test recovery.
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for the circuit breaker.
type CircuitBreakerConfig struct {
	// Name identifies the circuit breaker (usually the service name).
	Name string
	// FailureThreshold is the number of failures before opening the circuit.
	FailureThreshold int
	// SuccessThreshold is the number of successes in half-open state to close.
	SuccessThreshold int
	// Timeout is how long to wait before transitioning from open to half-open.
	Timeout time.Duration
	// MaxConcurrentInHalfOpen limits concurrent requests in half-open state.
	MaxConcurrentInHalfOpen int
	// OnStateChange is called when the circuit state changes.
	OnStateChange func(name string, from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns sensible production defaults.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:                    name,
		FailureThreshold:        5,
		SuccessThreshold:        2,
		Timeout:                 30 * time.Second,
		MaxConcurrentInHalfOpen: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern for resilient service calls.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu                 sync.RWMutex
	state              CircuitState
	failures           int
	successes          int
	lastFailure        time.Time
	halfOpenInProgress int
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Common errors.
var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// Execute runs the given function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := cb.canExecute(); err != nil {
		return err
	}

	// Track half-open request
	if cb.getState() == StateHalfOpen {
		cb.mu.Lock()
		cb.halfOpenInProgress++
		cb.mu.Unlock()
		defer func() {
			cb.mu.Lock()
			cb.halfOpenInProgress--
			cb.mu.Unlock()
		}()
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.recordResult(err)

	return err
}

// ExecuteWithFallback runs the function with fallback on circuit open.
func (cb *CircuitBreaker) ExecuteWithFallback(ctx context.Context, fn func() error, fallback func() error) error {
	err := cb.Execute(ctx, fn)
	if errors.Is(err, ErrCircuitOpen) || errors.Is(err, ErrTooManyRequests) {
		return fallback()
	}
	return err
}

// canExecute checks if a request can proceed.
func (cb *CircuitBreaker) canExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			return nil
		}
		return fmt.Errorf("%w: %s", ErrCircuitOpen, cb.config.Name)

	case StateHalfOpen:
		// Limit concurrent requests in half-open state
		if cb.halfOpenInProgress >= cb.config.MaxConcurrentInHalfOpen {
			return fmt.Errorf("%w: %s", ErrTooManyRequests, cb.config.Name)
		}
		return nil
	}

	return nil
}

// recordResult records the result of an execution.
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure handles a failed execution.
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailure = time.Now()
	cb.successes = 0

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		// A single failure in half-open returns to open
		cb.transitionTo(StateOpen)
	}
}

// onSuccess handles a successful execution.
func (cb *CircuitBreaker) onSuccess() {
	cb.successes++

	switch cb.state {
	case StateClosed:
		// Reset failure count on success
		cb.failures = 0
	case StateHalfOpen:
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
		}
	}
}

// transitionTo changes the circuit state.
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.state
	cb.state = newState
	cb.failures = 0
	cb.successes = 0

	if cb.config.OnStateChange != nil {
		// Call outside of lock to prevent deadlocks
		go cb.config.OnStateChange(cb.config.Name, oldState, newState)
	}
}

// getState returns the current state.
func (cb *CircuitBreaker) getState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	return cb.getState()
}

// Metrics returns current circuit breaker metrics.
func (cb *CircuitBreaker) Metrics() CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerMetrics{
		Name:        cb.config.Name,
		State:       cb.state,
		Failures:    cb.failures,
		Successes:   cb.successes,
		LastFailure: cb.lastFailure,
	}
}

// CircuitBreakerMetrics holds metrics for monitoring.
type CircuitBreakerMetrics struct {
	Name        string
	State       CircuitState
	Failures    int
	Successes   int
	LastFailure time.Time
}

// CircuitBreakerRegistry manages multiple circuit breakers.
type CircuitBreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig // Default config for new breakers
}

// NewCircuitBreakerRegistry creates a new registry.
func NewCircuitBreakerRegistry() *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
		config:   DefaultCircuitBreakerConfig(""),
	}
}

// Get returns or creates a circuit breaker for the given name.
func (r *CircuitBreakerRegistry) Get(name string) *CircuitBreaker {
	r.mu.RLock()
	cb, exists := r.breakers[name]
	r.mu.RUnlock()

	if exists {
		return cb
	}

	// Create new breaker
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = r.breakers[name]; exists {
		return cb
	}

	config := r.config
	config.Name = name
	cb = NewCircuitBreaker(config)
	r.breakers[name] = cb

	return cb
}

// AllMetrics returns metrics for all circuit breakers.
func (r *CircuitBreakerRegistry) AllMetrics() []CircuitBreakerMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make([]CircuitBreakerMetrics, 0, len(r.breakers))
	for _, cb := range r.breakers {
		metrics = append(metrics, cb.Metrics())
	}
	return metrics
}

// Global registry for convenience.
var globalRegistry = NewCircuitBreakerRegistry()

// GetCircuitBreaker returns a circuit breaker from the global registry.
func GetCircuitBreaker(name string) *CircuitBreaker {
	return globalRegistry.Get(name)
}

// AllCircuitBreakerMetrics returns metrics from the global registry.
func AllCircuitBreakerMetrics() []CircuitBreakerMetrics {
	return globalRegistry.AllMetrics()
}

// SetGlobalConfig sets the default config for new circuit breakers.
func SetGlobalConfig(config CircuitBreakerConfig) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.config = config
}
