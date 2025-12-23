// Package bootstrap provides helpers for initializing services with database connections.
package bootstrap

import (
	"context"
	"fmt"
	"log"

	"github.com/cobrun/cobrun-platform/pkg/config"
	"github.com/cobrun/cobrun-platform/pkg/database"
)

// Service holds all initialized components for a microservice.
type Service struct {
	Config      *config.Config
	Connections *database.Connections
}

// Options configures which databases to connect to.
type Options struct {
	// Enable specific database connections
	UseSQL    bool
	UseCosmos bool
	UseRedis  bool

	// Run migrations on startup (SQL only)
	RunMigrations bool
}

// DefaultOptions returns options that enable all databases.
func DefaultOptions() Options {
	return Options{
		UseSQL:        true,
		UseCosmos:     true,
		UseRedis:      true,
		RunMigrations: false,
	}
}

// Initialize sets up a service with configuration from Key Vault (production)
// or environment variables (development).
func Initialize(ctx context.Context, serviceName string, opts Options) (*Service, error) {
	// Load configuration (automatically uses Key Vault in production)
	cfg, err := config.Load(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	log.Printf("Starting %s in %s mode", serviceName, cfg.Environment)
	log.Printf("Key Vault: %s", valueOrNone(cfg.KeyVaultName))

	// Create database connection config from main config
	dbConfig := &database.ConnectionConfig{
		MaxRetries:  5,
		ConnTimeout: 30,
	}

	// Configure SQL if enabled
	if opts.UseSQL && cfg.SQLConnectionString != "" {
		dbConfig.SQLConnString = cfg.SQLConnectionString
		dbConfig.SQLUseMSI = cfg.IsProduction()
		log.Printf("SQL Server: connection string configured")
	}

	// Configure Cosmos DB if enabled
	if opts.UseCosmos && cfg.CosmosDBEndpoint != "" {
		dbConfig.CosmosEndpoint = cfg.CosmosDBEndpoint
		dbConfig.CosmosDatabase = cfg.CosmosDBDatabase
		log.Printf("Cosmos DB: %s/%s", cfg.CosmosDBEndpoint, cfg.CosmosDBDatabase)
	}

	// Configure Redis if enabled
	if opts.UseRedis && cfg.RedisHost != "" {
		dbConfig.RedisHost = cfg.RedisHost
		dbConfig.RedisPassword = cfg.RedisPassword
		log.Printf("Redis: %s", cfg.RedisHost)
	}

	// Create database connections
	conns, err := database.NewConnections(ctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connections: %w", err)
	}

	return &Service{
		Config:      cfg,
		Connections: conns,
	}, nil
}

// MustInitialize initializes the service and panics on error.
func MustInitialize(ctx context.Context, serviceName string, opts Options) *Service {
	svc, err := Initialize(ctx, serviceName, opts)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize service: %v", err))
	}
	return svc
}

// Close cleans up all resources.
func (s *Service) Close() {
	if s.Connections != nil {
		s.Connections.Close()
	}
}

func valueOrNone(s string) string {
	if s == "" {
		return "(none - using env vars)"
	}
	return s
}





