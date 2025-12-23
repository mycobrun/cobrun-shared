package cosmosdb

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

func TestDefaultDatabaseConfig(t *testing.T) {
	config := DefaultDatabaseConfig()

	if config == nil {
		t.Fatal("DefaultDatabaseConfig should not return nil")
	}

	if config.DatabaseName != "cobrun" {
		t.Errorf("expected DatabaseName=cobrun, got %s", config.DatabaseName)
	}

	if len(config.Containers) == 0 {
		t.Error("DefaultDatabaseConfig should have containers configured")
	}

	expectedContainers := []string{
		"trips",
		"ride_requests",
		"driver_offers",
		"driver_locations",
		"location_history",
		"events",
	}

	if len(config.Containers) != len(expectedContainers) {
		t.Errorf("expected %d containers, got %d", len(expectedContainers), len(config.Containers))
	}

	// Verify each expected container exists
	containerMap := make(map[string]bool)
	for _, container := range config.Containers {
		containerMap[container.Name] = true
	}

	for _, name := range expectedContainers {
		if !containerMap[name] {
			t.Errorf("expected container %s not found in config", name)
		}
	}
}

func TestContainerConfig(t *testing.T) {
	tests := []struct {
		name             string
		container        ContainerConfig
		expectedName     string
		expectedPK       string
		hasTTL           bool
		hasIndexing      bool
	}{
		{
			name: "trips container",
			container: ContainerConfig{
				Name:             "trips",
				PartitionKeyPath: "/rider_id",
				IndexingPolicy:   defaultIndexingPolicy(),
			},
			expectedName: "trips",
			expectedPK:   "/rider_id",
			hasTTL:       false,
			hasIndexing:  true,
		},
		{
			name: "driver_offers with TTL",
			container: ContainerConfig{
				Name:             "driver_offers",
				PartitionKeyPath: "/driver_id",
				TTLSeconds:       300,
				IndexingPolicy:   defaultIndexingPolicy(),
			},
			expectedName: "driver_offers",
			expectedPK:   "/driver_id",
			hasTTL:       true,
			hasIndexing:  true,
		},
		{
			name: "events with custom indexing",
			container: ContainerConfig{
				Name:             "events",
				PartitionKeyPath: "/type",
				TTLSeconds:       2592000,
				IndexingPolicy:   eventIndexingPolicy(),
			},
			expectedName: "events",
			expectedPK:   "/type",
			hasTTL:       true,
			hasIndexing:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.container.Name != tt.expectedName {
				t.Errorf("expected Name=%s, got %s", tt.expectedName, tt.container.Name)
			}

			if tt.container.PartitionKeyPath != tt.expectedPK {
				t.Errorf("expected PartitionKeyPath=%s, got %s", tt.expectedPK, tt.container.PartitionKeyPath)
			}

			hasTTL := tt.container.TTLSeconds > 0
			if hasTTL != tt.hasTTL {
				t.Errorf("expected hasTTL=%v, got %v", tt.hasTTL, hasTTL)
			}

			hasIndexing := tt.container.IndexingPolicy != nil
			if hasIndexing != tt.hasIndexing {
				t.Errorf("expected hasIndexing=%v, got %v", tt.hasIndexing, hasIndexing)
			}
		})
	}
}

func TestDefaultIndexingPolicy(t *testing.T) {
	policy := defaultIndexingPolicy()

	if policy == nil {
		t.Fatal("defaultIndexingPolicy should not return nil")
	}

	if policy.IndexingMode != azcosmos.IndexingModeConsistent {
		t.Errorf("expected IndexingMode=Consistent, got %v", policy.IndexingMode)
	}

	if !policy.Automatic {
		t.Error("expected Automatic=true")
	}

	if len(policy.IncludedPaths) == 0 {
		t.Error("expected IncludedPaths to have entries")
	}

	// Check for root path
	foundRoot := false
	for _, path := range policy.IncludedPaths {
		if path.Path == "/*" {
			foundRoot = true
			break
		}
	}
	if !foundRoot {
		t.Error("expected root path /* to be included")
	}

	// Check for _etag exclusion
	foundETag := false
	for _, path := range policy.ExcludedPaths {
		if path.Path == "/_etag/?" {
			foundETag = true
			break
		}
	}
	if !foundETag {
		t.Error("expected _etag path to be excluded")
	}
}

