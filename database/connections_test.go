package database

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConnectionConfig(t *testing.T) {
	// Save original env vars
	origVars := map[string]string{
		"SQL_HOST":       os.Getenv("SQL_HOST"),
		"SQL_PORT":       os.Getenv("SQL_PORT"),
		"SQL_DATABASE":   os.Getenv("SQL_DATABASE"),
		"REDIS_HOST":     os.Getenv("REDIS_HOST"),
		"COSMOSDB_ENDPOINT": os.Getenv("COSMOSDB_ENDPOINT"),
	}
	defer func() {
		for k, v := range origVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Clear env vars
	os.Unsetenv("SQL_HOST")
	os.Unsetenv("SQL_PORT")
	os.Unsetenv("SQL_DATABASE")
	os.Unsetenv("REDIS_HOST")
	os.Unsetenv("COSMOSDB_ENDPOINT")

	config := DefaultConnectionConfig()

	if config.SQLHost != "localhost" {
		t.Errorf("expected default SQLHost=localhost, got %s", config.SQLHost)
	}
	if config.SQLPort != 1433 {
		t.Errorf("expected default SQLPort=1433, got %d", config.SQLPort)
	}
	if config.SQLDatabase != "cobrun" {
		t.Errorf("expected default SQLDatabase=cobrun, got %s", config.SQLDatabase)
	}
	if config.RedisHost != "localhost:6379" {
		t.Errorf("expected default RedisHost=localhost:6379, got %s", config.RedisHost)
	}
	if config.MaxRetries != 5 {
		t.Errorf("expected default MaxRetries=5, got %d", config.MaxRetries)
	}
	if config.RetryDelay != 1000*time.Millisecond {
		t.Errorf("expected default RetryDelay=1s, got %v", config.RetryDelay)
	}
	if config.ConnTimeout != 30*time.Second {
		t.Errorf("expected default ConnTimeout=30s, got %v", config.ConnTimeout)
	}
}

func TestDefaultConnectionConfig_WithEnvVars(t *testing.T) {
	// Save and restore env vars
	origVars := map[string]string{
		"SQL_HOST":       os.Getenv("SQL_HOST"),
		"SQL_PORT":       os.Getenv("SQL_PORT"),
		"SQL_DATABASE":   os.Getenv("SQL_DATABASE"),
		"SQL_USER":       os.Getenv("SQL_USER"),
		"REDIS_HOST":     os.Getenv("REDIS_HOST"),
		"REDIS_PASSWORD": os.Getenv("REDIS_PASSWORD"),
		"COSMOSDB_ENDPOINT": os.Getenv("COSMOSDB_ENDPOINT"),
		"COSMOSDB_DATABASE": os.Getenv("COSMOSDB_DATABASE"),
	}
	defer func() {
		for k, v := range origVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set test env vars
	os.Setenv("SQL_HOST", "test.database.windows.net")
	os.Setenv("SQL_PORT", "1433")
	os.Setenv("SQL_DATABASE", "testdb")
	os.Setenv("SQL_USER", "testuser")
	os.Setenv("REDIS_HOST", "test.redis.cache.windows.net:6380")
	os.Setenv("REDIS_PASSWORD", "test-password")
	os.Setenv("COSMOSDB_ENDPOINT", "https://test.documents.azure.com:443/")
	os.Setenv("COSMOSDB_DATABASE", "testcosmos")

	config := DefaultConnectionConfig()

	if config.SQLHost != "test.database.windows.net" {
		t.Errorf("expected SQLHost from env, got %s", config.SQLHost)
	}
	if config.SQLDatabase != "testdb" {
		t.Errorf("expected SQLDatabase from env, got %s", config.SQLDatabase)
	}
	if config.SQLUser != "testuser" {
		t.Errorf("expected SQLUser from env, got %s", config.SQLUser)
	}
	if config.RedisHost != "test.redis.cache.windows.net:6380" {
		t.Errorf("expected RedisHost from env, got %s", config.RedisHost)
	}
	if config.RedisPassword != "test-password" {
		t.Errorf("expected RedisPassword from env, got %s", config.RedisPassword)
	}
	if config.CosmosEndpoint != "https://test.documents.azure.com:443/" {
		t.Errorf("expected CosmosEndpoint from env, got %s", config.CosmosEndpoint)
	}
	if config.CosmosDatabase != "testcosmos" {
		t.Errorf("expected CosmosDatabase from env, got %s", config.CosmosDatabase)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "env var set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "env var not set",
			key:          "TEST_VAR_UNSET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "env var empty string",
			key:          "TEST_VAR_EMPTY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			orig := os.Getenv(tt.key)
			defer func() {
				if orig == "" {
					os.Unsetenv(tt.key)
				} else {
					os.Setenv(tt.key, orig)
				}
			}()

			// Set test value
			if tt.envValue == "" {
				os.Unsetenv(tt.key)
			} else {
				os.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "valid int",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "42",
			expected:     42,
		},
		{
			name:         "invalid int",
			key:          "TEST_INT_INVALID",
			defaultValue: 10,
			envValue:     "not-a-number",
			expected:     10,
		},
		{
			name:         "not set",
			key:          "TEST_INT_UNSET",
			defaultValue: 10,
			envValue:     "",
			expected:     10,
		},
		{
			name:         "zero value",
			key:          "TEST_INT_ZERO",
			defaultValue: 10,
			envValue:     "0",
			expected:     0,
		},
		{
			name:         "negative value",
			key:          "TEST_INT_NEGATIVE",
			defaultValue: 10,
			envValue:     "-5",
			expected:     -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			orig := os.Getenv(tt.key)
			defer func() {
				if orig == "" {
					os.Unsetenv(tt.key)
				} else {
					os.Setenv(tt.key, orig)
				}
			}()

			// Set test value
			if tt.envValue == "" {
				os.Unsetenv(tt.key)
			} else {
				os.Setenv(tt.key, tt.envValue)
			}

			result := getEnvInt(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "true string",
			key:          "TEST_BOOL_TRUE",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "1 string",
			key:          "TEST_BOOL_ONE",
			defaultValue: false,
			envValue:     "1",
			expected:     true,
		},
		{
			name:         "yes string",
			key:          "TEST_BOOL_YES",
			defaultValue: false,
			envValue:     "yes",
			expected:     true,
		},
		{
			name:         "false string",
			key:          "TEST_BOOL_FALSE",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "0 string",
			key:          "TEST_BOOL_ZERO",
			defaultValue: true,
			envValue:     "0",
			expected:     false,
		},
		{
			name:         "not set",
			key:          "TEST_BOOL_UNSET",
			defaultValue: true,
			envValue:     "",
			expected:     true,
		},
		{
			name:         "invalid string",
			key:          "TEST_BOOL_INVALID",
			defaultValue: true,
			envValue:     "maybe",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			orig := os.Getenv(tt.key)
			defer func() {
				if orig == "" {
					os.Unsetenv(tt.key)
				} else {
					os.Setenv(tt.key, orig)
				}
			}()

			// Set test value
			if tt.envValue == "" {
				os.Unsetenv(tt.key)
			} else {
				os.Setenv(tt.key, tt.envValue)
			}

			result := getEnvBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConnectionConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  ConnectionConfig
		hasSQL  bool
		hasCosmos bool
		hasRedis bool
	}{
		{
			name: "all databases configured",
			config: ConnectionConfig{
				SQLHost:        "localhost",
				CosmosEndpoint: "https://test.documents.azure.com:443/",
				RedisHost:      "localhost:6379",
			},
			hasSQL:    true,
			hasCosmos: true,
			hasRedis:  true,
		},
		{
			name: "only SQL",
			config: ConnectionConfig{
				SQLHost:        "localhost",
				CosmosEndpoint: "",
				RedisHost:      "",
			},
			hasSQL:    true,
			hasCosmos: false,
			hasRedis:  false,
		},
		{
			name: "only Cosmos",
			config: ConnectionConfig{
				SQLHost:        "",
				CosmosEndpoint: "https://test.documents.azure.com:443/",
				RedisHost:      "",
			},
			hasSQL:    false,
			hasCosmos: true,
			hasRedis:  false,
		},
		{
			name: "only Redis",
			config: ConnectionConfig{
				SQLHost:        "",
				CosmosEndpoint: "",
				RedisHost:      "localhost:6379",
			},
			hasSQL:    false,
			hasCosmos: false,
			hasRedis:  true,
		},
		{
			name: "none configured",
			config: ConnectionConfig{
				SQLHost:        "",
				CosmosEndpoint: "",
				RedisHost:      "",
			},
			hasSQL:    false,
			hasCosmos: false,
			hasRedis:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasSQL := tt.config.SQLHost != "" || tt.config.SQLConnString != ""
			hasCosmos := tt.config.CosmosEndpoint != ""
			hasRedis := tt.config.RedisHost != ""

			if hasSQL != tt.hasSQL {
				t.Errorf("expected hasSQL=%v, got %v", tt.hasSQL, hasSQL)
			}
			if hasCosmos != tt.hasCosmos {
				t.Errorf("expected hasCosmos=%v, got %v", tt.hasCosmos, hasCosmos)
			}
			if hasRedis != tt.hasRedis {
				t.Errorf("expected hasRedis=%v, got %v", tt.hasRedis, hasRedis)
			}
		})
	}
}

func TestConnections_Structure(t *testing.T) {
	conns := &Connections{
		SQL:    nil,
		Cosmos: nil,
		Redis:  nil,
		Config: &ConnectionConfig{
			SQLHost:    "localhost",
			RedisHost:  "localhost:6379",
			MaxRetries: 3,
		},
	}

	if conns.Config == nil {
		t.Error("Config should not be nil")
	}

	if conns.Config.SQLHost != "localhost" {
		t.Errorf("expected SQLHost=localhost, got %s", conns.Config.SQLHost)
	}

	if conns.Config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", conns.Config.MaxRetries)
	}
}

func TestConnectionConfig_Timeouts(t *testing.T) {
	tests := []struct {
		name           string
		retryDelayMS   int
		connTimeoutSec int
		expectedDelay  time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "default values",
			retryDelayMS:    1000,
			connTimeoutSec:  30,
			expectedDelay:   1000 * time.Millisecond,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "custom values",
			retryDelayMS:    500,
			connTimeoutSec:  60,
			expectedDelay:   500 * time.Millisecond,
			expectedTimeout: 60 * time.Second,
		},
		{
			name:            "fast retry",
			retryDelayMS:    100,
			connTimeoutSec:  10,
			expectedDelay:   100 * time.Millisecond,
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConnectionConfig{
				RetryDelay:  time.Duration(tt.retryDelayMS) * time.Millisecond,
				ConnTimeout: time.Duration(tt.connTimeoutSec) * time.Second,
			}

			if config.RetryDelay != tt.expectedDelay {
				t.Errorf("expected RetryDelay=%v, got %v", tt.expectedDelay, config.RetryDelay)
			}

			if config.ConnTimeout != tt.expectedTimeout {
				t.Errorf("expected ConnTimeout=%v, got %v", tt.expectedTimeout, config.ConnTimeout)
			}
		})
	}
}
