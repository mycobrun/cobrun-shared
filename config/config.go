// Package config provides configuration loading with Azure Key Vault integration.
package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds common configuration for all services.
type Config struct {
	// Service identification
	ServiceName string
	Environment string
	Version     string

	// HTTP server
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Logging
	LogLevel string

	// Azure
	KeyVaultName        string
	AppInsightsKey      string
	CosmosDBEndpoint    string
	CosmosDBKey         string
	CosmosDBDatabase    string
	ServiceBusNS        string
	EventHubsNS         string
	RedisHost           string
	RedisPassword       string
	SQLConnectionString string

	// JWT
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string
}

// Load loads configuration from environment variables.
// For production, secrets are loaded from Azure Key Vault.
func Load(serviceName string) (*Config, error) {
	cfg := &Config{
		ServiceName:  serviceName,
		Environment:  getEnv("ENVIRONMENT", "development"),
		Version:      getEnv("VERSION", "0.0.1"),
		Port:         getEnvInt("PORT", 8080),
		ReadTimeout:  getEnvDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:  getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		KeyVaultName: getEnv("KEY_VAULT_NAME", ""),
	}

	// Load secrets from Key Vault in production
	if cfg.KeyVaultName != "" && cfg.Environment != "development" {
		if err := cfg.loadFromKeyVault(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to load secrets from Key Vault: %w", err)
		}
	} else {
		// Load from environment variables in development
		cfg.loadFromEnv()
	}

	return cfg, nil
}

// MustLoad loads configuration and panics on error.
func MustLoad(serviceName string) *Config {
	cfg, err := Load(serviceName)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

func (c *Config) loadFromEnv() {
	c.AppInsightsKey = getEnv("APPINSIGHTS_INSTRUMENTATIONKEY", "")
	c.CosmosDBEndpoint = getEnvWithFallback("COSMOSDB_ENDPOINT", "COSMOS_DB_ENDPOINT", "")
	c.CosmosDBKey = getEnvWithFallback("COSMOSDB_KEY", "COSMOS_DB_KEY", "")
	c.CosmosDBDatabase = getEnvWithFallback("COSMOSDB_DATABASE", "COSMOS_DB_DATABASE", "cobrun")
	c.ServiceBusNS = getEnv("SERVICEBUS_NAMESPACE", "")
	c.EventHubsNS = getEnv("EVENTHUBS_NAMESPACE", "")
	c.RedisHost = getEnv("REDIS_HOST", "localhost:6379")
	c.RedisPassword = getEnv("REDIS_PASSWORD", "")
	c.SQLConnectionString = getEnv("SQL_CONNECTION_STRING", "")
	c.JWTIssuer = getEnv("JWT_ISSUER", "cobrun")
	c.JWTAudience = getEnv("JWT_AUDIENCE", "cobrun-api")

	// JWT_SECRET is REQUIRED - no default allowed
	// Only allow a development default if explicitly in development mode
	if c.IsDevelopment() {
		c.JWTSecret = getEnv("JWT_SECRET", "development-only-secret-do-not-use-in-prod")
	} else {
		c.JWTSecret = requireEnv("JWT_SECRET")
	}
}

func (c *Config) loadFromKeyVault(ctx context.Context) error {
	kv, err := NewKeyVaultClient(c.KeyVaultName)
	if err != nil {
		return err
	}

	secrets := map[string]*string{
		"appinsights-key":       &c.AppInsightsKey,
		"cosmosdb-endpoint":     &c.CosmosDBEndpoint,
		"cosmosdb-key":          &c.CosmosDBKey,
		"servicebus-namespace":  &c.ServiceBusNS,
		"eventhubs-namespace":   &c.EventHubsNS,
		"redis-host":            &c.RedisHost,
		"redis-password":        &c.RedisPassword,
		"sql-connection-string": &c.SQLConnectionString,
		"jwt-secret":            &c.JWTSecret,
	}

	for name, ptr := range secrets {
		value, err := kv.GetSecret(ctx, name)
		if err != nil {
			// Log warning but continue - some secrets may be optional
			continue
		}
		*ptr = value
	}

	// Set defaults for non-secret values
	c.CosmosDBDatabase = getEnv("COSMOSDB_DATABASE", "cobrun")
	c.JWTIssuer = getEnv("JWT_ISSUER", "cobrun")
	c.JWTAudience = getEnv("JWT_AUDIENCE", "cobrun-api")

	return nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvWithFallback gets an environment variable with fallback to another key.
func getEnvWithFallback(primary, fallback, defaultValue string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	if value := os.Getenv(fallback); value != "" {
		return value
	}
	return defaultValue
}

// requireEnv gets a required environment variable and panics if not set.
// Use this for security-critical configuration that MUST be provided.
func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

// GetEnv gets an environment variable with a default value.
func GetEnv(key, defaultValue string) string {
	return getEnv(key, defaultValue)
}

// GetEnvInt gets an environment variable as an integer with a default value.
func GetEnvInt(key string, defaultValue int) int {
	return getEnvInt(key, defaultValue)
}

// GetEnvBool gets an environment variable as a boolean with a default value.
func GetEnvBool(key string, defaultValue bool) bool {
	return getEnvBool(key, defaultValue)
}

// GetEnvDuration gets an environment variable as a duration with a default value.
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	return getEnvDuration(key, defaultValue)
}

// GetEnvFloat gets an environment variable as a float with a default value.
func GetEnvFloat(key string, defaultValue float64) float64 {
	return getEnvFloat(key, defaultValue)
}
