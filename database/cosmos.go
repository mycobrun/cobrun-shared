// Package database provides database client utilities.
package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// CosmosConfig holds Cosmos DB configuration.
type CosmosConfig struct {
	Endpoint     string
	DatabaseName string
	// Key is optional - if empty, uses managed identity
	Key string
}

// CosmosClient wraps the Azure Cosmos DB client.
type CosmosClient struct {
	client   *azcosmos.Client
	database *azcosmos.DatabaseClient
	config   CosmosConfig
}

// NewCosmosClient creates a new Cosmos DB client.
func NewCosmosClient(ctx context.Context, config CosmosConfig) (*CosmosClient, error) {
	var client *azcosmos.Client
	var err error

	if config.Key != "" {
		// Use key-based authentication
		cred, err := azcosmos.NewKeyCredential(config.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to create key credential: %w", err)
		}
		client, err = azcosmos.NewClientWithKey(config.Endpoint, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create cosmos client with key: %w", err)
		}
	} else {
		// Use managed identity
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create default credential: %w", err)
		}
		client, err = azcosmos.NewClient(config.Endpoint, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create cosmos client: %w", err)
		}
	}

	database, err := client.NewDatabase(config.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return &CosmosClient{
		client:   client,
		database: database,
		config:   config,
	}, nil
}

// Container returns a container client.
func (c *CosmosClient) Container(name string) (*CosmosContainer, error) {
	container, err := c.database.NewContainer(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get container %s: %w", name, err)
	}
	return &CosmosContainer{container: container}, nil
}

// CosmosContainer wraps a Cosmos DB container.
type CosmosContainer struct {
	container *azcosmos.ContainerClient
}

// Create creates a new item in the container.
func (c *CosmosContainer) Create(ctx context.Context, partitionKey string, item interface{}) error {
	pk := azcosmos.NewPartitionKeyString(partitionKey)

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = c.container.CreateItem(ctx, pk, data, nil)
	if err != nil {
		return fmt.Errorf("failed to create item: %w", err)
	}

	return nil
}

// Read reads an item from the container.
func (c *CosmosContainer) Read(ctx context.Context, partitionKey, id string, result interface{}) error {
	pk := azcosmos.NewPartitionKeyString(partitionKey)

	resp, err := c.container.ReadItem(ctx, pk, id, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return ErrNotFound
		}
		return fmt.Errorf("failed to read item: %w", err)
	}

	if err := json.Unmarshal(resp.Value, result); err != nil {
		return fmt.Errorf("failed to unmarshal item: %w", err)
	}

	return nil
}

// Replace replaces an item in the container.
func (c *CosmosContainer) Replace(ctx context.Context, partitionKey, id string, item interface{}) error {
	pk := azcosmos.NewPartitionKeyString(partitionKey)

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = c.container.ReplaceItem(ctx, pk, id, data, nil)
	if err != nil {
		return fmt.Errorf("failed to replace item: %w", err)
	}

	return nil
}

// Upsert creates or replaces an item in the container.
func (c *CosmosContainer) Upsert(ctx context.Context, partitionKey string, item interface{}) error {
	pk := azcosmos.NewPartitionKeyString(partitionKey)

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = c.container.UpsertItem(ctx, pk, data, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert item: %w", err)
	}

	return nil
}

// Delete deletes an item from the container.
func (c *CosmosContainer) Delete(ctx context.Context, partitionKey, id string) error {
	pk := azcosmos.NewPartitionKeyString(partitionKey)

	_, err := c.container.DeleteItem(ctx, pk, id, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return nil // Item already doesn't exist
		}
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}

// Query executes a query against the container.
func (c *CosmosContainer) Query(ctx context.Context, partitionKey, query string, params []QueryParam, results interface{}) error {
	pk := azcosmos.NewPartitionKeyString(partitionKey)

	queryOptions := &azcosmos.QueryOptions{}
	for _, p := range params {
		queryOptions.QueryParameters = append(queryOptions.QueryParameters, azcosmos.QueryParameter{
			Name:  p.Name,
			Value: p.Value,
		})
	}

	pager := c.container.NewQueryItemsPager(query, pk, queryOptions)

	var items []json.RawMessage
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to get query results: %w", err)
		}
		for _, item := range resp.Items {
			items = append(items, item)
		}
	}

	// Marshal all items into array and unmarshal into results
	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	if err := json.Unmarshal(data, results); err != nil {
		return fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return nil
}

// QueryCrossPartition executes a cross-partition query.
func (c *CosmosContainer) QueryCrossPartition(ctx context.Context, query string, params []QueryParam, results interface{}) error {
	queryOptions := &azcosmos.QueryOptions{}
	for _, p := range params {
		queryOptions.QueryParameters = append(queryOptions.QueryParameters, azcosmos.QueryParameter{
			Name:  p.Name,
			Value: p.Value,
		})
	}

	pager := c.container.NewQueryItemsPager(query, azcosmos.PartitionKey{}, queryOptions)

	var items []json.RawMessage
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to get query results: %w", err)
		}
		for _, item := range resp.Items {
			items = append(items, item)
		}
	}

	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	if err := json.Unmarshal(data, results); err != nil {
		return fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return nil
}

