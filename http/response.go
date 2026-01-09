// Package http provides standardized HTTP response utilities.
package http

import (
	"encoding/json"
	"net/http"
)

// Response is a standard API response wrapper.
// Per Section 9 of technical design:
// {
//   "data": <payload>,
//   "meta": {"correlationId": "...", "nextCursor": "..."},
//   "error": null
// }
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
	Error   interface{} `json:"error,omitempty"` // null for success responses
}

// APIResponse is the standard API response envelope per Section 9.
type APIResponse struct {
	Data  interface{} `json:"data"`
	Meta  *APIMeta    `json:"meta,omitempty"`
	Error *APIError   `json:"error,omitempty"`
}

// APIMeta contains response metadata per Section 9.
type APIMeta struct {
	CorrelationID string `json:"correlationId,omitempty"`
	NextCursor    string `json:"nextCursor,omitempty"`
	Page          int    `json:"page,omitempty"`
	PerPage       int    `json:"per_page,omitempty"`
	Total         int64  `json:"total,omitempty"`
	TotalPages    int    `json:"total_pages,omitempty"`
}

// APIError contains error details per Section 9.
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Meta contains response metadata (legacy, for backward compatibility).
type Meta struct {
	Page       int   `json:"page,omitempty"`
	PerPage    int   `json:"per_page,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

// JSON sends a JSON response.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// Log error but don't write another response
			return
		}
	}
}

// OK sends a 200 OK response with data.
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// OKWithMeta sends a 200 OK response with data and metadata.
func OKWithMeta(w http.ResponseWriter, data interface{}, meta *Meta) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Created sends a 201 Created response.
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// NoContent sends a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Accepted sends a 202 Accepted response.
func Accepted(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusAccepted, Response{
		Success: true,
		Data:    data,
	})
}

// PaginatedResponse creates a paginated response.
func PaginatedResponse(data interface{}, page, perPage int, total int64) Response {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return Response{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// Paginated sends a paginated response.
func Paginated(w http.ResponseWriter, data interface{}, page, perPage int, total int64) {
	JSON(w, http.StatusOK, PaginatedResponse(data, page, perPage, total))
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, ErrorResponse{Success: false, Error: message})
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(w http.ResponseWriter, message string) {
	JSON(w, http.StatusUnauthorized, ErrorResponse{Success: false, Error: message})
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(w http.ResponseWriter, message string) {
	JSON(w, http.StatusForbidden, ErrorResponse{Success: false, Error: message})
}

// NotFound sends a 404 Not Found response.
func NotFound(w http.ResponseWriter, message string) {
	JSON(w, http.StatusNotFound, ErrorResponse{Success: false, Error: message})
}

// Conflict sends a 409 Conflict response.
func Conflict(w http.ResponseWriter, message string) {
	JSON(w, http.StatusConflict, ErrorResponse{Success: false, Error: message})
}

// TooManyRequests sends a 429 Too Many Requests response.
func TooManyRequests(w http.ResponseWriter, message string) {
	JSON(w, http.StatusTooManyRequests, ErrorResponse{Success: false, Error: message})
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(w http.ResponseWriter, message string) {
	JSON(w, http.StatusInternalServerError, ErrorResponse{Success: false, Error: message})
}

// ParseInt parses a string to an int.
func ParseInt(s string, v *int) (bool, error) {
	var val int
	err := json.Unmarshal([]byte(s), &val)
	if err != nil {
		return false, err
	}
	*v = val
	return true, nil
}

// === Standard API Response Methods (Section 9) ===

// APISuccess sends a standard API success response.
func APISuccess(w http.ResponseWriter, status int, data interface{}, correlationID string) {
	var meta *APIMeta
	if correlationID != "" {
		meta = &APIMeta{CorrelationID: correlationID}
	}
	JSON(w, status, APIResponse{
		Data:  data,
		Meta:  meta,
		Error: nil,
	})
}

// APIPaginated sends a paginated API response with cursor.
func APIPaginated(w http.ResponseWriter, data interface{}, correlationID, nextCursor string, page, perPage int, total int64) {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	JSON(w, http.StatusOK, APIResponse{
		Data: data,
		Meta: &APIMeta{
			CorrelationID: correlationID,
			NextCursor:    nextCursor,
			Page:          page,
			PerPage:       perPage,
			Total:         total,
			TotalPages:    totalPages,
		},
		Error: nil,
	})
}

// APIError sends a standard API error response.
func APIErrorResponse(w http.ResponseWriter, status int, code, message, correlationID string, details map[string]interface{}) {
	var meta *APIMeta
	if correlationID != "" {
		meta = &APIMeta{CorrelationID: correlationID}
	}
	JSON(w, status, APIResponse{
		Data: nil,
		Meta: meta,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// Common error codes per Section 9
const (
	ErrCodeBadRequest        = "BAD_REQUEST"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeConflict          = "CONFLICT"
	ErrCodeTooManyRequests   = "TOO_MANY_REQUESTS"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeDriverSuspended   = "DRIVER_SUSPENDED"
	ErrCodeDriverNotApproved = "DRIVER_NOT_APPROVED"
	ErrCodeDocumentExpired   = "DOCUMENT_EXPIRED"
	ErrCodeBGCheckFailed     = "BACKGROUND_CHECK_FAILED"
)
