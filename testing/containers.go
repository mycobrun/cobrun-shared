// Package testing provides test utilities and helpers.
package testing

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer provides a Redis container for testing.
type RedisContainer struct {
	*redis.RedisContainer
	ConnectionString string
}

// StartRedisContainer starts a Redis container for integration tests.
func StartRedisContainer(ctx context.Context) (*RedisContainer, error) {
	container, err := redis.Run(ctx,
		"redis:7-alpine",
		redis.WithLogLevel(redis.LogLevelNotice),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start Redis container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Redis connection string: %w", err)
	}

	return &RedisContainer{
		RedisContainer:   container,
		ConnectionString: connStr,
	}, nil
}

// MongoDBContainer provides a MongoDB container for testing.
type MongoDBContainer struct {
	*mongodb.MongoDBContainer
	ConnectionString string
}

// StartMongoDBContainer starts a MongoDB container for integration tests.
// This can be used to emulate CosmosDB for local testing.
func StartMongoDBContainer(ctx context.Context) (*MongoDBContainer, error) {
	container, err := mongodb.Run(ctx,
		"mongo:7.0",
		mongodb.WithUsername("test"),
		mongodb.WithPassword("test"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start MongoDB container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MongoDB connection string: %w", err)
	}

	return &MongoDBContainer{
		MongoDBContainer: container,
		ConnectionString: connStr,
	}, nil
}

// SQLServerContainer provides a SQL Server container for testing.
type SQLServerContainer struct {
	testcontainers.Container
	ConnectionString string
}

// StartSQLServerContainer starts a SQL Server container for integration tests.
func StartSQLServerContainer(ctx context.Context) (*SQLServerContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mcr.microsoft.com/mssql/server:2022-latest",
		ExposedPorts: []string{"1433/tcp"},
		Env: map[string]string{
			"ACCEPT_EULA":       "Y",
			"MSSQL_SA_PASSWORD": "YourStrong!Passw0rd",
			"MSSQL_PID":         "Developer",
		},
		WaitingFor: wait.ForLog("SQL Server is now ready for client connections").
			WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start SQL Server container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get SQL Server host: %w", err)
	}

	port, err := container.MappedPort(ctx, "1433")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get SQL Server port: %w", err)
	}

	connStr := fmt.Sprintf("sqlserver://sa:YourStrong!Passw0rd@%s:%s?database=master&encrypt=disable", host, port.Port())

	return &SQLServerContainer{
		Container:        container,
		ConnectionString: connStr,
	}, nil
}

// AzuriteContainer provides an Azurite container for Azure Storage emulation.
type AzuriteContainer struct {
	testcontainers.Container
	BlobEndpoint  string
	QueueEndpoint string
	TableEndpoint string
	ConnectionString string
}

// StartAzuriteContainer starts an Azurite container for Azure Storage integration tests.
func StartAzuriteContainer(ctx context.Context) (*AzuriteContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mcr.microsoft.com/azure-storage/azurite:latest",
		ExposedPorts: []string{"10000/tcp", "10001/tcp", "10002/tcp"},
		WaitingFor:   wait.ForListeningPort("10000/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Azurite container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Azurite host: %w", err)
	}

	blobPort, _ := container.MappedPort(ctx, "10000")
	queuePort, _ := container.MappedPort(ctx, "10001")
	tablePort, _ := container.MappedPort(ctx, "10002")

	// Default Azurite credentials
	accountName := "devstoreaccount1"
	accountKey := "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="

	connStr := fmt.Sprintf(
		"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=http://%s:%s/%s;QueueEndpoint=http://%s:%s/%s;TableEndpoint=http://%s:%s/%s;",
		accountName, accountKey,
		host, blobPort.Port(), accountName,
		host, queuePort.Port(), accountName,
		host, tablePort.Port(), accountName,
	)

	return &AzuriteContainer{
		Container:        container,
		BlobEndpoint:     fmt.Sprintf("http://%s:%s/%s", host, blobPort.Port(), accountName),
		QueueEndpoint:    fmt.Sprintf("http://%s:%s/%s", host, queuePort.Port(), accountName),
		TableEndpoint:    fmt.Sprintf("http://%s:%s/%s", host, tablePort.Port(), accountName),
		ConnectionString: connStr,
	}, nil
}

// Terminate terminates the container.
func (c *SQLServerContainer) Terminate(ctx context.Context) error {
	return c.Container.Terminate(ctx)
}

// Terminate terminates the container.
func (c *AzuriteContainer) Terminate(ctx context.Context) error {
	return c.Container.Terminate(ctx)
}

// ContainerCleanup provides a cleanup function for t.Cleanup.
type ContainerCleanup interface {
	Terminate(ctx context.Context) error
}

// CleanupContainer returns a cleanup function for testing.T.Cleanup.
func CleanupContainer(ctx context.Context, c ContainerCleanup) func() {
	return func() {
		if err := c.Terminate(ctx); err != nil {
			// Log but don't fail on cleanup
			fmt.Printf("failed to terminate container: %v\n", err)
		}
	}
}