// QueryParam represents a query parameter.
type QueryParam struct {
	Name  string
	Value interface{}
}

// CosmosDocument represents a base document with common fields.
type CosmosDocument struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	TTL       *int      `json:"ttl,omitempty"` // Time to live in seconds
}

// Common errors.
var (
	ErrNotFound = errors.New("item not found")
)

// Ping checks if the connection is healthy.
func (c *CosmosClient) Ping(ctx context.Context) error {
	// Try to read database properties as a health check
	_, err := c.database.Read(ctx, nil)
	if err != nil {
		return fmt.Errorf("cosmos health check failed: %w", err)
	}
	return nil
}

// Close closes the client (no-op for Cosmos, but good for interface consistency).
func (c *CosmosClient) Close() error {
	// Cosmos SDK doesn't require explicit close
	return nil
}

// Retry-enabled operations for production resilience

// CreateWithRetry creates a new item with retry logic.
func (c *CosmosContainer) CreateWithRetry(ctx context.Context, partitionKey string, item interface{}) error {
	return RetryCosmosOperation(ctx, func() error {
		return c.Create(ctx, partitionKey, item)
	})
}

// ReadWithRetry reads an item with retry logic.
func (c *CosmosContainer) ReadWithRetry(ctx context.Context, partitionKey, id string, result interface{}) error {
	return RetryCosmosOperation(ctx, func() error {
		return c.Read(ctx, partitionKey, id, result)
	})
}

// ReplaceWithRetry replaces an item with retry logic.
func (c *CosmosContainer) ReplaceWithRetry(ctx context.Context, partitionKey, id string, item interface{}) error {
	return RetryCosmosOperation(ctx, func() error {
		return c.Replace(ctx, partitionKey, id, item)
	})
}

// UpsertWithRetry creates or replaces an item with retry logic.
func (c *CosmosContainer) UpsertWithRetry(ctx context.Context, partitionKey string, item interface{}) error {
	return RetryCosmosOperation(ctx, func() error {
		return c.Upsert(ctx, partitionKey, item)
	})
}

// DeleteWithRetry deletes an item with retry logic.
func (c *CosmosContainer) DeleteWithRetry(ctx context.Context, partitionKey, id string) error {
	return RetryCosmosOperation(ctx, func() error {
		return c.Delete(ctx, partitionKey, id)
	})
}

// QueryWithRetry executes a query with retry logic.
func (c *CosmosContainer) QueryWithRetry(ctx context.Context, partitionKey, query string, params []QueryParam, results interface{}) error {
	return RetryCosmosOperation(ctx, func() error {
		return c.Query(ctx, partitionKey, query, params, results)
	})
}

// QueryCrossPartitionWithRetry executes a cross-partition query with retry logic.
func (c *CosmosContainer) QueryCrossPartitionWithRetry(ctx context.Context, query string, params []QueryParam, results interface{}) error {
	return RetryCosmosOperation(ctx, func() error {
		return c.QueryCrossPartition(ctx, query, params, results)
	})
}

// Convenience methods for backward compatibility and easier usage

// GetItem retrieves a single item by ID.
func (c *CosmosClient) GetItem(ctx context.Context, database, containerName, partitionKey, id string, result interface{}) error {
	container, err := c.Container(containerName)
	if err != nil {
		return err
	}
	return container.Read(ctx, partitionKey, id, result)
}

// UpsertItem creates or updates an item.
func (c *CosmosClient) UpsertItem(ctx context.Context, database, containerName, partitionKey string, item interface{}) error {
	container, err := c.Container(containerName)
	if err != nil {
		return err
	}
	return container.Upsert(ctx, partitionKey, item)
}

// QueryItems executes a parameterized query against a container.
// This is the SAFE way to query - always use parameterized queries.
// Example:
//
//	query := "SELECT * FROM c WHERE c.user_id = @userId"
//	params := []QueryParam{{Name: "@userId", Value: userID}}
//	err := client.QueryItems(ctx, "db", "container", query, params, &results)
func (c *CosmosClient) QueryItems(ctx context.Context, database, containerName, query string, params []QueryParam, results interface{}) error {
	container, err := c.Container(containerName)
	if err != nil {
		return err
	}
	// Use cross-partition query for flexibility
	return container.QueryCrossPartition(ctx, query, params, results)
}

// QueryItemsWithPartition executes a parameterized query within a specific partition.
func (c *CosmosClient) QueryItemsWithPartition(ctx context.Context, database, containerName, partitionKey, query string, params []QueryParam, results interface{}) error {
	container, err := c.Container(containerName)
	if err != nil {
		return err
	}
	return container.Query(ctx, partitionKey, query, params, results)
}

// DeleteItem deletes an item by ID.
func (c *CosmosClient) DeleteItem(ctx context.Context, database, containerName, partitionKey, id string) error {
	container, err := c.Container(containerName)
	if err != nil {
		return err
	}
	return container.Delete(ctx, partitionKey, id)
}
