// Package errors provides custom error types for the platform.
package errors

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *AppError
		wantSub string
	}{
		{
			name:    "without wrapped error",
			err:     New(CodeBadRequest, "invalid input"),
			wantSub: "BAD_REQUEST: invalid input",
		},
		{
			name:    "with wrapped error",
			err:     Wrap(errors.New("underlying error"), CodeInternal, "something failed"),
			wantSub: "INTERNAL_ERROR: something failed: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantSub {
				t.Errorf("Error() = %v, want %v", got, tt.wantSub)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	appErr := Wrap(underlying, CodeInternal, "wrapped")

	if appErr.Unwrap() != underlying {
		t.Error("Unwrap() should return underlying error")
	}

	// AppError without wrapped error
	appErr2 := New(CodeBadRequest, "no wrap")
	if appErr2.Unwrap() != nil {
		t.Error("Unwrap() should return nil for unwrapped error")
	}
}

func TestAppError_Is(t *testing.T) {
	err1 := New(CodeNotFound, "resource not found")
	err2 := New(CodeNotFound, "different message")
	err3 := New(CodeBadRequest, "bad request")

	if !err1.Is(err2) {
		t.Error("errors with same code should match")
	}

	if err1.Is(err3) {
		t.Error("errors with different code should not match")
	}

	if err1.Is(errors.New("not app error")) {
		t.Error("AppError should not match non-AppError")
	}
}

func TestAppError_WithDetails(t *testing.T) {
	err := New(CodeValidation, "validation failed")
	details := map[string]string{
		"email": "invalid format",
		"name":  "required",
	}

	err = err.WithDetails(details)

	if err.Details == nil {
		t.Fatal("Details should not be nil")
	}

	if err.Details["email"] != "invalid format" {
		t.Errorf("Details[email] = %s, want 'invalid format'", err.Details["email"])
	}

	if err.Details["name"] != "required" {
		t.Errorf("Details[name] = %s, want 'required'", err.Details["name"])
	}
}

func TestNew(t *testing.T) {
	err := New(CodeBadRequest, "test message")

	if err.Code != CodeBadRequest {
		t.Errorf("Code = %s, want %s", err.Code, CodeBadRequest)
	}

	if err.Message != "test message" {
		t.Errorf("Message = %s, want 'test message'", err.Message)
	}

	if err.Err != nil {
		t.Error("Err should be nil")
	}
}

func TestWrap(t *testing.T) {
	underlying := errors.New("underlying")
	err := Wrap(underlying, CodeInternal, "wrapped message")

	if err.Code != CodeInternal {
		t.Errorf("Code = %s, want %s", err.Code, CodeInternal)
	}

	if err.Message != "wrapped message" {
		t.Errorf("Message = %s, want 'wrapped message'", err.Message)
	}

	if err.Err != underlying {
		t.Error("Err should be the underlying error")
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		wantCode string
	}{
		{"Internal", Internal("internal error"), CodeInternal},
		{"NotFound", NotFound("user"), CodeNotFound},
		{"BadRequest", BadRequest("bad input"), CodeBadRequest},
		{"Validation", Validation("invalid"), CodeValidation},
		{"Unauthorized", Unauthorized(""), CodeUnauthorized},
		{"Forbidden", Forbidden(""), CodeForbidden},
		{"Conflict", Conflict("duplicate"), CodeConflict},
		{"Timeout", Timeout("request timed out"), CodeTimeout},
		{"Unavailable", Unavailable("service down"), CodeUnavailable},
		{"RateLimited", RateLimited("too many requests"), CodeRateLimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %s, want %s", tt.err.Code, tt.wantCode)
			}
		})
	}
}

func TestInternalWrap(t *testing.T) {
	underlying := errors.New("db connection failed")
	err := InternalWrap(underlying, "database error")

	if err.Code != CodeInternal {
		t.Errorf("Code = %s, want %s", err.Code, CodeInternal)
	}

	if err.Err != underlying {
		t.Error("should wrap underlying error")
	}
}

func TestNotFound_Message(t *testing.T) {
	err := NotFound("user")
	if err.Message != "user not found" {
		t.Errorf("Message = %s, want 'user not found'", err.Message)
	}
}

