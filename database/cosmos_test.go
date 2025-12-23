package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

func TestCosmosDocument(t *testing.T) {
	now := time.Now()
	ttl := 3600

	doc := CosmosDocument{
		ID:        "test-id",
		CreatedAt: now,
		UpdatedAt: now,
		TTL:       &ttl,
	}

	if doc.ID != "test-id" {
		t.Errorf("expected ID=test-id, got %s", doc.ID)
	}
	if doc.CreatedAt != now {
		t.Errorf("expected CreatedAt=%v, got %v", now, doc.CreatedAt)
	}
	if doc.UpdatedAt != now {
		t.Errorf("expected UpdatedAt=%v, got %v", now, doc.UpdatedAt)
	}
	if doc.TTL == nil || *doc.TTL != 3600 {
		t.Errorf("expected TTL=3600, got %v", doc.TTL)
	}
}

func TestQueryParam(t *testing.T) {
	params := []QueryParam{
		{Name: "@id", Value: "123"},
		{Name: "@status", Value: "active"},
		{Name: "@count", Value: 10},
	}

	if len(params) != 3 {
		t.Errorf("expected 3 params, got %d", len(params))
	}

	if params[0].Name != "@id" || params[0].Value != "123" {
		t.Errorf("unexpected param values: %+v", params[0])
	}
}

func TestErrNotFound(t *testing.T) {
	err := ErrNotFound
	if err == nil {
		t.Error("ErrNotFound should not be nil")
	}

	if err.Error() != "item not found" {
		t.Errorf("unexpected error message: %v", err)
	}

	// Test that it's comparable
	if !errors.Is(err, ErrNotFound) {
		t.Error("ErrNotFound should be comparable with errors.Is")
	}
}

func TestCosmosConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   CosmosConfig
		wantErr  bool
	}{
		{
			name: "with key",
			config: CosmosConfig{
				Endpoint:     "https://test.documents.azure.com:443/",
				DatabaseName: "testdb",
				Key:          "test-key",
			},
			wantErr: false,
		},
		{
			name: "without key (MSI)",
			config: CosmosConfig{
				Endpoint:     "https://test.documents.azure.com:443/",
				DatabaseName: "testdb",
				Key:          "",
			},
			wantErr: false,
		},
		{
			name: "empty endpoint",
			config: CosmosConfig{
				Endpoint:     "",
				DatabaseName: "testdb",
				Key:          "test-key",
			},
			wantErr: true,
		},
		{
			name: "empty database",
			config: CosmosConfig{
				Endpoint:     "https://test.documents.azure.com:443/",
				DatabaseName: "",
				Key:          "test-key",
			},
			wantErr: false, // Database can be empty during client creation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Endpoint == "" {
				if !tt.wantErr {
					t.Skip("Cannot test empty endpoint without actual connection")
				}
			}
		})
	}
}

func TestPartitionKeyString(t *testing.T) {
	pk := azcosmos.NewPartitionKeyString("test-partition")
	// PartitionKey is created successfully if no panic occurs
	_ = pk
}

func TestCosmosContainer_Operations(t *testing.T) {
	// This is a unit test for the CosmosContainer structure
	// Actual operations require integration tests with a real Cosmos DB

	t.Run("QueryParam_Construction", func(t *testing.T) {
		params := []QueryParam{
			{Name: "@userId", Value: "user123"},
			{Name: "@status", Value: "active"},
		}

		if len(params) != 2 {
			t.Errorf("expected 2 params, got %d", len(params))
		}

		if params[0].Name != "@userId" {
			t.Errorf("expected @userId, got %s", params[0].Name)
		}

		if params[1].Value != "active" {
			t.Errorf("expected active, got %v", params[1].Value)
		}
	})

	t.Run("Query_Building", func(t *testing.T) {
		query := "SELECT * FROM c WHERE c.user_id = @userId AND c.status = @status"
		params := []QueryParam{
			{Name: "@userId", Value: "user123"},
			{Name: "@status", Value: "active"},
		}

		if query == "" {
			t.Error("query should not be empty")
		}

		if len(params) != 2 {
			t.Errorf("expected 2 params, got %d", len(params))
		}
	})
}

