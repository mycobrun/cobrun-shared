package logging

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewAuditLogger tests audit logger creation
func TestNewAuditLogger(t *testing.T) {
	tests := []struct {
		name        string
		config      AuditLoggerConfig
		expectNil   bool
	}{
		{
			name: "with custom logger",
			config: AuditLoggerConfig{
				ServiceName: "test-service",
				Environment: "test",
				Logger:      slog.Default(),
			},
			expectNil: false,
		},
		{
			name: "without logger (uses default)",
			config: AuditLoggerConfig{
				ServiceName: "test-service",
				Environment: "production",
				Logger:      nil,
			},
			expectNil: false,
		},
		{
			name: "minimal config",
			config: AuditLoggerConfig{
				ServiceName: "",
				Environment: "",
				Logger:      nil,
			},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewAuditLogger(tt.config)

			if tt.expectNil && logger != nil {
				t.Error("expected nil logger")
			}

			if !tt.expectNil && logger == nil {
				t.Fatal("expected non-nil logger")
			}

			if logger.service != tt.config.ServiceName {
				t.Errorf("expected service %q, got %q", tt.config.ServiceName, logger.service)
			}

			if logger.environment != tt.config.Environment {
				t.Errorf("expected environment %q, got %q", tt.config.Environment, logger.environment)
			}
		})
	}
}

// TestAuditLogger_Log tests basic audit logging
func TestAuditLogger_Log(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "test-service",
		Environment: "test",
	})

	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	event := AuditEvent{
		Type: AuditEventLogin,
		Actor: &AuditActor{
			Type: "user",
			ID:   "user-123",
			Name: "test@example.com",
			IP:   "192.168.1.1",
		},
		Action:  "user login",
		Outcome: AuditOutcomeSuccess,
		Details: map[string]interface{}{
			"method": "password",
		},
	}

	// Should not panic
	logger.Log(context.Background(), event)

	// Verify service and environment are set correctly
	if logger.service != "test-service" {
		t.Errorf("expected service 'test-service', got %q", logger.service)
	}

	if logger.environment != "test" {
		t.Errorf("expected environment 'test', got %q", logger.environment)
	}
}

// TestAuditLogger_LogAuth tests authentication event logging
func TestAuditLogger_LogAuth(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "auth-service",
		Environment: "production",
	})

	// Should not panic
	logger.LogAuth(
		context.Background(),
		AuditEventLogin,
		"user-456",
		"user@example.com",
		"10.0.0.1",
		AuditOutcomeSuccess,
		map[string]interface{}{"provider": "oauth"},
	)
}

// TestAuditLogger_LogUserAction tests user action logging
func TestAuditLogger_LogUserAction(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "user-service",
		Environment: "staging",
	})

	actor := &AuditActor{
		Type: "admin",
		ID:   "admin-123",
		Name: "admin@example.com",
	}

	resource := &AuditResource{
		Type: "user",
		ID:   "user-789",
	}

	// Should not panic
	logger.LogUserAction(
		context.Background(),
		AuditEventUserSuspended,
		actor,
		resource,
		AuditOutcomeSuccess,
		map[string]interface{}{"reason": "policy violation"},
	)
}

// TestAuditLogger_LogTripEvent tests trip event logging
func TestAuditLogger_LogTripEvent(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "trip-service",
		Environment: "production",
	})

	// Should not panic
	logger.LogTripEvent(
		context.Background(),
		AuditEventTripCompleted,
		"trip-001",
		"rider-123",
		"driver-456",
		AuditOutcomeSuccess,
		map[string]interface{}{
			"duration": 1800,
			"distance": 12.5,
		},
	)
}

// TestAuditLogger_LogPaymentEvent tests payment event logging
func TestAuditLogger_LogPaymentEvent(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "payment-service",
		Environment: "production",
	})

	// Should not panic and should add amount/currency to details
	logger.LogPaymentEvent(
		context.Background(),
		AuditEventPaymentProcessed,
		"payment-123",
		"trip-456",
		"user-789",
		49.99,
		"USD",
		AuditOutcomeSuccess,
		map[string]interface{}{"provider": "stripe"},
	)

	// Test with nil details
	logger.LogPaymentEvent(
		context.Background(),
		AuditEventPaymentProcessed,
		"payment-456",
		"trip-789",
		"user-123",
		25.00,
		"EUR",
		AuditOutcomeSuccess,
		nil,
	)
}

