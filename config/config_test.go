// Package config provides configuration loading with Azure Key Vault integration.
package config

import (
	"os"
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	// Set up test environment variable
	os.Setenv("TEST_GET_ENV", "test-value")
	defer os.Unsetenv("TEST_GET_ENV")

	tests := []struct {
		name         string
		key          string
		defaultValue string
		want         string
	}{
		{"existing var", "TEST_GET_ENV", "default", "test-value"},
		{"missing var", "NONEXISTENT_VAR_12345", "default", "default"},
		{"empty default", "NONEXISTENT_VAR_12345", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEnv(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT_VALID", "42")
	os.Setenv("TEST_INT_INVALID", "not-an-int")
	defer os.Unsetenv("TEST_INT_VALID")
	defer os.Unsetenv("TEST_INT_INVALID")

	tests := []struct {
		name         string
		key          string
		defaultValue int
		want         int
	}{
		{"valid int", "TEST_INT_VALID", 0, 42},
		{"invalid int", "TEST_INT_INVALID", 99, 99},
		{"missing var", "NONEXISTENT_VAR_12345", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEnvInt(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnvInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "false")
	os.Setenv("TEST_BOOL_ONE", "1")
	os.Setenv("TEST_BOOL_ZERO", "0")
	os.Setenv("TEST_BOOL_TRUE_UPPER", "TRUE")
	defer func() {
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_FALSE")
		os.Unsetenv("TEST_BOOL_ONE")
		os.Unsetenv("TEST_BOOL_ZERO")
		os.Unsetenv("TEST_BOOL_TRUE_UPPER")
	}()

	tests := []struct {
		name         string
		key          string
		defaultValue bool
		want         bool
	}{
		{"true string", "TEST_BOOL_TRUE", false, true},
		{"false string", "TEST_BOOL_FALSE", true, false},
		{"1 string", "TEST_BOOL_ONE", false, true},
		{"0 string", "TEST_BOOL_ZERO", true, false},
		{"TRUE uppercase", "TEST_BOOL_TRUE_UPPER", false, true},
		{"missing with true default", "NONEXISTENT_VAR_12345", true, true},
		{"missing with false default", "NONEXISTENT_VAR_12345", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEnvBool(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnvBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	os.Setenv("TEST_DURATION_VALID", "5m")
	os.Setenv("TEST_DURATION_SECONDS", "30s")
	os.Setenv("TEST_DURATION_INVALID", "not-a-duration")
	defer func() {
		os.Unsetenv("TEST_DURATION_VALID")
		os.Unsetenv("TEST_DURATION_SECONDS")
		os.Unsetenv("TEST_DURATION_INVALID")
	}()

	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		want         time.Duration
	}{
		{"valid minutes", "TEST_DURATION_VALID", time.Second, 5 * time.Minute},
		{"valid seconds", "TEST_DURATION_SECONDS", time.Minute, 30 * time.Second},
		{"invalid duration", "TEST_DURATION_INVALID", time.Hour, time.Hour},
		{"missing var", "NONEXISTENT_VAR_12345", 10 * time.Minute, 10 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEnvDuration(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnvDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvFloat(t *testing.T) {
	os.Setenv("TEST_FLOAT_VALID", "3.14")
	os.Setenv("TEST_FLOAT_INT", "42")
	os.Setenv("TEST_FLOAT_INVALID", "not-a-float")
	defer func() {
		os.Unsetenv("TEST_FLOAT_VALID")
		os.Unsetenv("TEST_FLOAT_INT")
		os.Unsetenv("TEST_FLOAT_INVALID")
	}()

	tests := []struct {
		name         string
		key          string
		defaultValue float64
		want         float64
	}{
		{"valid float", "TEST_FLOAT_VALID", 0, 3.14},
		{"int as float", "TEST_FLOAT_INT", 0, 42.0},
		{"invalid float", "TEST_FLOAT_INVALID", 99.9, 99.9},
		{"missing var", "NONEXISTENT_VAR_12345", 1.5, 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEnvFloat(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnvFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvSlice(t *testing.T) {
	os.Setenv("TEST_SLICE_MULTI", "a,b,c")
	os.Setenv("TEST_SLICE_SPACES", " a , b , c ")
	os.Setenv("TEST_SLICE_SINGLE", "single")
	os.Setenv("TEST_SLICE_EMPTY", "")
	defer func() {
		os.Unsetenv("TEST_SLICE_MULTI")
		os.Unsetenv("TEST_SLICE_SPACES")
		os.Unsetenv("TEST_SLICE_SINGLE")
		os.Unsetenv("TEST_SLICE_EMPTY")
	}()

	tests := []struct {
		name         string
		key          string
		defaultValue string
		wantLen      int
		wantFirst    string
	}{
		{"multiple values", "TEST_SLICE_MULTI", "", 3, "a"},
		{"with spaces", "TEST_SLICE_SPACES", "", 3, "a"},
		{"single value", "TEST_SLICE_SINGLE", "", 1, "single"},
		{"empty with default", "TEST_SLICE_EMPTY", "x,y", 2, "x"},
		{"missing with default", "NONEXISTENT_VAR_12345", "default1,default2", 2, "default1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEnvSlice(tt.key, tt.defaultValue)
			if len(got) != tt.wantLen {
				t.Errorf("GetEnvSlice() len = %v, want %v", len(got), tt.wantLen)
			}
			if len(got) > 0 && got[0] != tt.wantFirst {
				t.Errorf("GetEnvSlice()[0] = %v, want %v", got[0], tt.wantFirst)
			}
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{"development", "development", true},
		{"production", "production", false},
		{"staging", "staging", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Environment: tt.environment}
			if got := c.IsDevelopment(); got != tt.want {
				t.Errorf("IsDevelopment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{"production", "production", true},
		{"development", "development", false},
		{"staging", "staging", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Environment: tt.environment}
			if got := c.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad_Development(t *testing.T) {
	// Clear any existing environment
	originalEnv := os.Getenv("ENVIRONMENT")
	os.Setenv("ENVIRONMENT", "development")
	defer os.Setenv("ENVIRONMENT", originalEnv)

	cfg, err := Load("test-service")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify service name
	if cfg.ServiceName != "test-service" {
		t.Errorf("ServiceName = %s, want test-service", cfg.ServiceName)
	}

	// Verify development mode
	if !cfg.IsDevelopment() {
		t.Error("should be in development mode")
	}

	// Verify defaults
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %s, want info", cfg.LogLevel)
	}

	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want 30s", cfg.ReadTimeout)
	}

	if cfg.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout = %v, want 30s", cfg.WriteTimeout)
	}

	if cfg.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want 60s", cfg.IdleTimeout)
	}

	// JWT defaults
	if cfg.JWTIssuer != "cobrun" {
		t.Errorf("JWTIssuer = %s, want cobrun", cfg.JWTIssuer)
	}

	if cfg.JWTAudience != "cobrun-api" {
		t.Errorf("JWTAudience = %s, want cobrun-api", cfg.JWTAudience)
	}

	// Rate limiting defaults
	if !cfg.RateLimitEnabled {
		t.Error("RateLimitEnabled should be true by default")
	}

	if cfg.RateLimitRPS != 100 {
		t.Errorf("RateLimitRPS = %f, want 100", cfg.RateLimitRPS)
	}

	if cfg.RateLimitBurst != 200 {
		t.Errorf("RateLimitBurst = %d, want 200", cfg.RateLimitBurst)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	originalEnv := os.Getenv("ENVIRONMENT")
	originalPort := os.Getenv("PORT")

	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("PORT", "9000")

	defer func() {
		os.Setenv("ENVIRONMENT", originalEnv)
		os.Setenv("PORT", originalPort)
	}()

	cfg, err := Load("test-service")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != 9000 {
		t.Errorf("Port = %d, want 9000", cfg.Port)
	}
}

func TestLoad_CustomTimeouts(t *testing.T) {
	originalEnv := os.Getenv("ENVIRONMENT")
	originalRead := os.Getenv("READ_TIMEOUT")
	originalWrite := os.Getenv("WRITE_TIMEOUT")
	originalIdle := os.Getenv("IDLE_TIMEOUT")

	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("READ_TIMEOUT", "10s")
	os.Setenv("WRITE_TIMEOUT", "15s")
	os.Setenv("IDLE_TIMEOUT", "30s")

	defer func() {
		os.Setenv("ENVIRONMENT", originalEnv)
		os.Setenv("READ_TIMEOUT", originalRead)
		os.Setenv("WRITE_TIMEOUT", originalWrite)
		os.Setenv("IDLE_TIMEOUT", originalIdle)
	}()

	cfg, err := Load("test-service")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout = %v, want 10s", cfg.ReadTimeout)
	}

	if cfg.WriteTimeout != 15*time.Second {
		t.Errorf("WriteTimeout = %v, want 15s", cfg.WriteTimeout)
	}

	if cfg.IdleTimeout != 30*time.Second {
		t.Errorf("IdleTimeout = %v, want 30s", cfg.IdleTimeout)
	}
}

func TestLoad_CORSOrigins_Development(t *testing.T) {
	originalEnv := os.Getenv("ENVIRONMENT")
	os.Setenv("ENVIRONMENT", "development")
	defer os.Setenv("ENVIRONMENT", originalEnv)

	cfg, err := Load("test-service")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Development should have localhost origins
	if len(cfg.CORSAllowedOrigins) < 2 {
		t.Error("development should have at least 2 CORS origins")
	}

	foundLocalhost := false
	for _, origin := range cfg.CORSAllowedOrigins {
		if origin == "http://localhost:3000" || origin == "http://localhost:8080" {
			foundLocalhost = true
			break
		}
	}
	if !foundLocalhost {
		t.Error("development should include localhost origins")
	}
}

func TestLoad_CustomCORSOrigins(t *testing.T) {
	originalEnv := os.Getenv("ENVIRONMENT")
	originalCORS := os.Getenv("CORS_ORIGINS")

	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("CORS_ORIGINS", "https://custom.example.com,https://other.example.com")

	defer func() {
		os.Setenv("ENVIRONMENT", originalEnv)
		os.Setenv("CORS_ORIGINS", originalCORS)
	}()

	cfg, err := Load("test-service")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Errorf("expected 2 CORS origins, got %d", len(cfg.CORSAllowedOrigins))
	}

	if cfg.CORSAllowedOrigins[0] != "https://custom.example.com" {
		t.Errorf("first origin = %s, want https://custom.example.com", cfg.CORSAllowedOrigins[0])
	}
}

func TestLoad_Version(t *testing.T) {
	originalEnv := os.Getenv("ENVIRONMENT")
	originalVersion := os.Getenv("VERSION")

	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("VERSION", "1.2.3")

	defer func() {
		os.Setenv("ENVIRONMENT", originalEnv)
		os.Setenv("VERSION", originalVersion)
	}()

	cfg, err := Load("test-service")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Version != "1.2.3" {
		t.Errorf("Version = %s, want 1.2.3", cfg.Version)
	}
}

func TestLoad_LogLevel(t *testing.T) {
	originalEnv := os.Getenv("ENVIRONMENT")
	originalLevel := os.Getenv("LOG_LEVEL")

	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("LOG_LEVEL", "debug")

	defer func() {
		os.Setenv("ENVIRONMENT", originalEnv)
		os.Setenv("LOG_LEVEL", originalLevel)
	}()

	cfg, err := Load("test-service")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %s, want debug", cfg.LogLevel)
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := &Config{
		ServiceName:         "test-service",
		Environment:         "production",
		Version:             "2.0.0",
		Port:                8080,
		ReadTimeout:         30 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         60 * time.Second,
		LogLevel:            "warn",
		KeyVaultName:        "test-vault",
		AppInsightsKey:      "test-key",
		CosmosDBEndpoint:    "https://cosmos.example.com",
		CosmosDBKey:         "secret-key",
		CosmosDBDatabase:    "testdb",
		ServiceBusNS:        "test-sb",
		EventHubsNS:         "test-eh",
		RedisHost:           "redis:6379",
		RedisPassword:       "redis-password",
		SQLConnectionString: "sql-connection",
		JWTSecret:           "jwt-secret",
		JWTIssuer:           "test-issuer",
		JWTAudience:         "test-audience",
		CORSAllowedOrigins:  []string{"https://example.com"},
		RateLimitEnabled:    true,
		RateLimitRPS:        50,
		RateLimitBurst:      100,
	}

	if cfg.ServiceName != "test-service" {
		t.Error("ServiceName field mismatch")
	}
	if cfg.Environment != "production" {
		t.Error("Environment field mismatch")
	}
	if cfg.CosmosDBEndpoint != "https://cosmos.example.com" {
		t.Error("CosmosDBEndpoint field mismatch")
	}
	if cfg.RedisHost != "redis:6379" {
		t.Error("RedisHost field mismatch")
	}
	if len(cfg.CORSAllowedOrigins) != 1 {
		t.Error("CORSAllowedOrigins field mismatch")
	}
}

func TestMustLoad_Development(t *testing.T) {
	originalEnv := os.Getenv("ENVIRONMENT")
	os.Setenv("ENVIRONMENT", "development")
	defer os.Setenv("ENVIRONMENT", originalEnv)

	// MustLoad should not panic in development mode
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustLoad() panicked unexpectedly: %v", r)
		}
	}()

	cfg := MustLoad("test-service")
	if cfg == nil {
		t.Error("MustLoad() returned nil")
	}
}

func TestGetEnvSlice_EmptyElements(t *testing.T) {
	os.Setenv("TEST_SLICE_EMPTY_ELEMENTS", "a,,b,,,c")
	defer os.Unsetenv("TEST_SLICE_EMPTY_ELEMENTS")

	got := GetEnvSlice("TEST_SLICE_EMPTY_ELEMENTS", "")
	// Empty elements should be filtered out
	if len(got) != 3 {
		t.Errorf("GetEnvSlice() should filter empty elements, got len = %d", len(got))
	}
}
