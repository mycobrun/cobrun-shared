// Package http provides common HTTP middlewares.
package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mycobrun/cobrun-shared/logging"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name           string
		existingID     string
		expectGenerate bool
	}{
		{
			name:           "generates new ID when none exists",
			existingID:     "",
			expectGenerate: true,
		},
		{
			name:           "uses existing ID from header",
			existingID:     "test-request-id-123",
			expectGenerate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.existingID != "" {
				req.Header.Set("X-Request-ID", tt.existingID)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			requestID := w.Header().Get("X-Request-ID")
			if requestID == "" {
				t.Error("X-Request-ID header should be set")
			}

			if tt.expectGenerate {
				if requestID == tt.existingID {
					t.Error("should generate new request ID")
				}
			} else {
				if requestID != tt.existingID {
					t.Errorf("expected request ID %s, got %s", tt.existingID, requestID)
				}
			}
		})
	}
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name            string
		allowedOrigins  []string
		requestOrigin   string
		method          string
		expectOrigin    string
		expectStatus    int
		expectMethods   bool
		expectHeaders   bool
		expectMaxAge    bool
	}{
		{
			name:           "allows wildcard origin",
			allowedOrigins: []string{"*"},
			requestOrigin:  "https://example.com",
			method:         "GET",
			expectOrigin:   "https://example.com",
			expectStatus:   http.StatusOK,
			expectMethods:  true,
			expectHeaders:  true,
			expectMaxAge:   true,
		},
		{
			name:           "allows specific origin",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://example.com",
			method:         "POST",
			expectOrigin:   "https://example.com",
			expectStatus:   http.StatusOK,
			expectMethods:  true,
			expectHeaders:  true,
			expectMaxAge:   true,
		},
		{
			name:           "blocks non-allowed origin",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://evil.com",
			method:         "GET",
			expectOrigin:   "",
			expectStatus:   http.StatusOK,
			expectMethods:  true,
			expectHeaders:  true,
			expectMaxAge:   true,
		},
		{
			name:           "handles OPTIONS preflight",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://example.com",
			method:         "OPTIONS",
			expectOrigin:   "https://example.com",
			expectStatus:   http.StatusOK,
			expectMethods:  true,
			expectHeaders:  true,
			expectMaxAge:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			handler := CORS(tt.allowedOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}

			origin := w.Header().Get("Access-Control-Allow-Origin")
			if origin != tt.expectOrigin {
				t.Errorf("expected origin %s, got %s", tt.expectOrigin, origin)
			}

			if tt.expectMethods {
				methods := w.Header().Get("Access-Control-Allow-Methods")
				if methods == "" {
					t.Error("Access-Control-Allow-Methods header should be set")
				}
			}

			if tt.expectHeaders {
				headers := w.Header().Get("Access-Control-Allow-Headers")
				if headers == "" {
					t.Error("Access-Control-Allow-Headers header should be set")
				}
			}

			if tt.expectMaxAge {
				maxAge := w.Header().Get("Access-Control-Max-Age")
				if maxAge == "" {
					t.Error("Access-Control-Max-Age header should be set")
				}
			}

			// For OPTIONS, next handler should not be called
			if tt.method == "OPTIONS" && nextCalled {
				t.Error("next handler should not be called for OPTIONS")
			}

			// For other methods, next handler should be called
			if tt.method != "OPTIONS" && !nextCalled {
				t.Error("next handler should be called for non-OPTIONS")
			}
		})
	}
}

func TestLogger(t *testing.T) {
	logger := logging.NewLogger("info")

	handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))

	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	req.Header.Set("User-Agent", "test-agent")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Logger middleware should not cause errors
	// Note: We can't easily capture log output without changing logger implementation
}

func TestRecoverer(t *testing.T) {
	logger := logging.NewLogger("info")

	tests := []struct {
		name         string
		handler      http.HandlerFunc
		expectPanic  bool
		expectStatus int
	}{
		{
			name: "recovers from panic",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			}),
			expectPanic:  true,
			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "normal execution",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			expectPanic:  false,
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Recoverer(logger)(tt.handler)

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}
		})
	}
}

func TestContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{
			name:        "sets application/json",
			contentType: "application/json",
		},
		{
			name:        "sets text/html",
			contentType: "text/html",
		},
		{
			name:        "sets text/plain",
			contentType: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ContentType(tt.contentType)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if ct := w.Header().Get("Content-Type"); ct != tt.contentType {
				t.Errorf("expected Content-Type %s, got %s", tt.contentType, ct)
			}
		})
	}
}

func TestJSONContentType(t *testing.T) {
	handler := JSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		if got := w.Header().Get(tt.header); got != tt.expected {
			t.Errorf("expected %s: %s, got %s", tt.header, tt.expected, got)
		}
	}
}

func TestTimeout(t *testing.T) {
	// Test that Timeout middleware wraps handlers correctly
	// Note: httptest.ResponseRecorder doesn't support true timeout behavior,
	// so we only test that the middleware is applied correctly
	handler := Timeout(100 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler completes within timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	if w.Body.String() != "success" {
		t.Errorf("expected 'success', got %q", w.Body.String())
	}
}

func TestCompress(t *testing.T) {
	handler := Compress(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response data that should be compressed if requested"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// When compression is enabled and client accepts gzip, content-encoding should be set
	// Note: chi's Compress middleware may not set encoding for small responses
	// This is just a smoke test to ensure middleware doesn't break
}

func TestMiddlewareChaining(t *testing.T) {
	logger := logging.NewLogger("info")

	// Chain multiple middlewares
	handler := SecurityHeaders(
		JSONContentType(
			RequestID(
				Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"message": "success"}`))
				})),
			),
		),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check that all middlewares applied their effects
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	if reqID := w.Header().Get("X-Request-ID"); reqID == "" {
		t.Error("X-Request-ID should be set")
	}

	if xfo := w.Header().Get("X-Frame-Options"); xfo != "DENY" {
		t.Error("X-Frame-Options should be set to DENY")
	}
}

func TestRealIP(t *testing.T) {
	handler := RealIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// RealIP middleware should modify RemoteAddr based on headers
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestCORSMultipleOrigins(t *testing.T) {
	allowedOrigins := []string{"https://app1.com", "https://app2.com"}
	handler := CORS(allowedOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		origin       string
		expectAllow  bool
	}{
		{"https://app1.com", true},
		{"https://app2.com", true},
		{"https://evil.com", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", tt.origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		allowedOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if tt.expectAllow {
			if allowedOrigin != tt.origin {
				t.Errorf("expected origin %s to be allowed, got %s", tt.origin, allowedOrigin)
			}
		} else {
			if allowedOrigin != "" {
				t.Errorf("origin %s should not be allowed", tt.origin)
			}
		}
	}
}

func TestLoggerWithRequestID(t *testing.T) {
	logger := logging.NewLogger("info")

	handler := Logger(logger)(RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Request ID should be in response headers
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("X-Request-ID should be set")
	}
}

func TestRecovererWithContext(t *testing.T) {
	logger := logging.NewLogger("error") // Use minimal logging for tests
	handler := Recoverer(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Simulate panic during context processing
		if ctx.Value("panic") != nil {
			panic("context panic")
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Test with panic
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), "panic", true))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	// Test without panic
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