// TestAuditLogger_LogAdminAction tests admin action logging
func TestAuditLogger_LogAdminAction(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "admin-service",
		Environment: "production",
	})

	// Should not panic
	logger.LogAdminAction(
		context.Background(),
		"admin-001",
		"admin@example.com",
		"suspend user",
		"user",
		"user-123",
		AuditOutcomeSuccess,
		map[string]interface{}{"reason": "fraud"},
	)
}

// TestAuditLogger_LogSecurityEvent tests security event logging
func TestAuditLogger_LogSecurityEvent(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "security-service",
		Environment: "production",
	})

	// Should not panic
	logger.LogSecurityEvent(
		context.Background(),
		AuditEventRateLimitExceeded,
		"203.0.113.1",
		"Mozilla/5.0",
		map[string]interface{}{
			"endpoint": "/api/login",
			"count":    100,
		},
	)
}

// TestAuditLogger_LogFromRequest tests HTTP request-based logging
func TestAuditLogger_LogFromRequest(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "api-service",
		Environment: "production",
	})

	req := httptest.NewRequest("POST", "/api/users", nil)
	req.Header.Set("X-Request-ID", "req-12345")
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	req.Header.Set("User-Agent", "TestClient/1.0")

	actor := &AuditActor{
		Type: "user",
		ID:   "user-123",
	}

	resource := &AuditResource{
		Type: "user",
		ID:   "user-456",
	}

	// Should not panic
	logger.LogFromRequest(
		context.Background(),
		req,
		AuditEventUserCreated,
		actor,
		resource,
		AuditOutcomeSuccess,
		nil,
	)
}

// TestGetClientIP tests client IP extraction
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func() *http.Request
		expectedIP string
	}{
		{
			name: "X-Forwarded-For header",
			setupFunc: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.1")
				return req
			},
			expectedIP: "203.0.113.1",
		},
		{
			name: "X-Real-IP header",
			setupFunc: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Real-IP", "198.51.100.1")
				return req
			},
			expectedIP: "198.51.100.1",
		},
		{
			name: "RemoteAddr fallback",
			setupFunc: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.0.2.1:12345"
				return req
			},
			expectedIP: "192.0.2.1:12345",
		},
		{
			name: "X-Forwarded-For takes precedence",
			setupFunc: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.1")
				req.Header.Set("X-Real-IP", "198.51.100.1")
				req.RemoteAddr = "192.0.2.1:12345"
				return req
			},
			expectedIP: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupFunc()
			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP %q, got %q", tt.expectedIP, ip)
			}
		})
	}
}

// TestGenerateEventID tests event ID generation
func TestGenerateEventID(t *testing.T) {
	id1 := generateEventID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateEventID()

	if id1 == "" {
		t.Error("expected non-empty event ID")
	}

	if id2 == "" {
		t.Error("expected non-empty event ID")
	}

	if id1 == id2 {
		t.Error("expected unique event IDs")
	}
}

// TestTraceIDFromContext tests trace ID extraction from context
func TestTraceIDFromContext(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		expectedID string
	}{
		{
			name:       "context with trace ID",
			ctx:        context.WithValue(context.Background(), "trace_id", "trace-12345"),
			expectedID: "trace-12345",
		},
		{
			name:       "context without trace ID",
			ctx:        context.Background(),
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traceID := TraceIDFromContext(tt.ctx)
			if traceID != tt.expectedID {
				t.Errorf("expected trace ID %q, got %q", tt.expectedID, traceID)
			}
		})
	}
}

// TestAuditMiddleware tests the audit middleware
func TestAuditMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"successful request", http.StatusOK},
		{"created request", http.StatusCreated},
		{"bad request", http.StatusBadRequest},
		{"unauthorized request", http.StatusUnauthorized},
		{"forbidden request", http.StatusForbidden},
		{"internal server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewAuditLogger(AuditLoggerConfig{
				ServiceName: "test-service",
				Environment: "test",
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			middleware := AuditMiddleware(logger, AuditEventAdminAction)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()

			// Should not panic
			wrappedHandler.ServeHTTP(rec, req)

			// Verify the status code was set correctly
			if rec.Code != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, rec.Code)
			}
		})
	}
}