func TestUnauthorized_DefaultMessage(t *testing.T) {
	err := Unauthorized("")
	if err.Message != "authentication required" {
		t.Errorf("Message = %s, want 'authentication required'", err.Message)
	}
}

func TestForbidden_DefaultMessage(t *testing.T) {
	err := Forbidden("")
	if err.Message != "access denied" {
		t.Errorf("Message = %s, want 'access denied'", err.Message)
	}
}

func TestValidationWithDetails(t *testing.T) {
	details := map[string]string{"field": "error"}
	err := ValidationWithDetails("validation failed", details)

	if err.Code != CodeValidation {
		t.Errorf("Code = %s, want %s", err.Code, CodeValidation)
	}

	if err.Details["field"] != "error" {
		t.Errorf("Details[field] = %s, want 'error'", err.Details["field"])
	}
}

func TestIsNotFound(t *testing.T) {
	notFoundErr := NotFound("resource")
	otherErr := BadRequest("bad")
	stdErr := errors.New("standard error")

	if !IsNotFound(notFoundErr) {
		t.Error("IsNotFound should return true for NotFound error")
	}

	if IsNotFound(otherErr) {
		t.Error("IsNotFound should return false for other AppError")
	}

	if IsNotFound(stdErr) {
		t.Error("IsNotFound should return false for standard error")
	}
}

func TestIsValidation(t *testing.T) {
	validationErr := Validation("invalid")
	otherErr := BadRequest("bad")
	stdErr := errors.New("standard error")

	if !IsValidation(validationErr) {
		t.Error("IsValidation should return true for Validation error")
	}

	if IsValidation(otherErr) {
		t.Error("IsValidation should return false for other AppError")
	}

	if IsValidation(stdErr) {
		t.Error("IsValidation should return false for standard error")
	}
}

func TestIsUnauthorized(t *testing.T) {
	unauthorizedErr := Unauthorized("not logged in")
	otherErr := BadRequest("bad")
	stdErr := errors.New("standard error")

	if !IsUnauthorized(unauthorizedErr) {
		t.Error("IsUnauthorized should return true for Unauthorized error")
	}

	if IsUnauthorized(otherErr) {
		t.Error("IsUnauthorized should return false for other AppError")
	}

	if IsUnauthorized(stdErr) {
		t.Error("IsUnauthorized should return false for standard error")
	}
}

func TestCode(t *testing.T) {
	appErr := NotFound("resource")
	stdErr := errors.New("standard error")

	code := Code(appErr)
	if code != CodeNotFound {
		t.Errorf("Code() = %s, want %s", code, CodeNotFound)
	}

	code = Code(stdErr)
	if code != "" {
		t.Errorf("Code() = %s, want empty string", code)
	}

	code = Code(nil)
	if code != "" {
		t.Errorf("Code(nil) = %s, want empty string", code)
	}
}

func TestErrorCodes(t *testing.T) {
	// Verify error code constants
	tests := []struct {
		code     string
		expected string
	}{
		{CodeInternal, "INTERNAL_ERROR"},
		{CodeNotFound, "NOT_FOUND"},
		{CodeBadRequest, "BAD_REQUEST"},
		{CodeUnauthorized, "UNAUTHORIZED"},
		{CodeForbidden, "FORBIDDEN"},
		{CodeConflict, "CONFLICT"},
		{CodeValidation, "VALIDATION_ERROR"},
		{CodeTimeout, "TIMEOUT"},
		{CodeUnavailable, "SERVICE_UNAVAILABLE"},
		{CodeRateLimited, "RATE_LIMITED"},
	}

	for _, tt := range tests {
		if tt.code != tt.expected {
			t.Errorf("code = %s, want %s", tt.code, tt.expected)
		}
	}
}

func TestAppError_ErrorsAs(t *testing.T) {
	appErr := NotFound("user")

	var target *AppError
	if !errors.As(appErr, &target) {
		t.Error("errors.As should work with AppError")
	}

	if target.Code != CodeNotFound {
		t.Errorf("Code = %s, want %s", target.Code, CodeNotFound)
	}
}
