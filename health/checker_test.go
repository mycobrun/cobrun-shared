// Package health provides health check utilities.
package health

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

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusHealthy, "healthy"},
		{StatusUnhealthy, "unhealthy"},
		{StatusDegraded, "degraded"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("status = %s, want %s", tt.status, tt.expected)
		}
	}
}

func TestNewChecker(t *testing.T) {
	checker := NewChecker("1.0.0")
	if checker == nil {
		t.Fatal("NewChecker returned nil")
	}

	if checker.version != "1.0.0" {
		t.Errorf("version = %s, want 1.0.0", checker.version)
	}

	if len(checker.checks) != 0 {
		t.Error("new checker should have no checks")
	}
}

func TestChecker_AddCheck(t *testing.T) {
	checker := NewChecker("1.0.0")

	checker.AddCheck("test", PingCheck(), false)
	if len(checker.checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(checker.checks))
	}

	checker.AddCheck("test2", PingCheck(), true)
	if len(checker.checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(checker.checks))
	}

	// Verify check properties
	if checker.checks[0].Name != "test" {
		t.Errorf("first check name = %s, want test", checker.checks[0].Name)
	}
	if checker.checks[0].Critical {
		t.Error("first check should not be critical")
	}
	if !checker.checks[1].Critical {
		t.Error("second check should be critical")
	}
}

func TestChecker_Check_AllHealthy(t *testing.T) {
	checker := NewChecker("1.0.0")
	checker.AddCheck("ping1", PingCheck(), false)
	checker.AddCheck("ping2", PingCheck(), true)

	response := checker.Check(context.Background())

	if response.Status != StatusHealthy {
		t.Errorf("status = %s, want healthy", response.Status)
	}

	if response.Version != "1.0.0" {
		t.Errorf("version = %s, want 1.0.0", response.Version)
	}

	if len(response.Checks) != 2 {
		t.Errorf("expected 2 check results, got %d", len(response.Checks))
	}

	for _, check := range response.Checks {
		if check.Status != StatusHealthy {
			t.Errorf("check %s status = %s, want healthy", check.Name, check.Status)
		}
		if check.Message != "" {
			t.Errorf("healthy check should have no message, got %s", check.Message)
		}
	}
}

func TestChecker_Check_CriticalFailure(t *testing.T) {
	checker := NewChecker("1.0.0")
	checker.AddCheck("healthy", PingCheck(), false)
	checker.AddCheck("failing", func(ctx context.Context) error {
		return errors.New("critical failure")
	}, true) // Critical check

	response := checker.Check(context.Background())

	if response.Status != StatusUnhealthy {
		t.Errorf("status = %s, want unhealthy", response.Status)
	}

	// Find the failing check
	var failingCheck *CheckResult
	for i := range response.Checks {
		if response.Checks[i].Name == "failing" {
			failingCheck = &response.Checks[i]
			break
		}
	}

	if failingCheck == nil {
		t.Fatal("failing check not found in results")
	}

	if failingCheck.Status != StatusUnhealthy {
		t.Errorf("failing check status = %s, want unhealthy", failingCheck.Status)
	}

	if failingCheck.Message != "critical failure" {
		t.Errorf("failing check message = %s, want 'critical failure'", failingCheck.Message)
	}
}

func TestChecker_Check_NonCriticalFailure(t *testing.T) {
	checker := NewChecker("1.0.0")
	checker.AddCheck("healthy", PingCheck(), true)
	checker.AddCheck("failing", func(ctx context.Context) error {
		return errors.New("non-critical failure")
	}, false) // Non-critical check

	response := checker.Check(context.Background())

	// Should be degraded, not unhealthy
	if response.Status != StatusDegraded {
		t.Errorf("status = %s, want degraded", response.Status)
	}
}

func TestChecker_Check_Latency(t *testing.T) {
	checker := NewChecker("1.0.0")
	checker.AddCheck("slow", func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}, false)

	response := checker.Check(context.Background())

	if len(response.Checks) != 1 {
		t.Fatal("expected 1 check result")
	}

	// Latency should be at least 10ms
	if response.Checks[0].Latency < 10 {
		t.Errorf("latency = %f, expected at least 10ms", response.Checks[0].Latency)
	}
}

func TestChecker_Check_Timestamp(t *testing.T) {
	checker := NewChecker("1.0.0")
	response := checker.Check(context.Background())

	if response.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}

	// Verify timestamp is valid RFC3339
	_, err := time.Parse(time.RFC3339, response.Timestamp)
	if err != nil {
		t.Errorf("timestamp is not valid RFC3339: %v", err)
	}
}

func TestChecker_LivenessHandler(t *testing.T) {
	checker := NewChecker("1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler := checker.LivenessHandler()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want 200", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("content-type = %s, want application/json", rec.Header().Get("Content-Type"))
	}

	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "alive") {
		t.Error("response should contain 'alive'")
	}
}

