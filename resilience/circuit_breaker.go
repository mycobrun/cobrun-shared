// Package resilience provides resilience patterns like circuit breakers.
package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// StateClosed allows requests to pass through.
	StateClosed CircuitState = iota
	// StateOpen blocks all requests.
	StateOpen
	// StateHalfOpen allows a limited number of requests for testing.
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

// Common errors.
var (
	ErrCircuitOpen    = errors.New("circuit breaker is open")
	ErrTooManyRetries = errors.New("too many retries")
)

// CircuitBreakerConfig configures a circuit breaker.
type CircuitBreakerConfig struct {
	// Name identifies this circuit breaker (for logging/metrics).
	Name string

	// FailureThreshold is the number of failures before opening the circuit.
	FailureThreshold int

	// SuccessThreshold is the number of successes in half-open to close.
	SuccessThreshold int

	// Timeout is how long the circuit stays open before transitioning to half-open.
	Timeout time.Duration

	// MaxRequests is the max requests allowed in half-open state.
	MaxRequests int

	// OnStateChange is called when state changes (optional).
	OnStateChange func(name string, from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:             name,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		MaxRequests:      3,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu                sync.RWMutex
	state             CircuitState
	failures          int
	successes         int
	lastFailure       time.Time
	halfOpenRequests  int
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRequests <= 0 {
		config.MaxRequests = 3
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Execute runs the given function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := fn()

	cb.recordResult(err)

	return err
}

// ExecuteWithContext runs the given function with context and circuit breaker protection.
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, fn func(context.Context) error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	// Check context before executing
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	err := fn(ctx)

	cb.recordResult(err)

	return err
}

// allowRequest determines if a request should be allowed.
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailure) >= cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			cb.halfOpenRequests = 1
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenRequests < cb.config.MaxRequests {
			cb.halfOpenRequests++
			return true
		}
		return false

	default:
		return false
	}
}

// recordResult updates state based on success/failure.
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0

	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
		}
	}
}

func (cb *CircuitBreaker) onFailure() {
	switch cb.state {
	case StateClosed:
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}

	case StateHalfOpen:
		cb.transitionTo(StateOpen)
		cb.lastFailure = time.Now()
	}
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0

	if cb.config.OnStateChange != nil {
		// Call in goroutine to avoid blocking
		go cb.config.OnStateChange(cb.config.Name, oldState, newState)
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0
}

// Metrics returns current metrics.
func (cb *CircuitBreaker) Metrics() CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return CircuitBreakerMetrics{
		Name:        cb.config.Name,
		State:       cb.state.String(),
		Failures:    cb.failures,
		Successes:   cb.successes,
		LastFailure: cb.lastFailure,
	}
}

// CircuitBreakerMetrics contains circuit breaker statistics.
type CircuitBreakerMetrics struct {
	Name        string    `json:"name"`
	State       string    `json:"state"`
	Failures    int       `json:"failures"`
	Successes   int       `json:"successes"`
	LastFailure time.Time `json:"last_failure,omitempty"`
}

// CircuitBreakerRegistry manages multiple circuit breakers.
type CircuitBreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
}

// NewCircuitBreakerRegistry creates a new registry.
func NewCircuitBreakerRegistry() *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// Get returns a circuit breaker by name, creating it if needed.
func (r *CircuitBreakerRegistry) Get(name string) *CircuitBreaker {
	r.mu.RLock()
	cb, exists := r.breakers[name]
	r.mu.RUnlock()

	if exists {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = r.breakers[name]; exists {
		return cb
	}

	cb = NewCircuitBreaker(DefaultCircuitBreakerConfig(name))
	r.breakers[name] = cb
	return cb
}

// GetWithConfig returns a circuit breaker with custom config.
func (r *CircuitBreakerRegistry) GetWithConfig(config CircuitBreakerConfig) *CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cb, exists := r.breakers[config.Name]; exists {
		return cb
	}

	cb := NewCircuitBreaker(config)
	r.breakers[config.Name] = cb
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

// GetCircuitBreakerWithConfig returns a circuit breaker with custom config.
func GetCircuitBreakerWithConfig(config CircuitBreakerConfig) *CircuitBreaker {
	return globalRegistry.GetWithConfig(config)
}

// AllCircuitBreakerMetrics returns metrics for all circuit breakers.
func AllCircuitBreakerMetrics() []CircuitBreakerMetrics {
	return globalRegistry.AllMetrics()
}
