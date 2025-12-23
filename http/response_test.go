// Package http provides standardized HTTP response utilities.
package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		data         interface{}
		expectStatus int
		expectBody   bool
	}{
		{
			name:         "sends JSON with data",
			status:       http.StatusOK,
			data:         map[string]string{"message": "hello"},
			expectStatus: http.StatusOK,
			expectBody:   true,
		},
		{
			name:         "sends JSON without data",
			status:       http.StatusNoContent,
			data:         nil,
			expectStatus: http.StatusNoContent,
			expectBody:   false,
		},
		{
			name:         "sends JSON with array",
			status:       http.StatusOK,
			data:         []string{"item1", "item2"},
			expectStatus: http.StatusOK,
			expectBody:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			JSON(w, tt.status, tt.data)

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", ct)
			}

			if tt.expectBody {
				if w.Body.Len() == 0 {
					t.Error("expected response body, got empty")
				}

				// Verify JSON is valid
				var result interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Errorf("response body is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestOK(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "success"}
	OK(w, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("expected success to be true")
	}

	if response.Data == nil {
		t.Error("expected data to be present")
	}
}

func TestOKWithMeta(t *testing.T) {
	w := httptest.NewRecorder()
	data := []string{"item1", "item2"}
	meta := &Meta{
		Page:       1,
		PerPage:    10,
		Total:      100,
		TotalPages: 10,
	}

	OKWithMeta(w, data, meta)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("expected success to be true")
	}

	if response.Meta == nil {
		t.Fatal("expected meta to be present")
	}

	if response.Meta.Page != 1 {
		t.Errorf("expected page 1, got %d", response.Meta.Page)
	}

	if response.Meta.Total != 100 {
		t.Errorf("expected total 100, got %d", response.Meta.Total)
	}
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"id": "123"}
	Created(w, data)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var response Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("expected success to be true")
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	NoContent(w)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	if w.Body.Len() != 0 {
		t.Error("expected empty body for 204 No Content")
	}
}

func TestAccepted(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"job_id": "456"}
	Accepted(w, data)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	var response Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("expected success to be true")
	}
}

func TestPaginatedResponse(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		perPage        int
		total          int64
		expectPages    int
	}{
		{
			name:        "calculates pages correctly",
			page:        1,
			perPage:     10,
			total:       25,
			expectPages: 3,
		},
		{
			name:        "handles exact division",
			page:        1,
			perPage:     10,
			total:       20,
			expectPages: 2,
		},
		{
			name:        "handles single page",
			page:        1,
			perPage:     10,
			total:       5,
			expectPages: 1,
		},
		{
			name:        "handles zero total",
			page:        1,
			perPage:     10,
			total:       0,
			expectPages: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []string{"item1", "item2"}
			response := PaginatedResponse(data, tt.page, tt.perPage, tt.total)

			if !response.Success {
				t.Error("expected success to be true")
			}

			if response.Meta == nil {
				t.Fatal("expected meta to be present")
			}

			if response.Meta.Page != tt.page {
				t.Errorf("expected page %d, got %d", tt.page, response.Meta.Page)
			}

			if response.Meta.PerPage != tt.perPage {
				t.Errorf("expected perPage %d, got %d", tt.perPage, response.Meta.PerPage)
			}

			if response.Meta.Total != tt.total {
				t.Errorf("expected total %d, got %d", tt.total, response.Meta.Total)
			}

			if response.Meta.TotalPages != tt.expectPages {
				t.Errorf("expected totalPages %d, got %d", tt.expectPages, response.Meta.TotalPages)
			}
		})
	}
}

func TestPaginated(t *testing.T) {
	w := httptest.NewRecorder()
	data := []string{"item1", "item2", "item3"}
	Paginated(w, data, 2, 10, 50)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("expected success to be true")
	}

	if response.Meta == nil {
		t.Fatal("expected meta to be present")
	}

	if response.Meta.Page != 2 {
		t.Errorf("expected page 2, got %d", response.Meta.Page)
	}

	if response.Meta.Total != 50 {
		t.Errorf("expected total 50, got %d", response.Meta.Total)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "Invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("expected success to be false")
	}

	if response.Error != "Invalid input" {
		t.Errorf("expected error 'Invalid input', got '%s'", response.Error)
	}
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Unauthorized(w, "Authentication required")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("expected success to be false")
	}

	if response.Error == "" {
		t.Error("expected error message")
	}
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Forbidden(w, "Access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("expected success to be false")
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "Resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("expected success to be false")
	}
}

func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	Conflict(w, "Resource already exists")

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("expected success to be false")
	}
}

func TestTooManyRequests(t *testing.T) {
	w := httptest.NewRecorder()
	TooManyRequests(w, "Rate limit exceeded")

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("expected success to be false")
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalError(w, "Internal server error")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("expected success to be false")
	}
}

func TestResponseStructure(t *testing.T) {
	// Test that Response struct marshals correctly
	resp := Response{
		Success: true,
		Data:    map[string]string{"key": "value"},
		Meta: &Meta{
			Page:    1,
			PerPage: 10,
			Total:   100,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var unmarshaled Response
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !unmarshaled.Success {
		t.Error("expected success to be true")
	}

	if unmarshaled.Meta == nil {
		t.Error("expected meta to be present")
	}
}

func TestErrorResponseStructure(t *testing.T) {
	// Test that ErrorResponse struct marshals correctly
	errResp := ErrorResponse{
		Success: false,
		Error:   "test error",
	}

	data, err := json.Marshal(errResp)
	if err != nil {
		t.Fatalf("failed to marshal error response: %v", err)
	}

	var unmarshaled ErrorResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if unmarshaled.Success {
		t.Error("expected success to be false")
	}

	if unmarshaled.Error != "test error" {
		t.Errorf("expected error 'test error', got '%s'", unmarshaled.Error)
	}
}

func TestJSONWithComplexData(t *testing.T) {
	w := httptest.NewRecorder()

	type ComplexData struct {
		ID        int                    `json:"id"`
		Name      string                 `json:"name"`
		Tags      []string               `json:"tags"`
		Metadata  map[string]interface{} `json:"metadata"`
	}

	data := ComplexData{
		ID:   1,
		Name: "test",
		Tags: []string{"tag1", "tag2"},
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	}

	JSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result ComplexData
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal complex data: %v", err)
	}

	if result.ID != data.ID {
		t.Errorf("expected ID %d, got %d", data.ID, result.ID)
	}

	if len(result.Tags) != len(data.Tags) {
		t.Errorf("expected %d tags, got %d", len(data.Tags), len(result.Tags))
	}
}

func TestPaginationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		perPage     int
		total       int64
		expectPages int
	}{
		{
			name:        "large numbers",
			perPage:     100,
			total:       10000,
			expectPages: 100,
		},
		{
			name:        "one item per page",
			perPage:     1,
			total:       5,
			expectPages: 5,
		},
		{
			name:        "more perPage than total",
			perPage:     100,
			total:       10,
			expectPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := PaginatedResponse([]string{}, 1, tt.perPage, tt.total)

			if resp.Meta.TotalPages != tt.expectPages {
				t.Errorf("expected %d pages, got %d", tt.expectPages, resp.Meta.TotalPages)
			}
		})
	}
}