func TestChecker_ReadinessHandler_Healthy(t *testing.T) {
	checker := NewChecker("1.0.0")
	checker.AddCheck("ping", PingCheck(), true)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler := checker.ReadinessHandler()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want 200", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "healthy") {
		t.Error("response should contain 'healthy'")
	}
}

func TestChecker_ReadinessHandler_Unhealthy(t *testing.T) {
	checker := NewChecker("1.0.0")
	checker.AddCheck("failing", func(ctx context.Context) error {
		return errors.New("failure")
	}, true)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler := checker.ReadinessHandler()
	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status code = %d, want 503", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "unhealthy") {
		t.Error("response should contain 'unhealthy'")
	}
}

func TestChecker_HealthHandler(t *testing.T) {
	checker := NewChecker("1.0.0")

	// HealthHandler should be the same as ReadinessHandler
	h1 := checker.HealthHandler()
	h2 := checker.ReadinessHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	rec1 := httptest.NewRecorder()
	rec2 := httptest.NewRecorder()

	h1(rec1, req)
	h2(rec2, req)

	if rec1.Code != rec2.Code {
		t.Error("HealthHandler and ReadinessHandler should return same status")
	}
}

func TestPingCheck(t *testing.T) {
	check := PingCheck()
	err := check(context.Background())

	if err != nil {
		t.Errorf("PingCheck should always succeed, got error: %v", err)
	}
}

func TestHTTPCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := HTTPCheck(server.URL, time.Second)
	err := check(context.Background())

	if err != nil {
		t.Errorf("HTTPCheck should succeed for 200 response, got error: %v", err)
	}
}

func TestHTTPCheck_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	check := HTTPCheck(server.URL, time.Second)
	err := check(context.Background())

	if err == nil {
		t.Error("HTTPCheck should fail for 500 response")
	}

	var checkErr *CheckError
	if errors.As(err, &checkErr) {
		if checkErr.Code != http.StatusInternalServerError {
			t.Errorf("error code = %d, want 500", checkErr.Code)
		}
	}
}

func TestHTTPCheck_ConnectionFailure(t *testing.T) {
	// Use a URL that will fail to connect
	check := HTTPCheck("http://localhost:99999", 100*time.Millisecond)
	err := check(context.Background())

	if err == nil {
		t.Error("HTTPCheck should fail for connection error")
	}
}

func TestHTTPCheck_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := HTTPCheck(server.URL, 50*time.Millisecond)
	err := check(context.Background())

	if err == nil {
		t.Error("HTTPCheck should fail on timeout")
	}
}

func TestCheckError(t *testing.T) {
	err := &CheckError{
		Service: "test-service",
		Code:    500,
		Message: "internal error",
	}

	if err.Error() != "internal error" {
		t.Errorf("Error() = %s, want 'internal error'", err.Error())
	}

	if err.Service != "test-service" {
		t.Errorf("Service = %s, want 'test-service'", err.Service)
	}

	if err.Code != 500 {
		t.Errorf("Code = %d, want 500", err.Code)
	}
}

// Mock database pinger for testing
type mockDatabasePinger struct {
	err error
}

func (m *mockDatabasePinger) PingContext(ctx context.Context) error {
	return m.err
}

func TestDatabaseCheck_Success(t *testing.T) {
	db := &mockDatabasePinger{err: nil}
	check := DatabaseCheck(db, time.Second)

	err := check(context.Background())
	if err != nil {
		t.Errorf("DatabaseCheck should succeed, got error: %v", err)
	}
}

func TestDatabaseCheck_Failure(t *testing.T) {
	db := &mockDatabasePinger{err: errors.New("connection refused")}
	check := DatabaseCheck(db, time.Second)

	err := check(context.Background())
	if err == nil {
		t.Error("DatabaseCheck should fail when ping fails")
	}
}

// Mock Redis client for testing
type mockRedisClient struct {
	err error
}

func (m *mockRedisClient) Ping(ctx context.Context) error {
	return m.err
}

func TestRedisCheck_Success(t *testing.T) {
	client := &mockRedisClient{err: nil}
	check := RedisCheck(client, time.Second)

	err := check(context.Background())
	if err != nil {
		t.Errorf("RedisCheck should succeed, got error: %v", err)
	}
}

func TestRedisCheck_Failure(t *testing.T) {
	client := &mockRedisClient{err: errors.New("connection refused")}
	check := RedisCheck(client, time.Second)

	err := check(context.Background())
	if err == nil {
		t.Error("RedisCheck should fail when ping fails")
	}
}

func TestNewServiceChecker(t *testing.T) {
	sc := NewServiceChecker(time.Second)
	if sc == nil {
		t.Fatal("NewServiceChecker returned nil")
	}

	if sc.timeout != time.Second {
		t.Errorf("timeout = %v, want 1s", sc.timeout)
	}

	if len(sc.services) != 0 {
		t.Error("new service checker should have no services")
	}
}

