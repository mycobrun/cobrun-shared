// Package errors provides custom error types for the platform.
package errors

import (
	"errors"
	"fmt"
)

// Standard error codes.
const (
	CodeInternal       = "INTERNAL_ERROR"
	CodeNotFound       = "NOT_FOUND"
	CodeBadRequest     = "BAD_REQUEST"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeConflict       = "CONFLICT"
	CodeValidation     = "VALIDATION_ERROR"
	CodeTimeout        = "TIMEOUT"
	CodeUnavailable    = "SERVICE_UNAVAILABLE"
	CodeRateLimited    = "RATE_LIMITED"
)

// AppError represents an application error with code and message.
type AppError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	Err     error             `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches another error.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithDetails adds details to the error.
func (e *AppError) WithDetails(details map[string]string) *AppError {
	e.Details = details
	return e
}

// Wrap wraps an error with an AppError.
func Wrap(err error, code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// New creates a new AppError.
func New(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Common error constructors.

// Internal creates an internal server error.
func Internal(message string) *AppError {
	return New(CodeInternal, message)
}

// InternalWrap wraps an error as an internal error.
func InternalWrap(err error, message string) *AppError {
	return Wrap(err, CodeInternal, message)
}

// NotFound creates a not found error.
func NotFound(resource string) *AppError {
	return New(CodeNotFound, fmt.Sprintf("%s not found", resource))
}

// BadRequest creates a bad request error.
func BadRequest(message string) *AppError {
	return New(CodeBadRequest, message)
}

// Validation creates a validation error.
func Validation(message string) *AppError {
	return New(CodeValidation, message)
}

// ValidationWithDetails creates a validation error with field details.
func ValidationWithDetails(message string, details map[string]string) *AppError {
	return New(CodeValidation, message).WithDetails(details)
}

// Unauthorized creates an unauthorized error.
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "authentication required"
	}
	return New(CodeUnauthorized, message)
}

// Forbidden creates a forbidden error.
func Forbidden(message string) *AppError {
	if message == "" {
		message = "access denied"
	}
	return New(CodeForbidden, message)
}

// Conflict creates a conflict error.
func Conflict(message string) *AppError {
	return New(CodeConflict, message)
}

// Timeout creates a timeout error.
func Timeout(message string) *AppError {
	return New(CodeTimeout, message)
}

// Unavailable creates a service unavailable error.
func Unavailable(message string) *AppError {
	return New(CodeUnavailable, message)
}

// RateLimited creates a rate limited error.
func RateLimited(message string) *AppError {
	return New(CodeRateLimited, message)
}

// IsNotFound checks if the error is a not found error.
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeNotFound
	}
	return false
}

// IsValidation checks if the error is a validation error.
func IsValidation(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeValidation
	}
	return false
}

// IsUnauthorized checks if the error is an unauthorized error.
func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeUnauthorized
	}
	return false
}

// Code returns the error code or empty string.
func Code(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ""
}
