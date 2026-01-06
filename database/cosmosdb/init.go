// Package cosmosdb provides Cosmos DB initialization and container setup utilities.
package cosmosdb

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// ContainerConfig defines configuration for a Cosmos DB container.
type ContainerConfig struct {
	Name             string
	PartitionKeyPath string
	UniqueKeys       []string
	IndexingPolicy   *azcosmos.IndexingPolicy
	TTLSeconds       int32
}

// DatabaseConfig holds the database configuration.
type DatabaseConfig struct {
	DatabaseName string
	Containers   []ContainerConfig
}

// DefaultDatabaseConfig returns the default Cobrun database configuration.
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		DatabaseName: "cobrun",
		Containers: []ContainerConfig{
			{
				Name:             "users",
				PartitionKeyPath: "/id",
				IndexingPolicy:   defaultIndexingPolicy(),
				UniqueKeys:       []string{"/email"},
			},
			{
				Name:             "verifications",
				PartitionKeyPath: "/user_id",
				TTLSeconds:       3600, // 1 hour TTL for verifications
				IndexingPolicy:   defaultIndexingPolicy(),
			},
			{
				Name:             "riders",
				PartitionKeyPath: "/user_id",
				IndexingPolicy:   defaultIndexingPolicy(),
			},
			{
				Name:             "drivers",
				PartitionKeyPath: "/user_id",
				IndexingPolicy:   defaultIndexingPolicy(),
			},
			{
				Name:             "trips",
				PartitionKeyPath: "/rider_id",
				IndexingPolicy:   defaultIndexingPolicy(),
			},
			{
				Name:             "ride_requests",
				PartitionKeyPath: "/rider_id",
				IndexingPolicy:   defaultIndexingPolicy(),
			},
			{
				Name:             "driver_offers",
				PartitionKeyPath: "/driver_id",
				TTLSeconds:       300, // 5 minutes TTL for offers
				IndexingPolicy:   defaultIndexingPolicy(),
			},
			{
				Name:             "driver_locations",
				PartitionKeyPath: "/driver_id",
				TTLSeconds:       86400, // 24 hours TTL for locations
				IndexingPolicy:   locationIndexingPolicy(),
			},
			{
				Name:             "location_history",
				PartitionKeyPath: "/driver_id",
				TTLSeconds:       604800, // 7 days TTL
				IndexingPolicy:   defaultIndexingPolicy(),
			},
			{
				Name:             "events",
				PartitionKeyPath: "/type",
				TTLSeconds:       2592000, // 30 days TTL
				IndexingPolicy:   eventIndexingPolicy(),
			},
		},
	}
}

// defaultIndexingPolicy returns the default indexing policy.
func defaultIndexingPolicy() *azcosmos.IndexingPolicy {
	return &azcosmos.IndexingPolicy{
		IndexingMode: azcosmos.IndexingModeConsistent,
		Automatic:    true,
		IncludedPaths: []azcosmos.IncludedPath{
			{Path: "/*"},
		},
		ExcludedPaths: []azcosmos.ExcludedPath{
			{Path: "/_etag/?"},
		},
	}
}

// locationIndexingPolicy returns an optimized indexing policy for location data.
func locationIndexingPolicy() *azcosmos.IndexingPolicy {
	return &azcosmos.IndexingPolicy{
		IndexingMode: azcosmos.IndexingModeConsistent,
		Automatic:    true,
		IncludedPaths: []azcosmos.IncludedPath{
			{Path: "/driver_id/?"},
			{Path: "/status/?"},
			{Path: "/city/?"},
			{Path: "/geohash/?"},
			{Path: "/updated_at/?"},
			{Path: "/location/?"},
		},
		ExcludedPaths: []azcosmos.ExcludedPath{
			{Path: "/*"},
			{Path: "/_etag/?"},
		},
	}
}