func TestLocationIndexingPolicy(t *testing.T) {
	policy := locationIndexingPolicy()

	if policy == nil {
		t.Fatal("locationIndexingPolicy should not return nil")
	}

	if policy.IndexingMode != azcosmos.IndexingModeConsistent {
		t.Errorf("expected IndexingMode=Consistent, got %v", policy.IndexingMode)
	}

	if !policy.Automatic {
		t.Error("expected Automatic=true")
	}

	// Verify specific paths are included
	expectedPaths := []string{"/driver_id/?", "/status/?", "/city/?", "/geohash/?", "/updated_at/?", "/location/?"}
	for _, expected := range expectedPaths {
		found := false
		for _, path := range policy.IncludedPaths {
			if path.Path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected path %s to be included in location indexing policy", expected)
		}
	}

	// Verify wildcard is excluded
	foundWildcard := false
	for _, path := range policy.ExcludedPaths {
		if path.Path == "/*" {
			foundWildcard = true
			break
		}
	}
	if !foundWildcard {
		t.Error("expected wildcard /* to be excluded in location indexing policy")
	}
}

func TestEventIndexingPolicy(t *testing.T) {
	policy := eventIndexingPolicy()

	if policy == nil {
		t.Fatal("eventIndexingPolicy should not return nil")
	}

	if policy.IndexingMode != azcosmos.IndexingModeConsistent {
		t.Errorf("expected IndexingMode=Consistent, got %v", policy.IndexingMode)
	}

	// Verify specific paths are included
	expectedPaths := []string{"/type/?", "/entity_id/?", "/entity_type/?", "/timestamp/?"}
	for _, expected := range expectedPaths {
		found := false
		for _, path := range policy.IncludedPaths {
			if path.Path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected path %s to be included in event indexing policy", expected)
		}
	}

	// Verify wildcard is excluded
	foundWildcard := false
	for _, path := range policy.ExcludedPaths {
		if path.Path == "/*" {
			foundWildcard = true
			break
		}
	}
	if !foundWildcard {
		t.Error("expected wildcard /* to be excluded in event indexing policy")
	}
}

