package logging

import (
	"errors"
	"testing"
	"time"
)

// TestNewAppInsightsClient tests creation of AppInsights client
func TestNewAppInsightsClient(t *testing.T) {
	tests := []struct {
		name               string
		instrumentationKey string
		expectNil          bool
	}{
		{
			name:               "valid instrumentation key",
			instrumentationKey: "test-key-12345",
			expectNil:          false,
		},
		{
			name:               "empty instrumentation key",
			instrumentationKey: "",
			expectNil:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewAppInsightsClient(tt.instrumentationKey)

			if tt.expectNil && client != nil {
				t.Error("expected nil client for empty instrumentation key")
			}

			if !tt.expectNil && client == nil {
				t.Error("expected non-nil client for valid instrumentation key")
			}

			if !tt.expectNil && client != nil && client.client == nil {
				t.Error("expected internal client to be initialized")
			}
		})
	}
}

// TestAppInsightsClient_TrackEvent tests event tracking
func TestAppInsightsClient_TrackEvent(t *testing.T) {
	tests := []struct {
		name       string
		client     *AppInsightsClient
		eventName  string
		properties map[string]string
		shouldPanic bool
	}{
		{
			name:       "valid event with properties",
			client:     NewAppInsightsClient("test-key"),
			eventName:  "TestEvent",
			properties: map[string]string{"key1": "value1", "key2": "value2"},
			shouldPanic: false,
		},
		{
			name:       "valid event without properties",
			client:     NewAppInsightsClient("test-key"),
			eventName:  "TestEvent",
			properties: nil,
			shouldPanic: false,
		},
		{
			name:       "nil client",
			client:     nil,
			eventName:  "TestEvent",
			properties: map[string]string{"key": "value"},
			shouldPanic: false,
		},
		{
			name:       "empty instrumentation key client",
			client:     NewAppInsightsClient(""),
			eventName:  "TestEvent",
			properties: map[string]string{"key": "value"},
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			// Should not panic even with nil client
			tt.client.TrackEvent(tt.eventName, tt.properties)
		})
	}
}

// TestAppInsightsClient_TrackMetric tests metric tracking
func TestAppInsightsClient_TrackMetric(t *testing.T) {
	tests := []struct {
		name        string
		client      *AppInsightsClient
		metricName  string
		metricValue float64
		shouldPanic bool
	}{
		{
			name:        "valid metric positive value",
			client:      NewAppInsightsClient("test-key"),
			metricName:  "ResponseTime",
			metricValue: 123.45,
			shouldPanic: false,
		},
		{
			name:        "valid metric zero value",
			client:      NewAppInsightsClient("test-key"),
			metricName:  "ErrorCount",
			metricValue: 0,
			shouldPanic: false,
		},
		{
			name:        "valid metric negative value",
			client:      NewAppInsightsClient("test-key"),
			metricName:  "Temperature",
			metricValue: -10.5,
			shouldPanic: false,
		},
		{
			name:        "nil client",
			client:      nil,
			metricName:  "Metric",
			metricValue: 100,
			shouldPanic: false,
		},
		{
			name:        "empty instrumentation key client",
			client:      NewAppInsightsClient(""),
			metricName:  "Metric",
			metricValue: 100,
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			tt.client.TrackMetric(tt.metricName, tt.metricValue)
		})
	}
}

// TestAppInsightsClient_TrackException tests exception tracking
func TestAppInsightsClient_TrackException(t *testing.T) {
	tests := []struct {
		name        string
		client      *AppInsightsClient
		err         error
		shouldPanic bool
	}{
		{
			name:        "valid exception",
			client:      NewAppInsightsClient("test-key"),
			err:         errors.New("test error"),
			shouldPanic: false,
		},
		{
			name:        "nil error",
			client:      NewAppInsightsClient("test-key"),
			err:         nil,
			shouldPanic: false,
		},
		{
			name:        "nil client",
			client:      nil,
			err:         errors.New("test error"),
			shouldPanic: false,
		},
		{
			name:        "empty instrumentation key client",
			client:      NewAppInsightsClient(""),
			err:         errors.New("test error"),
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			tt.client.TrackException(tt.err)
		})
	}
}

// TestAppInsightsClient_TrackRequest tests HTTP request tracking
func TestAppInsightsClient_TrackRequest(t *testing.T) {
	tests := []struct {
		name         string
		client       *AppInsightsClient
		requestName  string
		url          string
		duration     time.Duration
		responseCode string
		success      bool
		shouldPanic  bool
	}{
		{
			name:         "successful request",
			client:       NewAppInsightsClient("test-key"),
			requestName:  "GET /api/users",
			url:          "https://api.example.com/users",
			duration:     100 * time.Millisecond,
			responseCode: "200",
			success:      true,
			shouldPanic:  false,
		},
		{
			name:         "failed request",
			client:       NewAppInsightsClient("test-key"),
			requestName:  "POST /api/orders",
			url:          "https://api.example.com/orders",
			duration:     250 * time.Millisecond,
			responseCode: "500",
			success:      false,
			shouldPanic:  false,
		},
		{
			name:         "nil client",
			client:       nil,
			requestName:  "GET /api/test",
			url:          "https://api.example.com/test",
			duration:     50 * time.Millisecond,
			responseCode: "200",
			success:      true,
			shouldPanic:  false,
		},
		{
			name:         "empty instrumentation key client",
			client:       NewAppInsightsClient(""),
			requestName:  "GET /api/test",
			url:          "https://api.example.com/test",
			duration:     50 * time.Millisecond,
			responseCode: "200",
			success:      true,
			shouldPanic:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			tt.client.TrackRequest(tt.requestName, tt.url, tt.duration, tt.responseCode, tt.success)
		})
	}
}