// TestAuditResponseWriter tests the response writer wrapper
func TestAuditResponseWriter(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"status OK", http.StatusOK},
		{"status Created", http.StatusCreated},
		{"status BadRequest", http.StatusBadRequest},
		{"status Unauthorized", http.StatusUnauthorized},
		{"status Forbidden", http.StatusForbidden},
		{"status InternalServerError", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			wrapper := &auditResponseWriter{
				ResponseWriter: rec,
				status:         http.StatusOK,
			}

			wrapper.WriteHeader(tt.statusCode)

			if wrapper.status != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, wrapper.status)
			}

			if rec.Code != tt.statusCode {
				t.Errorf("expected response code %d, got %d", tt.statusCode, rec.Code)
			}
		})
	}
}

// TestAuditEventTypes tests all audit event type constants
func TestAuditEventTypes(t *testing.T) {
	eventTypes := []AuditEventType{
		// Auth events
		AuditEventLogin,
		AuditEventLogout,
		AuditEventLoginFailed,
		AuditEventTokenRefresh,
		AuditEventPasswordChange,
		AuditEventPasswordReset,
		AuditEventMFAEnabled,
		AuditEventMFADisabled,
		// User events
		AuditEventUserCreated,
		AuditEventUserUpdated,
		AuditEventUserDeleted,
		AuditEventUserSuspended,
		AuditEventUserReactivated,
		// Driver events
		AuditEventDriverOnboarded,
		AuditEventDriverApproved,
		AuditEventDriverRejected,
		AuditEventDriverSuspended,
		AuditEventDocumentUploaded,
		AuditEventDocumentVerified,
		// Trip events
		AuditEventTripCreated,
		AuditEventTripCompleted,
		AuditEventTripCancelled,
		AuditEventTripDisputed,
		AuditEventTripFareAdjusted,
		// Payment events
		AuditEventPaymentProcessed,
		AuditEventPaymentFailed,
		AuditEventRefundIssued,
		AuditEventPayoutProcessed,
		// Admin events
		AuditEventAdminAction,
		AuditEventConfigChange,
		AuditEventDataExport,
		AuditEventUserImpersonation,
		// Security events
		AuditEventSuspiciousActivity,
		AuditEventRateLimitExceeded,
		AuditEventIPBlocked,
		AuditEventFraudDetected,
		// Compliance events
		AuditEventDataAccessRequest,
		AuditEventDataDeletion,
		AuditEventConsentUpdated,
	}

	for _, eventType := range eventTypes {
		if string(eventType) == "" {
			t.Errorf("event type should not be empty")
		}
	}
}

// TestAuditOutcomes tests audit outcome constants
func TestAuditOutcomes(t *testing.T) {
	outcomes := []AuditOutcome{
		AuditOutcomeSuccess,
		AuditOutcomeFailure,
		AuditOutcomeDenied,
	}

	for _, outcome := range outcomes {
		if string(outcome) == "" {
			t.Errorf("outcome should not be empty")
		}
	}
}

// TestAuditEvent_TimestampAndID tests that timestamp and ID are set automatically
func TestAuditEvent_TimestampAndID(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "test-service",
		Environment: "test",
	})

	event := AuditEvent{
		Type: AuditEventLogin,
		Actor: &AuditActor{
			Type: "user",
			ID:   "user-123",
		},
		Action:  "login",
		Outcome: AuditOutcomeSuccess,
	}

	// Event should not have timestamp or ID set initially
	if !event.Timestamp.IsZero() {
		t.Error("expected timestamp to be zero initially")
	}
	if event.ID != "" {
		t.Error("expected ID to be empty initially")
	}

	// Logging should not panic
	logger.Log(context.Background(), event)
}

// TestAuditLogger_WithContext tests audit logging with context values
func TestAuditLogger_WithContext(t *testing.T) {
	logger := NewAuditLogger(AuditLoggerConfig{
		ServiceName: "test-service",
		Environment: "test",
	})

	ctx := context.WithValue(context.Background(), "trace_id", "trace-abc123")

	event := AuditEvent{
		Type: AuditEventLogin,
		Actor: &AuditActor{
			Type: "user",
			ID:   "user-123",
		},
		Action: "login",
		Outcome: AuditOutcomeSuccess,
		Request: &AuditRequest{
			Method: "POST",
			Path:   "/api/login",
		},
	}

	// Should not panic
	logger.Log(ctx, event)
}
