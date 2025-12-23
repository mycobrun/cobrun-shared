// Package auth provides JWT authentication utilities.
package auth

import (
	"testing"
	"time"
)

func TestClaims_Role(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		want  string
	}{
		{"no roles", nil, ""},
		{"empty roles", []string{}, ""},
		{"single role", []string{"admin"}, "admin"},
		{"multiple roles", []string{"admin", "user"}, "admin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{Roles: tt.roles}
			if got := c.Role(); got != tt.want {
				t.Errorf("Role() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaims_HasRole(t *testing.T) {
	claims := &Claims{Roles: []string{"admin", "user", "driver"}}

	tests := []struct {
		name string
		role string
		want bool
	}{
		{"has admin", "admin", true},
		{"has user", "user", true},
		{"has driver", "driver", true},
		{"no superuser", "superuser", false},
		{"no empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claims.HasRole(tt.role); got != tt.want {
				t.Errorf("HasRole(%s) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestClaims_HasRole_Empty(t *testing.T) {
	claims := &Claims{Roles: nil}
	if claims.HasRole("admin") {
		t.Error("empty claims should not have any role")
	}
}

func TestDefaultJWTConfig(t *testing.T) {
	config := DefaultJWTConfig()

	if config.AccessExpiry != 15*time.Minute {
		t.Errorf("AccessExpiry = %v, want 15 minutes", config.AccessExpiry)
	}

	if config.RefreshExpiry != 7*24*time.Hour {
		t.Errorf("RefreshExpiry = %v, want 7 days", config.RefreshExpiry)
	}
}

func TestNewJWTManager(t *testing.T) {
	config := JWTConfig{
		Secret:        "test-secret",
		Issuer:        "test-issuer",
		Audience:      "test-audience",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}

	manager := NewJWTManager(config)
	if manager == nil {
		t.Fatal("NewJWTManager returned nil")
	}
}

func TestJWTManager_GenerateAndValidateAccessToken(t *testing.T) {
	config := JWTConfig{
		Secret:        "test-secret-key-at-least-32-bytes!!",
		Issuer:        "cobrun",
		Audience:      "cobrun-api",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}

	manager := NewJWTManager(config)

	// Generate token
	token, err := manager.GenerateAccessToken("user-123", "test@example.com", "rider", []string{"user"})
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	if token == "" {
		t.Error("token should not be empty")
	}

	// Validate token
	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %s, want user-123", claims.UserID)
	}

	if claims.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", claims.Email)
	}

	if claims.UserType != "rider" {
		t.Errorf("UserType = %s, want rider", claims.UserType)
	}

	if !claims.HasRole("user") {
		t.Error("should have role 'user'")
	}
}

func TestJWTManager_GenerateAndValidateRefreshToken(t *testing.T) {
	config := JWTConfig{
		Secret:        "test-secret-key-at-least-32-bytes!!",
		Issuer:        "cobrun",
		Audience:      "cobrun-api",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}

	manager := NewJWTManager(config)

	// Generate refresh token
	token, err := manager.GenerateRefreshToken("user-123")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	if token == "" {
		t.Error("token should not be empty")
	}

	// Validate refresh token
	userID, err := manager.ValidateRefreshToken(token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}

	if userID != "user-123" {
		t.Errorf("UserID = %s, want user-123", userID)
	}
}

func TestJWTManager_ValidateToken_Invalid(t *testing.T) {
	config := JWTConfig{
		Secret:        "test-secret-key-at-least-32-bytes!!",
		Issuer:        "cobrun",
		Audience:      "cobrun-api",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}

	manager := NewJWTManager(config)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "not-a-jwt"},
		{"tampered token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.ValidateToken(tt.token)
			if err == nil {
				t.Error("ValidateToken should fail for invalid token")
			}
		})
	}
}

func TestJWTManager_ValidateToken_WrongSecret(t *testing.T) {
	config1 := JWTConfig{
		Secret:       "secret-key-1-at-least-32-bytes!!!",
		Issuer:       "cobrun",
		Audience:     "cobrun-api",
		AccessExpiry: 15 * time.Minute,
	}

	config2 := JWTConfig{
		Secret:       "secret-key-2-at-least-32-bytes!!!",
		Issuer:       "cobrun",
		Audience:     "cobrun-api",
		AccessExpiry: 15 * time.Minute,
	}

	manager1 := NewJWTManager(config1)
	manager2 := NewJWTManager(config2)

	// Generate with manager1
	token, _ := manager1.GenerateAccessToken("user-123", "test@example.com", "rider", nil)

	// Validate with manager2 (different secret)
	_, err := manager2.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken should fail with wrong secret")
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	config := JWTConfig{
		Secret:        "test-secret-key-at-least-32-bytes!!",
		Issuer:        "cobrun",
		Audience:      "cobrun-api",
		AccessExpiry:  -1 * time.Hour, // Already expired
		RefreshExpiry: 7 * 24 * time.Hour,
	}

	manager := NewJWTManager(config)

	// Generate already expired token
	token, err := manager.GenerateAccessToken("user-123", "test@example.com", "rider", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Validation should fail with expired error
	_, err = manager.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken should fail for expired token")
	}
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestJWTErrors(t *testing.T) {
	if ErrInvalidToken.Error() != "invalid token" {
		t.Error("ErrInvalidToken message mismatch")
	}
	if ErrTokenExpired.Error() != "token expired" {
		t.Error("ErrTokenExpired message mismatch")
	}
	if ErrNoToken.Error() != "no token provided" {
		t.Error("ErrNoToken message mismatch")
	}
}

func TestClaims_Fields(t *testing.T) {
	claims := &Claims{
		UserID:   "user-123",
		Email:    "test@example.com",
		Name:     "Test User",
		UserType: "rider",
		Roles:    []string{"user", "admin"},
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %s, want user-123", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", claims.Email)
	}
	if claims.Name != "Test User" {
		t.Errorf("Name = %s, want Test User", claims.Name)
	}
	if claims.UserType != "rider" {
		t.Errorf("UserType = %s, want rider", claims.UserType)
	}
}