// TestAppInsightsClient_TrackDependency tests dependency tracking
func TestAppInsightsClient_TrackDependency(t *testing.T) {
	tests := []struct {
		name           string
		client         *AppInsightsClient
		depName        string
		depType        string
		target         string
		data           string
		duration       time.Duration
		success        bool
		shouldPanic    bool
	}{
		{
			name:        "successful database dependency",
			client:      NewAppInsightsClient("test-key"),
			depName:     "SELECT * FROM users",
			depType:     "SQL",
			target:      "mydb.database.windows.net",
			data:        "SELECT * FROM users WHERE id = ?",
			duration:    50 * time.Millisecond,
			success:     true,
			shouldPanic: false,
		},
		{
			name:        "successful HTTP dependency",
			client:      NewAppInsightsClient("test-key"),
			depName:     "GET https://api.external.com/data",
			depType:     "HTTP",
			target:      "api.external.com",
			data:        "/data",
			duration:    200 * time.Millisecond,
			success:     true,
			shouldPanic: false,
		},
		{
			name:        "failed dependency",
			client:      NewAppInsightsClient("test-key"),
			depName:     "Redis GET key",
			depType:     "Redis",
			target:      "cache.redis.cache.windows.net",
			data:        "GET mykey",
			duration:    10 * time.Millisecond,
			success:     false,
			shouldPanic: false,
		},
		{
			name:        "nil client",
			client:      nil,
			depName:     "Test",
			depType:     "Test",
			target:      "target",
			data:        "data",
			duration:    10 * time.Millisecond,
			success:     true,
			shouldPanic: false,
		},
		{
			name:        "empty instrumentation key client",
			client:      NewAppInsightsClient(""),
			depName:     "Test",
			depType:     "Test",
			target:      "target",
			data:        "data",
			duration:    10 * time.Millisecond,
			success:     true,
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			tt.client.TrackDependency(tt.depName, tt.depType, tt.target, tt.data, tt.duration, tt.success)
		})
	}
}

// TestAppInsightsClient_Flush tests flushing telemetry
func TestAppInsightsClient_Flush(t *testing.T) {
	tests := []struct {
		name        string
		client      *AppInsightsClient
		shouldPanic bool
	}{
		{
			name:        "flush with valid client",
			client:      NewAppInsightsClient("test-key"),
			shouldPanic: false,
		},
		{
			name:        "flush with nil client",
			client:      nil,
			shouldPanic: false,
		},
		{
			name:        "flush with empty instrumentation key client",
			client:      NewAppInsightsClient(""),
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			tt.client.Flush()
		})
	}
}

// TestAppInsightsClient_Close tests closing the client
func TestAppInsightsClient_Close(t *testing.T) {
	tests := []struct {
		name        string
		client      *AppInsightsClient
		shouldPanic bool
	}{
		{
			name:        "close with valid client",
			client:      NewAppInsightsClient("test-key"),
			shouldPanic: false,
		},
		{
			name:        "close with nil client",
			client:      nil,
			shouldPanic: false,
		},
		{
			name:        "close with empty instrumentation key client",
			client:      NewAppInsightsClient(""),
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			tt.client.Close()
		})
	}
}

// TestAppInsightsClient_MultipleOperations tests chaining multiple operations
func TestAppInsightsClient_MultipleOperations(t *testing.T) {
	client := NewAppInsightsClient("test-key")
	if client == nil {
		t.Skip("skipping test with real client")
	}

	// Should be able to perform multiple operations without panic
	client.TrackEvent("Event1", map[string]string{"key": "value"})
	client.TrackMetric("Metric1", 100.0)
	client.TrackException(errors.New("test error"))
	client.TrackRequest("Request1", "http://example.com", 100*time.Millisecond, "200", true)
	client.TrackDependency("Dep1", "SQL", "db.example.com", "SELECT 1", 50*time.Millisecond, true)
	client.Flush()
	client.Close()
}

// TestAppInsightsClient_NilClientSafety tests that all operations are safe with nil client
func TestAppInsightsClient_NilClientSafety(t *testing.T) {
	var client *AppInsightsClient = nil

	// All operations should be safe with nil client
	client.TrackEvent("Event", map[string]string{"key": "value"})
	client.TrackMetric("Metric", 100.0)
	client.TrackException(errors.New("error"))
	client.TrackRequest("Request", "http://example.com", 100*time.Millisecond, "200", true)
	client.TrackDependency("Dep", "SQL", "db", "SELECT 1", 50*time.Millisecond, true)
	client.Flush()
	client.Close()
}
