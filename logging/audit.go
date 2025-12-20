// Package logging provides audit logging for compliance and security.
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// AuditEventType represents the type of audit event.
type AuditEventType string

const (
	// Authentication events
	AuditEventLogin           AuditEventType = "auth.login"
	AuditEventLogout          AuditEventType = "auth.logout"
	AuditEventLoginFailed     AuditEventType = "auth.login_failed"
	AuditEventTokenRefresh    AuditEventType = "auth.token_refresh"
	AuditEventPasswordChange  AuditEventType = "auth.password_change"
	AuditEventPasswordReset   AuditEventType = "auth.password_reset"
	AuditEventMFAEnabled      AuditEventType = "auth.mfa_enabled"
	AuditEventMFADisabled     AuditEventType = "auth.mfa_disabled"

	// User events
	AuditEventUserCreated     AuditEventType = "user.created"
	AuditEventUserUpdated     AuditEventType = "user.updated"
	AuditEventUserDeleted     AuditEventType = "user.deleted"
	AuditEventUserSuspended   AuditEventType = "user.suspended"
	AuditEventUserReactivated AuditEventType = "user.reactivated"

	// Driver events
	AuditEventDriverOnboarded AuditEventType = "driver.onboarded"
	AuditEventDriverApproved  AuditEventType = "driver.approved"
	AuditEventDriverRejected  AuditEventType = "driver.rejected"
	AuditEventDriverSuspended AuditEventType = "driver.suspended"
	AuditEventDocumentUploaded AuditEventType = "driver.document_uploaded"
	AuditEventDocumentVerified AuditEventType = "driver.document_verified"

	// Trip events
	AuditEventTripCreated    AuditEventType = "trip.created"
	AuditEventTripCompleted  AuditEventType = "trip.completed"
	AuditEventTripCancelled  AuditEventType = "trip.cancelled"
	AuditEventTripDisputed   AuditEventType = "trip.disputed"
	AuditEventTripFareAdjusted AuditEventType = "trip.fare_adjusted"

	// Payment events
	AuditEventPaymentProcessed AuditEventType = "payment.processed"
	AuditEventPaymentFailed    AuditEventType = "payment.failed"
	AuditEventRefundIssued     AuditEventType = "payment.refund"
	AuditEventPayoutProcessed  AuditEventType = "payment.payout"

	// Admin events
	AuditEventAdminAction     AuditEventType = "admin.action"
	AuditEventConfigChange    AuditEventType = "admin.config_change"
	AuditEventDataExport      AuditEventType = "admin.data_export"
	AuditEventUserImpersonation AuditEventType = "admin.impersonation"

	// Security events
	AuditEventSuspiciousActivity AuditEventType = "security.suspicious"
	AuditEventRateLimitExceeded  AuditEventType = "security.rate_limit"
	AuditEventIPBlocked          AuditEventType = "security.ip_blocked"
	AuditEventFraudDetected      AuditEventType = "security.fraud"

	// Compliance events
	AuditEventDataAccessRequest AuditEventType = "compliance.data_access"
	AuditEventDataDeletion      AuditEventType = "compliance.data_deletion"
	AuditEventConsentUpdated    AuditEventType = "compliance.consent"
)

// AuditEvent represents an audit log entry.
type AuditEvent struct {
	// Unique event ID
	ID string `json:"id"`
	// Timestamp of the event
	Timestamp time.Time `json:"timestamp"`
	// Type of audit event
	Type AuditEventType `json:"type"`
	// Actor who performed the action
	Actor *AuditActor `json:"actor"`
	// Resource that was affected
	Resource *AuditResource `json:"resource,omitempty"`
	// Action performed
	Action string `json:"action"`
	// Outcome of the action
	Outcome AuditOutcome `json:"outcome"`
	// Additional details
	Details map[string]interface{} `json:"details,omitempty"`
	// Request context
	Request *AuditRequest `json:"request,omitempty"`
	// Service that generated the event
	Service string `json:"service"`
	// Environment (dev, staging, prod)
	Environment string `json:"environment"`
}

