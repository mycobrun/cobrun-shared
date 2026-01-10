// Package database provides database client utilities.
package database

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
// Returns ErrConflict if an item with the same ID already exists.
func (c *CosmosContainer) Create(ctx context.Context, partitionKey string, item interface{}) error {
	pk := azcosmos.NewPartitionKeyString(partitionKey)

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = c.container.CreateItem(ctx, pk, data, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 409 {
			return ErrConflict
		}
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
// Note: The azcosmos SDK v0.3.6 requires a partition key for queries.
// This function attempts to extract the partition key from common query parameters.
// If no partition key is found, it signals the caller to use QueryAllPartitions instead.
func (c *CosmosContainer) QueryCrossPartition(ctx context.Context, query string, params []QueryParam, results interface{}) error {
	queryOptions := &azcosmos.QueryOptions{}
	for _, p := range params {
		queryOptions.QueryParameters = append(queryOptions.QueryParameters, azcosmos.QueryParameter{
			Name:  p.Name,
			Value: p.Value,
		})
	}

	// Try to extract partition key value from common query parameters
	var partitionKeyValue string
	for _, p := range params {
		if p.Name == "@email" || p.Name == "@user_id" || p.Name == "@userId" || p.Name == "@id" {
			if v, ok := p.Value.(string); ok && v != "" {
				partitionKeyValue = v
				break
			}
		}
	}

	// If we found a partition key value, use partition-scoped query
	if partitionKeyValue != "" {
		return c.Query(ctx, partitionKeyValue, query, params, results)
	}

	// For queries without a recognizable partition key, try with an empty partition key
	// This may return empty results for containers that don't have empty partition keys
	// Note: This is a limitation of the azcosmos SDK v0.3.6 which doesn't support
	// true cross-partition queries. For cross-partition queries, use CosmosClient.QueryAllPartitions
	pk := azcosmos.NewPartitionKeyString("")
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
	ErrConflict = errors.New("item already exists")
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

// CreateItem creates a new item. Returns ErrConflict if an item with the same ID already exists.
func (c *CosmosClient) CreateItem(ctx context.Context, database, containerName, partitionKey string, item interface{}) error {
	container, err := c.Container(containerName)
	if err != nil {
		return err
	}
	return container.Create(ctx, partitionKey, item)
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
	// Check if any recognizable partition key parameter is present
	hasPartitionKey := false
	for _, p := range params {
		if p.Name == "@email" || p.Name == "@user_id" || p.Name == "@userId" || p.Name == "@id" {
			if v, ok := p.Value.(string); ok && v != "" {
				hasPartitionKey = true
				break
			}
		}
	}

	// If no partition key parameter, use REST API for true cross-partition query
	if !hasPartitionKey && c.config.Key != "" {
		return c.QueryAllPartitions(ctx, containerName, query, params, results)
	}

	container, err := c.Container(containerName)
	if err != nil {
		return err
	}
	// Use cross-partition query for flexibility
	return container.QueryCrossPartition(ctx, query, params, results)
}

// QueryAllPartitions executes a true cross-partition query using the REST API.
// This bypasses the SDK's limitation of requiring a partition key for queries.
// Note: This only works when using key-based authentication (not managed identity).
func (c *CosmosClient) QueryAllPartitions(ctx context.Context, containerName, query string, params []QueryParam, results interface{}) error {
	if c.config.Key == "" {
		return fmt.Errorf("QueryAllPartitions requires key-based authentication")
	}

	// Build the query body
	queryBody := map[string]interface{}{
		"query": query,
	}

	if len(params) > 0 {
		parameters := make([]map[string]interface{}, 0, len(params))
		for _, p := range params {
			parameters = append(parameters, map[string]interface{}{
				"name":  p.Name,
				"value": p.Value,
			})
		}
		queryBody["parameters"] = parameters
	}

	bodyBytes, err := json.Marshal(queryBody)
	if err != nil {
		return fmt.Errorf("failed to marshal query body: %w", err)
	}

	// Build the URL
	resourceLink := fmt.Sprintf("dbs/%s/colls/%s", c.config.DatabaseName, containerName)
	queryURL := fmt.Sprintf("%s/%s/docs", c.config.Endpoint, resourceLink)

	var allDocuments []json.RawMessage
	var continuationToken string

	// Paginate through all results
	for {
		req, err := http.NewRequestWithContext(ctx, "POST", queryURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Generate auth header
		dateStr := time.Now().UTC().Format(http.TimeFormat)
		authHeader := c.generateAuthHeader("POST", "docs", resourceLink, dateStr)

		// Set required headers
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("x-ms-date", dateStr)
		req.Header.Set("x-ms-version", "2018-12-31")
		req.Header.Set("Content-Type", "application/query+json")
		req.Header.Set("x-ms-documentdb-isquery", "true")
		req.Header.Set("x-ms-documentdb-query-enablecrosspartition", "true")

		if continuationToken != "" {
			req.Header.Set("x-ms-continuation", continuationToken)
		}

		// Execute request
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		// Parse response
		var queryResp struct {
			Documents []json.RawMessage `json:"Documents"`
			Count     int               `json:"_count"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		allDocuments = append(allDocuments, queryResp.Documents...)

		// Check for continuation
		continuationToken = resp.Header.Get("x-ms-continuation")
		if continuationToken == "" {
			break
		}
	}

	// Marshal all documents into the results
	data, err := json.Marshal(allDocuments)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := json.Unmarshal(data, results); err != nil {
		return fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return nil
}

// generateAuthHeader generates the Cosmos DB authorization header.
func (c *CosmosClient) generateAuthHeader(verb, resourceType, resourceLink, dateStr string) string {
	// Decode the master key
	key, err := base64.StdEncoding.DecodeString(c.config.Key)
	if err != nil {
		return ""
	}

	// Build the string to sign
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n",
		strings.ToLower(verb),
		strings.ToLower(resourceType),
		resourceLink,
		strings.ToLower(dateStr),
		"",
	)

	// Generate HMAC signature
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Build auth header
	return url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", signature))
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
