// Package errors provides HTTP error response utilities.
package errors

import (
	"encoding/json"
	"net/http"
)

// HTTPError maps error codes to HTTP status codes.
var httpStatusMap = map[string]int{
	CodeInternal:       http.StatusInternalServerError,
	CodeNotFound:       http.StatusNotFound,
	CodeBadRequest:     http.StatusBadRequest,
	CodeUnauthorized:   http.StatusUnauthorized,
	CodeForbidden:      http.StatusForbidden,
	CodeConflict:       http.StatusConflict,
	CodeValidation:     http.StatusBadRequest,
	CodeTimeout:        http.StatusGatewayTimeout,
	CodeUnavailable:    http.StatusServiceUnavailable,
	CodeRateLimited:    http.StatusTooManyRequests,
}

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error   ErrorBody `json:"error"`
	TraceID string    `json:"trace_id,omitempty"`
}

// ErrorBody contains the error details.
type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// HTTPStatus returns the HTTP status code for an error.
func HTTPStatus(err error) int {
	if appErr, ok := err.(*AppError); ok {
		if status, exists := httpStatusMap[appErr.Code]; exists {
			return status
		}
	}
	return http.StatusInternalServerError
}

// WriteError writes an error response to the HTTP response writer.
func WriteError(w http.ResponseWriter, err error, traceID string) {
	status := HTTPStatus(err)

	var code, message string
	var details map[string]string

	if appErr, ok := err.(*AppError); ok {
		code = appErr.Code
		message = appErr.Message
		details = appErr.Details
	} else {
		code = CodeInternal
		message = "An internal error occurred"
	}

	response := ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		TraceID: traceID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// WriteErrorWithStatus writes an error response with a specific status code.
func WriteErrorWithStatus(w http.ResponseWriter, status int, code, message string) {
	response := ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}