// AuditActor represents who performed the action.
type AuditActor struct {
	// Type of actor (user, driver, admin, system)
	Type string `json:"type"`
	// Actor ID
	ID string `json:"id"`
	// Actor name/email for display
	Name string `json:"name,omitempty"`
	// IP address
	IP string `json:"ip,omitempty"`
	// User agent
	UserAgent string `json:"user_agent,omitempty"`
	// Session ID
	SessionID string `json:"session_id,omitempty"`
}

// AuditResource represents the resource affected by the action.
type AuditResource struct {
	// Resource type (user, trip, payment, etc.)
	Type string `json:"type"`
	// Resource ID
	ID string `json:"id"`
	// Additional identifiers
	Identifiers map[string]string `json:"identifiers,omitempty"`
}

// AuditRequest represents the HTTP request context.
type AuditRequest struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	RequestID string `json:"request_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

// AuditOutcome represents the outcome of an action.
type AuditOutcome string

const (
	AuditOutcomeSuccess AuditOutcome = "success"
	AuditOutcomeFailure AuditOutcome = "failure"
	AuditOutcomeDenied  AuditOutcome = "denied"
)

// AuditLogger provides structured audit logging.
type AuditLogger struct {
	logger      *slog.Logger
	service     string
	environment string
}

// AuditLoggerConfig holds configuration for the audit logger.
type AuditLoggerConfig struct {
	ServiceName string
	Environment string
	Logger      *slog.Logger
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(config AuditLoggerConfig) *AuditLogger {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &AuditLogger{
		logger:      logger.With("audit", true),
		service:     config.ServiceName,
		environment: config.Environment,
	}
}

// Log logs an audit event.
func (l *AuditLogger) Log(ctx context.Context, event AuditEvent) {
	event.Service = l.service
	event.Environment = l.environment
	event.Timestamp = time.Now().UTC()

	if event.ID == "" {
		event.ID = generateEventID()
	}

	// Extract trace ID from context if available
	if event.Request != nil && event.Request.TraceID == "" {
		if traceID := TraceIDFromContext(ctx); traceID != "" {
			event.Request.TraceID = traceID
		}
	}

	// Convert to JSON for structured logging
	eventJSON, _ := json.Marshal(event)

	l.logger.LogAttrs(ctx, slog.LevelInfo, "audit_event",
		slog.String("event_type", string(event.Type)),
		slog.String("action", event.Action),
		slog.String("outcome", string(event.Outcome)),
		slog.String("event", string(eventJSON)),
	)
}

// LogAuth logs an authentication event.
func (l *AuditLogger) LogAuth(ctx context.Context, eventType AuditEventType, userID, userEmail, ip string, outcome AuditOutcome, details map[string]interface{}) {
	l.Log(ctx, AuditEvent{
		Type: eventType,
		Actor: &AuditActor{
			Type: "user",
			ID:   userID,
			Name: userEmail,
			IP:   ip,
		},
		Action:  string(eventType),
		Outcome: outcome,
		Details: details,
	})
}

// LogUserAction logs a user action event.
func (l *AuditLogger) LogUserAction(ctx context.Context, eventType AuditEventType, actor *AuditActor, resource *AuditResource, outcome AuditOutcome, details map[string]interface{}) {
	l.Log(ctx, AuditEvent{
		Type:     eventType,
		Actor:    actor,
		Resource: resource,
		Action:   string(eventType),
		Outcome:  outcome,
		Details:  details,
	})
}

// LogTripEvent logs a trip-related event.
func (l *AuditLogger) LogTripEvent(ctx context.Context, eventType AuditEventType, tripID, riderID, driverID string, outcome AuditOutcome, details map[string]interface{}) {
	l.Log(ctx, AuditEvent{
		Type: eventType,
		Actor: &AuditActor{
			Type: "user",
			ID:   riderID,
		},
		Resource: &AuditResource{
			Type: "trip",
			ID:   tripID,
			Identifiers: map[string]string{
				"rider_id":  riderID,
				"driver_id": driverID,
			},
		},
		Action:  string(eventType),
		Outcome: outcome,
		Details: details,
	})
}

// LogPaymentEvent logs a payment-related event.
func (l *AuditLogger) LogPaymentEvent(ctx context.Context, eventType AuditEventType, paymentID, tripID, userID string, amount float64, currency string, outcome AuditOutcome, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["amount"] = amount
	details["currency"] = currency

	l.Log(ctx, AuditEvent{
		Type: eventType,
		Actor: &AuditActor{
			Type: "user",
			ID:   userID,
		},
		Resource: &AuditResource{
			Type: "payment",
			ID:   paymentID,
			Identifiers: map[string]string{
				"trip_id": tripID,
			},
		},
		Action:  string(eventType),
		Outcome: outcome,
		Details: details,
	})
}

// LogAdminAction logs an admin action.
func (l *AuditLogger) LogAdminAction(ctx context.Context, adminID, adminEmail, action, targetType, targetID string, outcome AuditOutcome, details map[string]interface{}) {
	l.Log(ctx, AuditEvent{
		Type: AuditEventAdminAction,
		Actor: &AuditActor{
			Type: "admin",
			ID:   adminID,
			Name: adminEmail,
		},
		Resource: &AuditResource{
			Type: targetType,
			ID:   targetID,
		},
		Action:  action,
		Outcome: outcome,
		Details: details,
	})
}

// LogSecurityEvent logs a security-related event.
func (l *AuditLogger) LogSecurityEvent(ctx context.Context, eventType AuditEventType, ip, userAgent string, details map[string]interface{}) {
	l.Log(ctx, AuditEvent{
		Type: eventType,
		Actor: &AuditActor{
			Type:      "unknown",
			IP:        ip,
			UserAgent: userAgent,
		},
		Action:  string(eventType),
		Outcome: AuditOutcomeDenied,
		Details: details,
	})
}

// LogFromRequest logs an event with HTTP request context.
func (l *AuditLogger) LogFromRequest(ctx context.Context, r *http.Request, eventType AuditEventType, actor *AuditActor, resource *AuditResource, outcome AuditOutcome, details map[string]interface{}) {
	if actor.IP == "" {
		actor.IP = getClientIP(r)
	}
	if actor.UserAgent == "" {
		actor.UserAgent = r.UserAgent()
	}

	l.Log(ctx, AuditEvent{
		Type:     eventType,
		Actor:    actor,
		Resource: resource,
		Action:   string(eventType),
		Outcome:  outcome,
		Details:  details,
		Request: &AuditRequest{
			Method:    r.Method,
			Path:      r.URL.Path,
			RequestID: r.Header.Get("X-Request-ID"),
		},
	})
}

// Helper functions

func generateEventID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// TraceIDFromContext extracts trace ID from context.
// This should be implemented to integrate with your tracing package.
func TraceIDFromContext(ctx context.Context) string {
	// If using OpenTelemetry, extract trace ID
	// This is a placeholder - integrate with your telemetry package
	if traceID := ctx.Value("trace_id"); traceID != nil {
		return traceID.(string)
	}
	return ""
}

// AuditMiddleware creates an HTTP middleware that logs audit events for requests.
func AuditMiddleware(logger *AuditLogger, eventType AuditEventType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap response writer to capture status code
			wrapped := &auditResponseWriter{ResponseWriter: w, status: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log audit event based on response status
			outcome := AuditOutcomeSuccess
			if wrapped.status >= 400 {
				outcome = AuditOutcomeFailure
			}
			if wrapped.status == http.StatusForbidden || wrapped.status == http.StatusUnauthorized {
				outcome = AuditOutcomeDenied
			}

			// Extract user from context if available
			userID := ""
			if uid := r.Context().Value("user_id"); uid != nil {
				userID = uid.(string)
			}

			logger.LogFromRequest(r.Context(), r, eventType,
				&AuditActor{
					Type: "user",
					ID:   userID,
				},
				nil,
				outcome,
				map[string]interface{}{
					"status_code": wrapped.status,
				},
			)
		})
	}
}

type auditResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *auditResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