func TestIsConflictError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "409 in message",
			err:      &mockError{msg: "Status: 409 Conflict"},
			expected: true,
		},
		{
			name:     "Conflict in message",
			err:      &mockError{msg: "Resource Conflict occurred"},
			expected: true,
		},
		{
			name:     "already exists in message",
			err:      &mockError{msg: "Resource already exists"},
			expected: true,
		},
		{
			name:     "other error",
			err:      &mockError{msg: "Internal server error"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConflictError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestContainerTTLConfiguration(t *testing.T) {
	config := DefaultDatabaseConfig()

	tests := []struct {
		name        string
		container   string
		expectedTTL int32
	}{
		{name: "trips - no TTL", container: "trips", expectedTTL: 0},
		{name: "ride_requests - no TTL", container: "ride_requests", expectedTTL: 0},
		{name: "driver_offers - 5 minutes", container: "driver_offers", expectedTTL: 300},
		{name: "driver_locations - 24 hours", container: "driver_locations", expectedTTL: 86400},
		{name: "location_history - 7 days", container: "location_history", expectedTTL: 604800},
		{name: "events - 30 days", container: "events", expectedTTL: 2592000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var containerConfig *ContainerConfig
			for i := range config.Containers {
				if config.Containers[i].Name == tt.container {
					containerConfig = &config.Containers[i]
					break
				}
			}

			if containerConfig == nil {
				t.Fatalf("container %s not found in config", tt.container)
			}

			if containerConfig.TTLSeconds != tt.expectedTTL {
				t.Errorf("expected TTL=%d, got %d", tt.expectedTTL, containerConfig.TTLSeconds)
			}
		})
	}
}

func TestPartitionKeyPaths(t *testing.T) {
	config := DefaultDatabaseConfig()

	tests := []struct {
		container   string
		expectedPK  string
	}{
		{container: "trips", expectedPK: "/rider_id"},
		{container: "ride_requests", expectedPK: "/rider_id"},
		{container: "driver_offers", expectedPK: "/driver_id"},
		{container: "driver_locations", expectedPK: "/driver_id"},
		{container: "location_history", expectedPK: "/driver_id"},
		{container: "events", expectedPK: "/type"},
	}

	for _, tt := range tests {
		t.Run(tt.container, func(t *testing.T) {
			var containerConfig *ContainerConfig
			for i := range config.Containers {
				if config.Containers[i].Name == tt.container {
					containerConfig = &config.Containers[i]
					break
				}
			}

			if containerConfig == nil {
				t.Fatalf("container %s not found in config", tt.container)
			}

			if containerConfig.PartitionKeyPath != tt.expectedPK {
				t.Errorf("expected PartitionKeyPath=%s, got %s", tt.expectedPK, containerConfig.PartitionKeyPath)
			}
		})
	}
}

func TestNewInitializer(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		initializer := NewInitializer(nil, nil)

		if initializer == nil {
			t.Fatal("NewInitializer should not return nil")
		}

		if initializer.config == nil {
			t.Error("config should be set to default when nil is passed")
		}

		if initializer.config.DatabaseName != "cobrun" {
			t.Errorf("expected default database name=cobrun, got %s", initializer.config.DatabaseName)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		customConfig := &DatabaseConfig{
			DatabaseName: "custom-db",
			Containers: []ContainerConfig{
				{
					Name:             "custom-container",
					PartitionKeyPath: "/id",
				},
			},
		}

		initializer := NewInitializer(nil, customConfig)

		if initializer == nil {
			t.Fatal("NewInitializer should not return nil")
		}

		if initializer.config != customConfig {
			t.Error("config should be set to the provided custom config")
		}

		if initializer.config.DatabaseName != "custom-db" {
			t.Errorf("expected database name=custom-db, got %s", initializer.config.DatabaseName)
		}
	})
}

func TestToHelper(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{name: "int", value: 42},
		{name: "string", value: "test"},
		{name: "bool", value: true},
		{name: "float", value: 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ptr interface{}
			switch v := tt.value.(type) {
			case int:
				ptr = to(v)
				if ptr == nil {
					t.Error("to() should not return nil")
				}
				if *ptr.(*int) != v {
					t.Errorf("expected *ptr=%v, got %v", v, *ptr.(*int))
				}
			case string:
				ptr = to(v)
				if ptr == nil {
					t.Error("to() should not return nil")
				}
				if *ptr.(*string) != v {
					t.Errorf("expected *ptr=%v, got %v", v, *ptr.(*string))
				}
			case bool:
				ptr = to(v)
				if ptr == nil {
					t.Error("to() should not return nil")
				}
				if *ptr.(*bool) != v {
					t.Errorf("expected *ptr=%v, got %v", v, *ptr.(*bool))
				}
			case float64:
				ptr = to(v)
				if ptr == nil {
					t.Error("to() should not return nil")
				}
				if *ptr.(*float64) != v {
					t.Errorf("expected *ptr=%v, got %v", v, *ptr.(*float64))
				}
			}
		})
	}
}

func TestIndexingPolicyConsistency(t *testing.T) {
	// Test that all indexing policies have consistent structure
	policies := []struct {
		name   string
		policy *azcosmos.IndexingPolicy
	}{
		{name: "default", policy: defaultIndexingPolicy()},
		{name: "location", policy: locationIndexingPolicy()},
		{name: "event", policy: eventIndexingPolicy()},
	}

	for _, tt := range policies {
		t.Run(tt.name, func(t *testing.T) {
			if tt.policy == nil {
				t.Fatal("indexing policy should not be nil")
			}

			if tt.policy.IndexingMode != azcosmos.IndexingModeConsistent {
				t.Error("all policies should use consistent indexing mode")
			}

			if !tt.policy.Automatic {
				t.Error("all policies should have automatic indexing enabled")
			}

			if len(tt.policy.IncludedPaths) == 0 {
				t.Error("all policies should have at least one included path")
			}

			if len(tt.policy.ExcludedPaths) == 0 {
				t.Error("all policies should have at least one excluded path")
			}
		})
	}
}