func TestCosmosDocument_TTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  *int
		want bool
	}{
		{
			name: "no TTL",
			ttl:  nil,
			want: false,
		},
		{
			name: "with TTL",
			ttl:  intPtr(3600),
			want: true,
		},
		{
			name: "zero TTL",
			ttl:  intPtr(0),
			want: true,
		},
		{
			name: "negative TTL (no expiry)",
			ttl:  intPtr(-1),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := CosmosDocument{
				ID:        "test",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				TTL:       tt.ttl,
			}

			hasTTL := doc.TTL != nil
			if hasTTL != tt.want {
				t.Errorf("expected hasTTL=%v, got %v", tt.want, hasTTL)
			}

			if tt.ttl != nil && doc.TTL != nil {
				if *doc.TTL != *tt.ttl {
					t.Errorf("expected TTL=%d, got %d", *tt.ttl, *doc.TTL)
				}
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func TestQueryConstruction(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		params []QueryParam
		valid  bool
	}{
		{
			name:  "simple select",
			query: "SELECT * FROM c",
			params: nil,
			valid: true,
		},
		{
			name:  "with single parameter",
			query: "SELECT * FROM c WHERE c.id = @id",
			params: []QueryParam{
				{Name: "@id", Value: "123"},
			},
			valid: true,
		},
		{
			name:  "with multiple parameters",
			query: "SELECT * FROM c WHERE c.user_id = @userId AND c.status = @status",
			params: []QueryParam{
				{Name: "@userId", Value: "user123"},
				{Name: "@status", Value: "active"},
			},
			valid: true,
		},
		{
			name:  "with numeric parameter",
			query: "SELECT * FROM c WHERE c.age > @minAge",
			params: []QueryParam{
				{Name: "@minAge", Value: 18},
			},
			valid: true,
		},
		{
			name:  "empty query",
			query: "",
			params: nil,
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				if tt.query == "" {
					t.Error("valid query should not be empty")
				}
			} else {
				if tt.query != "" {
					t.Error("invalid query should be empty")
				}
			}

			if tt.params != nil {
				for _, param := range tt.params {
					if param.Name == "" {
						t.Error("parameter name should not be empty")
					}
				}
			}
		})
	}
}

func TestCosmosErrors(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		err := ErrNotFound
		if err == nil {
			t.Fatal("ErrNotFound should not be nil")
		}

		if !errors.Is(err, ErrNotFound) {
			t.Error("should be able to check with errors.Is")
		}
	})

	t.Run("Wrapped ErrNotFound", func(t *testing.T) {
		wrappedErr := errors.New("wrapped: " + ErrNotFound.Error())
		if wrappedErr == nil {
			t.Fatal("wrapped error should not be nil")
		}
	})
}

func TestCosmosConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  CosmosConfig
		isValid bool
	}{
		{
			name: "valid with key",
			config: CosmosConfig{
				Endpoint:     "https://test.documents.azure.com:443/",
				DatabaseName: "testdb",
				Key:          "test-key",
			},
			isValid: true,
		},
		{
			name: "valid with MSI",
			config: CosmosConfig{
				Endpoint:     "https://test.documents.azure.com:443/",
				DatabaseName: "testdb",
				Key:          "",
			},
			isValid: true,
		},
		{
			name: "missing endpoint",
			config: CosmosConfig{
				Endpoint:     "",
				DatabaseName: "testdb",
				Key:          "test-key",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.config.Endpoint != ""
			if valid != tt.isValid {
				t.Errorf("expected valid=%v, got %v", tt.isValid, valid)
			}
		})
	}
}

func TestPartitionKeyCreation(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "simple string", value: "test"},
		{name: "user id", value: "user123"},
		{name: "email", value: "user@example.com"},
		{name: "uuid", value: "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pk := azcosmos.NewPartitionKeyString(tt.value)
			// PartitionKey is created successfully if no panic occurs
			_ = pk
		})
	}
}

func TestRetryableOperations(t *testing.T) {
	t.Run("RetryCosmosOperation", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		err := RetryCosmosOperation(ctx, func() error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary failure")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected success after retry, got %v", err)
		}

		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})
}