// eventIndexingPolicy returns an optimized indexing policy for events.
func eventIndexingPolicy() *azcosmos.IndexingPolicy {
	return &azcosmos.IndexingPolicy{
		IndexingMode: azcosmos.IndexingModeConsistent,
		Automatic:    true,
		IncludedPaths: []azcosmos.IncludedPath{
			{Path: "/type/?"},
			{Path: "/entity_id/?"},
			{Path: "/entity_type/?"},
			{Path: "/timestamp/?"},
		},
		ExcludedPaths: []azcosmos.ExcludedPath{
			{Path: "/*"},
			{Path: "/_etag/?"},
		},
	}
}

// Initializer handles Cosmos DB initialization.
type Initializer struct {
	client *azcosmos.Client
	config *DatabaseConfig
}

// NewInitializer creates a new Cosmos DB initializer.
func NewInitializer(client *azcosmos.Client, config *DatabaseConfig) *Initializer {
	if config == nil {
		config = DefaultDatabaseConfig()
	}
	return &Initializer{
		client: client,
		config: config,
	}
}

// Initialize creates the database and all containers if they don't exist.
func (i *Initializer) Initialize(ctx context.Context) error {
	// Create database
	if err := i.createDatabase(ctx); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Create containers
	for _, container := range i.config.Containers {
		if err := i.createContainer(ctx, container); err != nil {
			return fmt.Errorf("failed to create container %s: %w", container.Name, err)
		}
	}

	log.Printf("Cosmos DB initialization completed for database: %s", i.config.DatabaseName)
	return nil
}

// createDatabase creates the database if it doesn't exist.
func (i *Initializer) createDatabase(ctx context.Context) error {
	props := azcosmos.DatabaseProperties{
		ID: i.config.DatabaseName,
	}

	throughput := azcosmos.NewManualThroughputProperties(400) // 400 RU/s

	_, err := i.client.CreateDatabase(ctx, props, &azcosmos.CreateDatabaseOptions{
		ThroughputProperties: &throughput,
	})
	if err != nil {
		// Ignore if already exists
		if isConflictError(err) {
			log.Printf("Database %s already exists", i.config.DatabaseName)
			return nil
		}
		return err
	}

	log.Printf("Created database: %s", i.config.DatabaseName)
	return nil
}

// createContainer creates a container if it doesn't exist.
func (i *Initializer) createContainer(ctx context.Context, config ContainerConfig) error {
	database, err := i.client.NewDatabase(i.config.DatabaseName)
	if err != nil {
		return err
	}

	props := azcosmos.ContainerProperties{
		ID: config.Name,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{config.PartitionKeyPath},
		},
	}

	// Set indexing policy if provided
	if config.IndexingPolicy != nil {
		props.IndexingPolicy = config.IndexingPolicy
	}

	// Set TTL if provided
	if config.TTLSeconds > 0 {
		props.DefaultTimeToLive = to(config.TTLSeconds)
	}

	// Set unique keys if provided
	if len(config.UniqueKeys) > 0 {
		uniqueKeys := make([]azcosmos.UniqueKey, len(config.UniqueKeys))
		for j, key := range config.UniqueKeys {
			uniqueKeys[j] = azcosmos.UniqueKey{
				Paths: []string{key},
			}
		}
		props.UniqueKeyPolicy = &azcosmos.UniqueKeyPolicy{
			UniqueKeys: uniqueKeys,
		}
	}

	_, err = database.CreateContainer(ctx, props, nil)
	if err != nil {
		// Ignore if already exists
		if isConflictError(err) {
			log.Printf("Container %s already exists", config.Name)
			return nil
		}
		return err
	}

	log.Printf("Created container: %s (partition key: %s)", config.Name, config.PartitionKeyPath)
	return nil
}

// isConflictError checks if the error is a conflict (already exists) error.
func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	// Check for 409 Conflict status code by checking error message
	errMsg := err.Error()
	return strings.Contains(errMsg, "409") || strings.Contains(errMsg, "Conflict") || strings.Contains(errMsg, "already exists")
}

// Helper function to get pointer to value
func to[T any](v T) *T {
	return &v
}

