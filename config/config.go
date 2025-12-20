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
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	// Logging
	LogLevel string

	// Azure Key Vault
	KeyVaultName string

	// Azure SQL Database
	SQLHost             string
	SQLPort             int
	SQLDatabase         string
	SQLUser             string
	SQLPassword         string
	SQLConnectionString string

	// Azure Cosmos DB
	CosmosDBEndpoint string
	CosmosDBDatabase string
	CosmosDBKey      string

	// Azure Redis Cache
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisSSL      bool

	// Azure Services
	AppInsightsKey string
	ServiceBusNS   string
	EventHubsNS    string

	// JWT Authentication
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string
}

// Load loads configuration from environment variables.
// For production, secrets are loaded from Azure Key Vault.
// serviceName is optional; if not provided, it uses SERVICE_NAME env var.
func Load(serviceName ...string) (*Config, error) {
	name := GetEnv("SERVICE_NAME", "unknown")
	if len(serviceName) > 0 && serviceName[0] != "" {
		name = serviceName[0]
	}
	
	cfg := &Config{
		ServiceName:  name,
		Environment:  getEnv("ENVIRONMENT", "development"),
		Version:      getEnv("VERSION", "0.0.1"),
		Port:            getEnvInt("PORT", 8080),
		ReadTimeout:     getEnvDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:     getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		KeyVaultName: getEnv("KEY_VAULT_NAME", ""),
	}

	// Load secrets from Key Vault in production/staging
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
func MustLoad(serviceName ...string) *Config {
	cfg, err := Load(serviceName...)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

func (c *Config) loadFromEnv() {
	// Azure SQL Database
	c.SQLHost = getEnv("AZURE_SQL_HOST", getEnv("SQL_HOST", "localhost"))
	c.SQLPort = getEnvInt("AZURE_SQL_PORT", getEnvInt("SQL_PORT", 1433))
	c.SQLDatabase = getEnv("AZURE_SQL_DATABASE", getEnv("SQL_DATABASE", "cobrun"))
	c.SQLUser = getEnv("AZURE_SQL_USER", getEnv("SQL_USER", "sa"))
	c.SQLPassword = getEnv("AZURE_SQL_PASSWORD", getEnv("SQL_PASSWORD", ""))
	c.SQLConnectionString = getEnv("SQL_CONNECTION_STRING", "")

	// Build connection string if not provided
	if c.SQLConnectionString == "" && c.SQLHost != "" {
		c.SQLConnectionString = c.buildSQLConnectionString()
	}

	// Azure Cosmos DB
	c.CosmosDBEndpoint = getEnv("COSMOS_ENDPOINT", getEnv("COSMOSDB_ENDPOINT", ""))
	c.CosmosDBDatabase = getEnv("COSMOS_DATABASE", getEnv("COSMOSDB_DATABASE", "cobrun"))
	c.CosmosDBKey = getEnv("COSMOS_KEY", getEnv("COSMOSDB_KEY", ""))

	// Azure Redis Cache
	c.RedisHost = getEnv("REDIS_HOST", "localhost")
	c.RedisPort = getEnvInt("REDIS_PORT", 6379)
	c.RedisPassword = getEnv("REDIS_PASSWORD", getEnv("REDIS_KEY", ""))
	c.RedisSSL = getEnvBool("REDIS_SSL", false)

	// Azure Services
	c.AppInsightsKey = getEnv("APPINSIGHTS_INSTRUMENTATIONKEY", "")
	c.ServiceBusNS = getEnv("SERVICEBUS_NAMESPACE", "")
	c.EventHubsNS = getEnv("EVENTHUBS_NAMESPACE", "")

	// JWT Authentication
	c.JWTIssuer = getEnv("JWT_ISSUER", "cobrun")
	c.JWTAudience = getEnv("JWT_AUDIENCE", "cobrun-api")

	// JWT_SECRET is REQUIRED - no default allowed in production
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

	// Map Key Vault secret names to config fields
	// Secret names match what we stored in Azure Key Vault
	secretMappings := []struct {
		secretName string
		target     *string
		required   bool
	}{
		// SQL Database
		{"sql-host", &c.SQLHost, true},
		{"sql-database", &c.SQLDatabase, true},
		{"sql-user", &c.SQLUser, true},
		{"sql-password", &c.SQLPassword, true},
		{"sql-connection-string", &c.SQLConnectionString, false},

		// Cosmos DB
		{"cosmos-endpoint", &c.CosmosDBEndpoint, true},
		{"cosmos-database", &c.CosmosDBDatabase, false},
		{"cosmos-key", &c.CosmosDBKey, true},

		// Redis
		{"redis-host", &c.RedisHost, true},
		{"redis-password", &c.RedisPassword, true},

		// JWT
		{"jwt-secret", &c.JWTSecret, false},

		// Other Azure services
		{"appinsights-key", &c.AppInsightsKey, false},
		{"servicebus-namespace", &c.ServiceBusNS, false},
		{"eventhubs-namespace", &c.EventHubsNS, false},
	}

	for _, mapping := range secretMappings {
		value, err := kv.GetSecret(ctx, mapping.secretName)
		if err != nil {
			if mapping.required {
				return fmt.Errorf("required secret %s not found: %w", mapping.secretName, err)
			}
			// Skip optional secrets
			continue
		}
		*mapping.target = value
	}

	// Load Redis port separately (stored as string in Key Vault)
	if portStr, err := kv.GetSecret(ctx, "redis-port"); err == nil {
		if port, err := strconv.Atoi(portStr); err == nil {
			c.RedisPort = port
		}
	}

	// Set defaults for non-secret values
	c.SQLPort = getEnvInt("AZURE_SQL_PORT", 1433)
	c.RedisSSL = true // Always use SSL in production
	c.JWTIssuer = getEnv("JWT_ISSUER", "cobrun")
	c.JWTAudience = getEnv("JWT_AUDIENCE", "cobrun-api")

	// Build connection string if not loaded from Key Vault
	if c.SQLConnectionString == "" && c.SQLHost != "" {
		c.SQLConnectionString = c.buildSQLConnectionString()
	}

	return nil
}

// buildSQLConnectionString builds an Azure SQL connection string from individual components.
func (c *Config) buildSQLConnectionString() string {
	return fmt.Sprintf(
		"Server=tcp:%s,%d;Initial Catalog=%s;Persist Security Info=False;User ID=%s;Password=%s;MultipleActiveResultSets=False;Encrypt=True;TrustServerCertificate=False;Connection Timeout=30;",
		c.SQLHost, c.SQLPort, c.SQLDatabase, c.SQLUser, c.SQLPassword,
	)
}

// GetRedisAddr returns the Redis address in host:port format.
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsStaging returns true if running in staging mode.
func (c *Config) IsStaging() bool {
	return c.Environment == "staging"
}

// Helper functions (exported for use by service-specific configs)

// GetEnv returns an environment variable value or a default.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// internal getEnv for use within this package
func getEnv(key, defaultValue string) string {
	return GetEnv(key, defaultValue)
}

// RequireEnv gets a required environment variable and panics if not set.
// Use this for security-critical configuration that MUST be provided.
func RequireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

// internal requireEnv for use within this package
func requireEnv(key string) string {
	return RequireEnv(key)
}

// GetEnvInt returns an environment variable as int or a default.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// internal getEnvInt for use within this package
func getEnvInt(key string, defaultValue int) int {
	return GetEnvInt(key, defaultValue)
}

// GetEnvBool returns an environment variable as bool or a default.
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

// internal getEnvBool for use within this package
func getEnvBool(key string, defaultValue bool) bool {
	return GetEnvBool(key, defaultValue)
}

// GetEnvDuration returns an environment variable as time.Duration or a default.
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

// internal getEnvDuration for use within this package
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	return GetEnvDuration(key, defaultValue)
}

// GetEnvFloat returns an environment variable as float64 or a default.
func GetEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}
