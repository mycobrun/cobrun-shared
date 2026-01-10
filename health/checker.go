// Package health provides health check utilities.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckFunc is a function that performs a health check.
type CheckFunc func(ctx context.Context) error

// Check represents a single health check.
type Check struct {
	Name     string
	CheckFn  CheckFunc
	Critical bool // If true, failure means the service is unhealthy
}

// CheckResult represents the result of a health check.
type CheckResult struct {
	Name    string  `json:"name"`
	Status  Status  `json:"status"`
	Message string  `json:"message,omitempty"`
	Latency float64 `json:"latency_ms"`
}

// HealthResponse is the response for health endpoints.
type HealthResponse struct {
	Status    Status        `json:"status"`
	Timestamp string        `json:"timestamp"`
	Version   string        `json:"version,omitempty"`
	Checks    []CheckResult `json:"checks,omitempty"`
}

// Checker manages health checks.
type Checker struct {
	checks  []Check
	version string
	mu      sync.RWMutex
}

// NewChecker creates a new health checker.
func NewChecker(version string) *Checker {
	return &Checker{
		checks:  make([]Check, 0),
		version: version,
	}
}

// AddCheck adds a health check.
func (c *Checker) AddCheck(name string, fn CheckFunc, critical bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checks = append(c.checks, Check{
		Name:     name,
		CheckFn:  fn,
		Critical: critical,
	})
}

// Check runs all health checks.
func (c *Checker) Check(ctx context.Context) HealthResponse {
	c.mu.RLock()
	checks := c.checks
	c.mu.RUnlock()

	results := make([]CheckResult, len(checks))
	overallStatus := StatusHealthy
	hasDegraded := false

	var wg sync.WaitGroup
	for i, check := range checks {
		wg.Add(1)
		go func(i int, check Check) {
			defer wg.Done()

			start := time.Now()
			err := check.CheckFn(ctx)
			latency := time.Since(start).Seconds() * 1000

			result := CheckResult{
				Name:    check.Name,
				Status:  StatusHealthy,
				Latency: latency,
			}

			if err != nil {
				result.Status = StatusUnhealthy
				result.Message = err.Error()

				if check.Critical {
					overallStatus = StatusUnhealthy
				} else {
					hasDegraded = true
				}
			}

			results[i] = result
		}(i, check)
	}

	wg.Wait()

	if overallStatus == StatusHealthy && hasDegraded {
		overallStatus = StatusDegraded
	}

	return HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   c.version,
		Checks:    results,
	}
}

// LivenessHandler returns an HTTP handler for liveness checks.
// Liveness just checks if the service is running.
func (c *Checker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	}
}

// ReadinessHandler returns an HTTP handler for readiness checks.
// Readiness checks if the service is ready to accept traffic.
func (c *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := c.Check(ctx)

		w.Header().Set("Content-Type", "application/json")

		status := http.StatusOK
		if response.Status == StatusUnhealthy {
			status = http.StatusServiceUnavailable
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(response)
	}
}

// HealthHandler returns an HTTP handler for detailed health checks.
func (c *Checker) HealthHandler() http.HandlerFunc {
	return c.ReadinessHandler()
}

// Common health check functions.

// PingCheck creates a simple ping check that always succeeds.
func PingCheck() CheckFunc {
	return func(ctx context.Context) error {
		return nil
	}
}

// HTTPCheck creates a health check for an HTTP endpoint.
func HTTPCheck(url string, timeout time.Duration) CheckFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		client := &http.Client{Timeout: timeout}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return &CheckError{
				Service: url,
				Code:    resp.StatusCode,
				Message: "unhealthy status code",
			}
		}
		return nil
	}
}

// CheckError represents a health check error.
type CheckError struct {
	Service string
	Code    int
	Message string
}

func (e *CheckError) Error() string {
	return e.Message
}

// DatabasePinger is an interface for database ping functionality.
type DatabasePinger interface {
	PingContext(ctx context.Context) error
}

// DatabaseCheck creates a health check for a database connection.
func DatabaseCheck(db DatabasePinger, timeout time.Duration) CheckFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return db.PingContext(ctx)
	}
}

// RedisClient is an interface for Redis ping functionality.
type RedisClient interface {
	Ping(ctx context.Context) error
}

// RedisCheck creates a health check for a Redis connection.
func RedisCheck(client RedisClient, timeout time.Duration) CheckFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return client.Ping(ctx)
	}
}

// ServiceChecker checks the health of downstream services.
type ServiceChecker struct {
	services map[string]string // name -> health URL
	timeout  time.Duration
}

// NewServiceChecker creates a new service checker.
func NewServiceChecker(timeout time.Duration) *ServiceChecker {
	return &ServiceChecker{
		services: make(map[string]string),
		timeout:  timeout,
	}
}

// AddService adds a service to check.
func (s *ServiceChecker) AddService(name, healthURL string) {
	s.services[name] = healthURL
}

// CheckAll checks all registered services.
func (s *ServiceChecker) CheckAll(ctx context.Context) map[string]error {
	results := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, url := range s.services {
		wg.Add(1)
		go func(name, url string) {
			defer wg.Done()

			checkFn := HTTPCheck(url, s.timeout)
			err := checkFn(ctx)

			mu.Lock()
			results[name] = err
			mu.Unlock()
		}(name, url)
	}

	wg.Wait()
	return results
}

// CheckFunc returns a health check function for all services.
func (s *ServiceChecker) CheckFunc() CheckFunc {
	return func(ctx context.Context) error {
		results := s.CheckAll(ctx)
		var failures []string
		for name, err := range results {
			if err != nil {
				failures = append(failures, name+": "+err.Error())
			}
		}
		if len(failures) > 0 {
			return &CheckError{
				Message: "downstream services unhealthy: " + join(failures, "; "),
			}
		}
		return nil
	}
}

// join concatenates strings with a separator.
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