func TestServiceChecker_AddService(t *testing.T) {
	sc := NewServiceChecker(time.Second)

	sc.AddService("api", "http://api:8080/health")
	if len(sc.services) != 1 {
		t.Errorf("expected 1 service, got %d", len(sc.services))
	}

	sc.AddService("db", "http://db:5432/health")
	if len(sc.services) != 2 {
		t.Errorf("expected 2 services, got %d", len(sc.services))
	}

	if sc.services["api"] != "http://api:8080/health" {
		t.Errorf("api URL = %s, want http://api:8080/health", sc.services["api"])
	}
}

func TestServiceChecker_CheckAll(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server2.Close()

	sc := NewServiceChecker(time.Second)
	sc.AddService("healthy", server1.URL)
	sc.AddService("unhealthy", server2.URL)

	results := sc.CheckAll(context.Background())

	if results["healthy"] != nil {
		t.Errorf("healthy service should have no error, got: %v", results["healthy"])
	}

	if results["unhealthy"] == nil {
		t.Error("unhealthy service should have error")
	}
}

func TestServiceChecker_CheckFunc_AllHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sc := NewServiceChecker(time.Second)
	sc.AddService("service1", server.URL)
	sc.AddService("service2", server.URL)

	check := sc.CheckFunc()
	err := check(context.Background())

	if err != nil {
		t.Errorf("CheckFunc should succeed when all services healthy, got: %v", err)
	}
}

func TestServiceChecker_CheckFunc_SomeUnhealthy(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()

	sc := NewServiceChecker(time.Second)
	sc.AddService("healthy", healthy.URL)
	sc.AddService("unhealthy", unhealthy.URL)

	check := sc.CheckFunc()
	err := check(context.Background())

	if err == nil {
		t.Error("CheckFunc should fail when some services unhealthy")
	}

	if !strings.Contains(err.Error(), "downstream services unhealthy") {
		t.Errorf("error message should contain 'downstream services unhealthy', got: %s", err.Error())
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		name     string
		strs     []string
		sep      string
		expected string
	}{
		{"empty slice", []string{}, ", ", ""},
		{"single element", []string{"a"}, ", ", "a"},
		{"two elements", []string{"a", "b"}, ", ", "a, b"},
		{"three elements", []string{"a", "b", "c"}, "; ", "a; b; c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := join(tt.strs, tt.sep)
			if result != tt.expected {
				t.Errorf("join() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestChecker_Check_NoChecks(t *testing.T) {
	checker := NewChecker("1.0.0")
	response := checker.Check(context.Background())

	if response.Status != StatusHealthy {
		t.Errorf("status = %s, want healthy (no checks means healthy)", response.Status)
	}

	if len(response.Checks) != 0 {
		t.Errorf("expected 0 check results, got %d", len(response.Checks))
	}
}

func TestChecker_ConcurrentAddCheck(t *testing.T) {
	checker := NewChecker("1.0.0")

	// Add checks concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			checker.AddCheck("test", PingCheck(), false)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if len(checker.checks) != 10 {
		t.Errorf("expected 10 checks, got %d", len(checker.checks))
	}
}

func TestChecker_ConcurrentCheck(t *testing.T) {
	checker := NewChecker("1.0.0")
	checker.AddCheck("slow", func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}, false)

	// Run checks concurrently
	done := make(chan HealthResponse)
	for i := 0; i < 5; i++ {
		go func() {
			done <- checker.Check(context.Background())
		}()
	}

	// Verify all responses
	for i := 0; i < 5; i++ {
		resp := <-done
		if resp.Status != StatusHealthy {
			t.Errorf("concurrent check returned %s, want healthy", resp.Status)
		}
	}
}

func TestCheckResult_Fields(t *testing.T) {
	result := CheckResult{
		Name:    "database",
		Status:  StatusHealthy,
		Message: "",
		Latency: 5.5,
	}

	if result.Name != "database" {
		t.Error("Name field mismatch")
	}
	if result.Status != StatusHealthy {
		t.Error("Status field mismatch")
	}
	if result.Latency != 5.5 {
		t.Error("Latency field mismatch")
	}
}

func TestHealthResponse_Fields(t *testing.T) {
	resp := HealthResponse{
		Status:    StatusHealthy,
		Timestamp: "2024-01-01T00:00:00Z",
		Version:   "1.0.0",
		Checks:    []CheckResult{},
	}

	if resp.Status != StatusHealthy {
		t.Error("Status field mismatch")
	}
	if resp.Version != "1.0.0" {
		t.Error("Version field mismatch")
	}
	if resp.Timestamp != "2024-01-01T00:00:00Z" {
		t.Error("Timestamp field mismatch")
	}
}
