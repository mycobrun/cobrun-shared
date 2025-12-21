// Package testing provides test utilities and helpers.
package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// TestContext creates a context with a timeout for testing.
func TestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// TestContextWithTimeout creates a context with a custom timeout.
func TestContextWithTimeout(t *testing.T, timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx
}

// HTTPTestRequest creates an HTTP request for testing.
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// NewHTTPTestRequest creates a new HTTP test request.
func NewHTTPTestRequest(method, path string) *HTTPTestRequest {
	return &HTTPTestRequest{
		Method:  method,
		Path:    path,
		Headers: make(map[string]string),
	}
}

// WithBody adds a JSON body to the request.
func (r *HTTPTestRequest) WithBody(body interface{}) *HTTPTestRequest {
	r.Body = body
	return r
}

// WithHeader adds a header to the request.
func (r *HTTPTestRequest) WithHeader(key, value string) *HTTPTestRequest {
	r.Headers[key] = value
	return r
}

// WithAuth adds an Authorization header with a Bearer token.
func (r *HTTPTestRequest) WithAuth(token string) *HTTPTestRequest {
	return r.WithHeader("Authorization", "Bearer "+token)
}

// WithContentType sets the Content-Type header.
func (r *HTTPTestRequest) WithContentType(contentType string) *HTTPTestRequest {
	return r.WithHeader("Content-Type", contentType)
}

// WithJSON sets Content-Type to application/json.
func (r *HTTPTestRequest) WithJSON() *HTTPTestRequest {
	return r.WithContentType("application/json")
}

// Build builds the HTTP request.
func (r *HTTPTestRequest) Build(t *testing.T) *http.Request {
	var body io.Reader
	if r.Body != nil {
		data, err := json.Marshal(r.Body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		body = bytes.NewReader(data)
	}

	req := httptest.NewRequest(r.Method, r.Path, body)
	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	if r.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}

// HTTPTestResponse wraps httptest.ResponseRecorder with helper methods.
type HTTPTestResponse struct {
	*httptest.ResponseRecorder
	t *testing.T
}

// NewHTTPTestResponse creates a new HTTP test response.
func NewHTTPTestResponse(t *testing.T) *HTTPTestResponse {
	return &HTTPTestResponse{
		ResponseRecorder: httptest.NewRecorder(),
		t:                t,
	}
}

// AssertStatus asserts the response status code.
func (r *HTTPTestResponse) AssertStatus(expected int) *HTTPTestResponse {
	if r.Code != expected {
		r.t.Errorf("expected status %d, got %d", expected, r.Code)
	}
	return r
}

// AssertOK asserts status 200.
func (r *HTTPTestResponse) AssertOK() *HTTPTestResponse {
	return r.AssertStatus(http.StatusOK)
}

// AssertCreated asserts status 201.
func (r *HTTPTestResponse) AssertCreated() *HTTPTestResponse {
	return r.AssertStatus(http.StatusCreated)
}

// AssertBadRequest asserts status 400.
func (r *HTTPTestResponse) AssertBadRequest() *HTTPTestResponse {
	return r.AssertStatus(http.StatusBadRequest)
}

// AssertUnauthorized asserts status 401.
func (r *HTTPTestResponse) AssertUnauthorized() *HTTPTestResponse {
	return r.AssertStatus(http.StatusUnauthorized)
}

// AssertForbidden asserts status 403.
func (r *HTTPTestResponse) AssertForbidden() *HTTPTestResponse {
	return r.AssertStatus(http.StatusForbidden)
}

// AssertNotFound asserts status 404.
func (r *HTTPTestResponse) AssertNotFound() *HTTPTestResponse {
	return r.AssertStatus(http.StatusNotFound)
}

// AssertConflict asserts status 409.
func (r *HTTPTestResponse) AssertConflict() *HTTPTestResponse {
	return r.AssertStatus(http.StatusConflict)
}

// AssertInternalError asserts status 500.
func (r *HTTPTestResponse) AssertInternalError() *HTTPTestResponse {
	return r.AssertStatus(http.StatusInternalServerError)
}

// DecodeJSON decodes the response body as JSON.
func (r *HTTPTestResponse) DecodeJSON(v interface{}) *HTTPTestResponse {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		r.t.Fatalf("failed to decode JSON: %v", err)
	}
	return r
}

// AssertJSONField asserts a JSON field has an expected value.
func (r *HTTPTestResponse) AssertJSONField(field string, expected interface{}) *HTTPTestResponse {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		r.t.Fatalf("failed to decode JSON: %v", err)
	}

	actual, ok := data[field]
	if !ok {
		r.t.Errorf("field %q not found in response", field)
		return r
	}

	if actual != expected {
		r.t.Errorf("field %q: expected %v, got %v", field, expected, actual)
	}

	return r
}

// TestRouter creates a chi router for testing.
func TestRouter() chi.Router {
	return chi.NewRouter()
}

// ExecuteRequest executes a request against a handler.
func ExecuteRequest(t *testing.T, handler http.Handler, req *http.Request) *HTTPTestResponse {
	resp := NewHTTPTestResponse(t)
	handler.ServeHTTP(resp, req)
	return resp
}

// MustJSON marshals to JSON or panics.
func MustJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// StringPtr returns a pointer to a string.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int.
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to a bool.
func BoolPtr(b bool) *bool {
	return &b
}

// TimePtr returns a pointer to a time.
func TimePtr(t time.Time) *time.Time {
	return &t
}

// Float64Ptr returns a pointer to a float64.
func Float64Ptr(f float64) *float64 {
	return &f
}
