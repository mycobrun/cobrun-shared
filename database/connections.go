// Package database provides unified database connection management.
package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/cobrun/cobrun-platform/pkg/config"
	"github.com/cobrun/cobrun-platform/pkg/database/cosmosdb"
)

// Connections holds all database connections.
type Connections struct {
	SQL      *SQLClient
	Cosmos   *CosmosClient
	Redis    *RedisClient
	Config   *ConnectionConfig
}

// ConnectionConfig holds all database configuration.
type ConnectionConfig struct {
	// SQL Server
	SQLHost       string
	SQLPort       int
	SQLDatabase   string
	SQLUser       string
	SQLPassword   string
	SQLUseMSI     bool
	SQLConnString string

	// Cosmos DB
	CosmosEndpoint string
	CosmosKey      string
	CosmosDatabase string

	// Redis
	RedisHost     string
	RedisPassword string
	RedisDB       int

	// Connection options
	MaxRetries    int
	RetryDelay    time.Duration
	ConnTimeout   time.Duration
}

// DefaultConnectionConfig returns configuration loaded from environment variables.
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		// SQL Server
		SQLHost:       getEnv("SQL_HOST", "localhost"),
		SQLPort:       getEnvInt("SQL_PORT", 1433),
		SQLDatabase:   getEnv("SQL_DATABASE", "cobrun"),
		SQLUser:       getEnv("SQL_USER", "sa"),
		SQLPassword:   getEnv("SQL_PASSWORD", ""),
		SQLUseMSI:     getEnvBool("SQL_USE_MSI", false),
		SQLConnString: os.Getenv("SQL_CONNECTION_STRING"),

		// Cosmos DB
		CosmosEndpoint: getEnv("COSMOSDB_ENDPOINT", ""),
		CosmosKey:      getEnv("COSMOSDB_KEY", ""),
		CosmosDatabase: getEnv("COSMOSDB_DATABASE", "cobrun"),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		// Options
		MaxRetries:  getEnvInt("DB_MAX_RETRIES", 5),
		RetryDelay:  time.Duration(getEnvInt("DB_RETRY_DELAY_MS", 1000)) * time.Millisecond,
		ConnTimeout: time.Duration(getEnvInt("DB_CONN_TIMEOUT_SEC", 30)) * time.Second,
	}
}

// ConnectionConfigFromConfig creates a ConnectionConfig from a config.Config.
// This allows seamless integration with Key Vault-based configuration.
func ConnectionConfigFromConfig(cfg *config.Config) *ConnectionConfig {
	return &ConnectionConfig{
		// SQL Server (from Key Vault or env)
		SQLUseMSI:     cfg.IsProduction(), // Use MSI in production
		SQLConnString: cfg.SQLConnectionString,

		// Cosmos DB (from Key Vault or env)
		CosmosEndpoint: cfg.CosmosDBEndpoint,
		CosmosDatabase: cfg.CosmosDBDatabase,

		// Redis (from Key Vault or env)
		RedisHost:     cfg.RedisHost,
		RedisPassword: cfg.RedisPassword,
		RedisDB:       0,

		// Options
		MaxRetries:  getEnvInt("DB_MAX_RETRIES", 5),
		RetryDelay:  time.Duration(getEnvInt("DB_RETRY_DELAY_MS", 1000)) * time.Millisecond,
		ConnTimeout: time.Duration(getEnvInt("DB_CONN_TIMEOUT_SEC", 30)) * time.Second,
	}
}

// NewConnectionsFromConfig creates database connections using a config.Config.
// This is the preferred method for production use with Key Vault integration.
func NewConnectionsFromConfig(ctx context.Context, cfg *config.Config) (*Connections, error) {
	connConfig := ConnectionConfigFromConfig(cfg)
	return NewConnections(ctx, connConfig)
}

