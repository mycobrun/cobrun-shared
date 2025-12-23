// Package validation provides input validation utilities.
package validation

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"valid US phone", "+14155551234", false},
		{"valid UK phone", "+447911123456", false},
		{"missing plus", "14155551234", true},
		{"too short", "+1", true},
		{"too long", "+123456789012345678", true},
		{"invalid chars", "+1415abc1234", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVar(tt.phone, "phone")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVar(%q, 'phone') error = %v, wantErr %v", tt.phone, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLatitude(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		wantErr bool
	}{
		{"valid positive", 37.7749, false},
		{"valid negative", -33.8688, false},
		{"zero", 0, false},
		{"max", 90, false},
		{"min", -90, false},
		{"too high", 91, true},
		{"too low", -91, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVar(tt.lat, "latitude")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVar(%v, 'latitude') error = %v, wantErr %v", tt.lat, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLongitude(t *testing.T) {
	tests := []struct {
		name    string
		lng     float64
		wantErr bool
	}{
		{"valid positive", 122.4194, false},
		{"valid negative", -122.4194, false},
		{"zero", 0, false},
		{"max", 180, false},
		{"min", -180, false},
		{"too high", 181, true},
		{"too low", -181, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVar(tt.lng, "longitude")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVar(%v, 'longitude') error = %v, wantErr %v", tt.lng, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUserType(t *testing.T) {
	tests := []struct {
		name     string
		userType string
		wantErr  bool
	}{
		{"rider", "rider", false},
		{"driver", "driver", false},
		{"admin", "admin", false},
		{"invalid", "superuser", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVar(tt.userType, "user_type")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVar(%q, 'user_type') error = %v, wantErr %v", tt.userType, err, tt.wantErr)
			}
		})
	}
}

func TestValidateStruct(t *testing.T) {
	type TestRequest struct {
		Email    string  `json:"email" validate:"required,email"`
		Phone    string  `json:"phone" validate:"required,phone"`
		Lat      float64 `json:"lat" validate:"required,latitude"`
		Lng      float64 `json:"lng" validate:"required,longitude"`
		UserType string  `json:"user_type" validate:"required,user_type"`
	}

	tests := []struct {
		name    string
		req     TestRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: TestRequest{
				Email:    "test@example.com",
				Phone:    "+14155551234",
				Lat:      37.7749,
				Lng:      -122.4194,
				UserType: "rider",
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			req: TestRequest{
				Email:    "notanemail",
				Phone:    "+14155551234",
				Lat:      37.7749,
				Lng:      -122.4194,
				UserType: "rider",
			},
			wantErr: true,
		},
		{
			name: "invalid phone",
			req: TestRequest{
				Email:    "test@example.com",
				Phone:    "1234567890",
				Lat:      37.7749,
				Lng:      -122.4194,
				UserType: "rider",
			},
			wantErr: true,
		},
		{
			name: "missing required fields",
			req:  TestRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, err := ValidateStruct(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && len(errors) == 0 {
				t.Error("ValidateStruct() expected validation errors but got none")
			}
		})
	}
}

func TestDecodeAndValidate(t *testing.T) {
	type TestRequest struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	tests := []struct {
		name       string
		body       string
		contentType string
		wantOK     bool
		wantStatus int
	}{
		{
			name:        "valid request",
			body:        `{"name": "John", "email": "john@example.com"}`,
			contentType: "application/json",
			wantOK:      true,
			wantStatus:  0,
		},
		{
			name:        "invalid json",
			body:        `{"name": invalid}`,
			contentType: "application/json",
			wantOK:      false,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "validation error",
			body:        `{"name": "", "email": "notanemail"}`,
			contentType: "application/json",
			wantOK:      false,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "wrong content type",
			body:        `name=John`,
			contentType: "application/x-www-form-urlencoded",
			wantOK:      false,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			var testReq TestRequest
			ok := DecodeAndValidate(w, req, &testReq)

			if ok != tt.wantOK {
				t.Errorf("DecodeAndValidate() = %v, want %v", ok, tt.wantOK)
			}

			if !ok && w.Code != tt.wantStatus {
				t.Errorf("DecodeAndValidate() status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestParseValidationErrors(t *testing.T) {
	type TestStruct struct {
		Email string `json:"email" validate:"required,email"`
	}

	err := Validate(TestStruct{Email: "invalid"})
	if err == nil {
		t.Fatal("expected validation error")
	}

	errors := ParseValidationErrors(err)
	if len(errors) == 0 {
		t.Error("expected at least one validation error")
	}

	found := false
	for _, e := range errors {
		if e.Field == "email" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error for email field")
	}
}