// NewConnections creates database connections based on configuration.
func NewConnections(ctx context.Context, config *ConnectionConfig) (*Connections, error) {
	if config == nil {
		config = DefaultConnectionConfig()
	}

	conns := &Connections{
		Config: config,
	}

	// Connect to SQL Server if configured
	if config.SQLHost != "" || config.SQLConnString != "" {
		sqlConfig := SQLConfig{
			Host:     config.SQLHost,
			Port:     config.SQLPort,
			Database: config.SQLDatabase,
			User:     config.SQLUser,
			Password: config.SQLPassword,
			UseMSI:   config.SQLUseMSI,
		}

		var err error
		for i := 0; i <= config.MaxRetries; i++ {
			conns.SQL, err = NewSQLClient(ctx, sqlConfig)
			if err == nil {
				log.Println("Connected to SQL Server")
				break
			}
			if i < config.MaxRetries {
				log.Printf("SQL Server connection attempt %d failed, retrying in %v: %v", i+1, config.RetryDelay, err)
				time.Sleep(config.RetryDelay)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQL Server after %d attempts: %w", config.MaxRetries, err)
		}
	}

	// Connect to Cosmos DB if configured
	if config.CosmosEndpoint != "" {
		cosmosConfig := CosmosConfig{
			Endpoint:     config.CosmosEndpoint,
			Key:          config.CosmosKey,
			DatabaseName: config.CosmosDatabase,
		}

		var err error
		for i := 0; i <= config.MaxRetries; i++ {
			conns.Cosmos, err = NewCosmosClient(ctx, cosmosConfig)
			if err == nil {
				log.Println("Connected to Cosmos DB")
				break
			}
			if i < config.MaxRetries {
				log.Printf("Cosmos DB connection attempt %d failed, retrying in %v: %v", i+1, config.RetryDelay, err)
				time.Sleep(config.RetryDelay)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Cosmos DB after %d attempts: %w", config.MaxRetries, err)
		}
	}

	// Connect to Redis if configured
	if config.RedisHost != "" {
		redisConfig := RedisConfig{
			Host:     config.RedisHost,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		}

		var err error
		for i := 0; i <= config.MaxRetries; i++ {
			conns.Redis, err = NewRedisClient(ctx, redisConfig)
			if err == nil {
				log.Println("Connected to Redis")
				break
			}
			if i < config.MaxRetries {
				log.Printf("Redis connection attempt %d failed, retrying in %v: %v", i+1, config.RetryDelay, err)
				time.Sleep(config.RetryDelay)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Redis after %d attempts: %w", config.MaxRetries, err)
		}
	}

	return conns, nil
}

// InitializeAll initializes all databases (creates schemas, containers, etc).
func (c *Connections) InitializeAll(ctx context.Context) error {
	// Initialize Cosmos DB containers
	if c.Cosmos != nil {
		initializer := cosmosdb.NewInitializer(c.Cosmos.client, nil)
		if err := initializer.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize Cosmos DB: %w", err)
		}
	}

	// Initialize Redis
	if c.Redis != nil {
		initializer := NewRedisInitializer(c.Redis)
		if err := initializer.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize Redis: %w", err)
		}
	}

	return nil
}

// RunMigrations runs SQL database migrations.
func (c *Connections) RunMigrations(ctx context.Context, migrationsFS interface{}, dir string) (int, error) {
	if c.SQL == nil {
		return 0, fmt.Errorf("SQL client not initialized")
	}

	migrator := NewMigrator(c.SQL)

	// Type assert and load migrations
	if fs, ok := migrationsFS.(interface{ ReadDir(string) ([]interface{}, error) }); ok {
		_ = fs // Load from FS
	}

	// For now, just initialize and return
	if err := migrator.Initialize(ctx); err != nil {
		return 0, err
	}

	return migrator.Up(ctx)
}

// Close closes all database connections.
func (c *Connections) Close() {
	if c.SQL != nil {
		if err := c.SQL.Close(); err != nil {
			log.Printf("Error closing SQL connection: %v", err)
		}
	}
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			log.Printf("Error closing Redis connection: %v", err)
		}
	}
	// Note: Cosmos client doesn't need explicit close
}

// HealthCheck performs health checks on all connections.
func (c *Connections) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)

	if c.SQL != nil {
		if err := c.SQL.Ping(ctx); err != nil {
			results["sql"] = err
		} else {
			results["sql"] = nil
		}
	}

	if c.Cosmos != nil {
		// Cosmos health check - try to get database
		_, err := c.Cosmos.client.NewDatabase(c.Config.CosmosDatabase)
		results["cosmos"] = err
	}

	if c.Redis != nil {
		results["redis"] = c.Redis.Ping(ctx)
	}

	return results
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

