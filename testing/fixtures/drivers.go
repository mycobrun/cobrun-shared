// Package fixtures provides test data for unit and integration tests.
package fixtures

import (
	"time"

	"github.com/google/uuid"
)

// DriverAvailabilityFixture represents a test driver availability record.
type DriverAvailabilityFixture struct {
	ID                 string    `json:"id"`
	PartitionKey       string    `json:"pk"`
	DriverID           string    `json:"driver_id"`
	TenantID           string    `json:"tenant_id"`
	State              string    `json:"state"`
	PrevState          string    `json:"prev_state"`
	LastGeoLat         float64   `json:"last_geo_lat"`
	LastGeoLng         float64   `json:"last_geo_lng"`
	LastGeoTimestamp   time.Time `json:"last_geo_timestamp"`
	ActiveTripID       string    `json:"active_trip_id,omitempty"`
	DeviceID           string    `json:"device_id"`
	SessionID          string    `json:"session_id"`
	AppVersion         string    `json:"app_version"`
	Platform           string    `json:"platform"`
	ProductEligibility []string  `json:"product_eligibility"`
	VehicleID          string    `json:"vehicle_id"`
	VehicleType        string    `json:"vehicle_type"`
	City               string    `json:"city"`
	OnlineSince        time.Time `json:"online_since,omitempty"`
	LastStateAt        time.Time `json:"last_state_at"`
	LastHeartbeat      time.Time `json:"last_heartbeat"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// DriverProfileFixture represents a test driver profile for prerequisite checking.
type DriverProfileFixture struct {
	ID                string    `json:"id"`
	Status            string    `json:"status"`
	BackgroundStatus  string    `json:"background_status"`
	LicenseExpiry     time.Time `json:"license_expiry"`
	InsuranceExpiry   time.Time `json:"insurance_expiry"`
	VehicleRegExpiry  time.Time `json:"vehicle_reg_expiry"`
	DocumentsVerified bool      `json:"documents_verified"`
	IsSuspended       bool      `json:"is_suspended"`
	IsRestricted      bool      `json:"is_restricted"`
	HasActiveVehicle  bool      `json:"has_active_vehicle"`
}

// DriverPlatformEventFixture represents a test driver platform event.
type DriverPlatformEventFixture struct {
	ID            string                 `json:"id"`
	PartitionKey  string                 `json:"pk"`
	EventType     string                 `json:"event_type"`
	DriverID      string                 `json:"driver_id"`
	TenantID      string                 `json:"tenant_id"`
	CorrelationID string                 `json:"correlation_id"`
	FromState     string                 `json:"from_state"`
	ToState       string                 `json:"to_state"`
	GeoLat        float64                `json:"geo_lat,omitempty"`
	GeoLng        float64                `json:"geo_lng,omitempty"`
	DeviceID      string                 `json:"device_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	AppVersion    string                 `json:"app_version,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Version       int                    `json:"version"`
}

// TestDriverAvailabilities provides common driver availability fixtures for testing.
var TestDriverAvailabilities = struct {
	OfflineDriver          DriverAvailabilityFixture
	OnlineAvailableDriver  DriverAvailabilityFixture
	OnlineUnavailableDriver DriverAvailabilityFixture
	EnrouteDriver          DriverAvailabilityFixture
	OnTripDriver           DriverAvailabilityFixture
	ArrivedDriver          DriverAvailabilityFixture
}{
	OfflineDriver: DriverAvailabilityFixture{
		ID:                 "test-driver-offline",
		PartitionKey:       "cobrun:test-driver-offline",
		DriverID:           "test-driver-offline",
		TenantID:           "cobrun",
		State:              "OFFLINE",
		PrevState:          "",
		LastGeoLat:         47.6062,
		LastGeoLng:         -122.3321,
		LastGeoTimestamp:   time.Now().Add(-1 * time.Hour),
		DeviceID:           "device-001",
		SessionID:          "session-001",
		AppVersion:         "1.0.0",
		Platform:           "ios",
		ProductEligibility: []string{"STANDARD", "COMFORT"},
		VehicleID:          "vehicle-001",
		VehicleType:        "sedan",
		City:               "Seattle",
		LastStateAt:        time.Now().Add(-1 * time.Hour),
		LastHeartbeat:      time.Now().Add(-1 * time.Hour),
		CreatedAt:          time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:          time.Now().Add(-1 * time.Hour),
	},
	OnlineAvailableDriver: DriverAvailabilityFixture{
		ID:                 "test-driver-available",
		PartitionKey:       "cobrun:test-driver-available",
		DriverID:           "test-driver-available",
		TenantID:           "cobrun",
		State:              "ONLINE_AVAILABLE",
		PrevState:          "OFFLINE",
		LastGeoLat:         47.6062,
		LastGeoLng:         -122.3321,
		LastGeoTimestamp:   time.Now(),
		DeviceID:           "device-002",
		SessionID:          "session-002",
		AppVersion:         "1.0.0",
		Platform:           "android",
		ProductEligibility: []string{"STANDARD", "XL"},
		VehicleID:          "vehicle-002",
		VehicleType:        "suv",
		City:               "Seattle",
		OnlineSince:        time.Now().Add(-30 * time.Minute),
		LastStateAt:        time.Now().Add(-30 * time.Minute),
		LastHeartbeat:      time.Now(),
		CreatedAt:          time.Now().Add(-60 * 24 * time.Hour),
		UpdatedAt:          time.Now(),
	},
	OnlineUnavailableDriver: DriverAvailabilityFixture{
		ID:                 "test-driver-unavailable",
		PartitionKey:       "cobrun:test-driver-unavailable",
		DriverID:           "test-driver-unavailable",
		TenantID:           "cobrun",
		State:              "ONLINE_UNAVAILABLE",
		PrevState:          "ONLINE_AVAILABLE",
		LastGeoLat:         47.6062,
		LastGeoLng:         -122.3321,
		LastGeoTimestamp:   time.Now(),
		DeviceID:           "device-003",
		SessionID:          "session-003",
		AppVersion:         "1.0.0",
		Platform:           "ios",
		ProductEligibility: []string{"STANDARD"},
		VehicleID:          "vehicle-003",
		VehicleType:        "sedan",
		City:               "Seattle",
		OnlineSince:        time.Now().Add(-2 * time.Hour),
		LastStateAt:        time.Now().Add(-5 * time.Minute),
		LastHeartbeat:      time.Now(),
		CreatedAt:          time.Now().Add(-90 * 24 * time.Hour),
		UpdatedAt:          time.Now(),
	},
	EnrouteDriver: DriverAvailabilityFixture{
		ID:                 "test-driver-enroute",
		PartitionKey:       "cobrun:test-driver-enroute",
		DriverID:           "test-driver-enroute",
		TenantID:           "cobrun",
		State:              "ENROUTE_TO_PICKUP",
		PrevState:          "ONLINE_AVAILABLE",
		LastGeoLat:         47.6100,
		LastGeoLng:         -122.3400,
		LastGeoTimestamp:   time.Now(),
		ActiveTripID:       "trip-001",
		DeviceID:           "device-004",
		SessionID:          "session-004",
		AppVersion:         "1.0.0",
		Platform:           "android",
		ProductEligibility: []string{"STANDARD", "XL"},
		VehicleID:          "vehicle-004",
		VehicleType:        "suv",
		City:               "Seattle",
		OnlineSince:        time.Now().Add(-1 * time.Hour),
		LastStateAt:        time.Now().Add(-3 * time.Minute),
		LastHeartbeat:      time.Now(),
		CreatedAt:          time.Now().Add(-45 * 24 * time.Hour),
		UpdatedAt:          time.Now(),
	},
	ArrivedDriver: DriverAvailabilityFixture{
		ID:                 "test-driver-arrived",
		PartitionKey:       "cobrun:test-driver-arrived",
		DriverID:           "test-driver-arrived",
		TenantID:           "cobrun",
		State:              "DRIVER_ARRIVED",
		PrevState:          "ENROUTE_TO_PICKUP",
		LastGeoLat:         47.6062,
		LastGeoLng:         -122.3321,
		LastGeoTimestamp:   time.Now(),
		ActiveTripID:       "trip-002",
		DeviceID:           "device-005",
		SessionID:          "session-005",
		AppVersion:         "1.0.0",
		Platform:           "ios",
		ProductEligibility: []string{"STANDARD"},
		VehicleID:          "vehicle-005",
		VehicleType:        "sedan",
		City:               "Seattle",
		OnlineSince:        time.Now().Add(-90 * time.Minute),
		LastStateAt:        time.Now().Add(-1 * time.Minute),
		LastHeartbeat:      time.Now(),
		CreatedAt:          time.Now().Add(-120 * 24 * time.Hour),
		UpdatedAt:          time.Now(),
	},
	OnTripDriver: DriverAvailabilityFixture{
		ID:                 "test-driver-ontrip",
		PartitionKey:       "cobrun:test-driver-ontrip",
		DriverID:           "test-driver-ontrip",
		TenantID:           "cobrun",
		State:              "ON_TRIP",
		PrevState:          "DRIVER_ARRIVED",
		LastGeoLat:         47.6200,
		LastGeoLng:         -122.3500,
		LastGeoTimestamp:   time.Now(),
		ActiveTripID:       "trip-003",
		DeviceID:           "device-006",
		SessionID:          "session-006",
		AppVersion:         "1.0.0",
		Platform:           "android",
		ProductEligibility: []string{"STANDARD", "COMFORT", "XL"},
		VehicleID:          "vehicle-006",
		VehicleType:        "suv",
		City:               "Seattle",
		OnlineSince:        time.Now().Add(-2 * time.Hour),
		LastStateAt:        time.Now().Add(-10 * time.Minute),
		LastHeartbeat:      time.Now(),
		CreatedAt:          time.Now().Add(-180 * 24 * time.Hour),
		UpdatedAt:          time.Now(),
	},
}

// TestDriverProfiles provides common driver profile fixtures for testing prerequisites.
var TestDriverProfiles = struct {
	ValidDriver       DriverProfileFixture
	ExpiredLicense    DriverProfileFixture
	SuspendedDriver   DriverProfileFixture
	PendingBackground DriverProfileFixture
	NoVehicle         DriverProfileFixture
	RestrictedDriver  DriverProfileFixture
}{
	ValidDriver: DriverProfileFixture{
		ID:                "valid-driver-001",
		Status:            "active",
		BackgroundStatus:  "approved",
		LicenseExpiry:     time.Now().Add(365 * 24 * time.Hour),
		InsuranceExpiry:   time.Now().Add(180 * 24 * time.Hour),
		VehicleRegExpiry:  time.Now().Add(90 * 24 * time.Hour),
		DocumentsVerified: true,
		IsSuspended:       false,
		IsRestricted:      false,
		HasActiveVehicle:  true,
	},
	ExpiredLicense: DriverProfileFixture{
		ID:                "expired-license-001",
		Status:            "active",
		BackgroundStatus:  "approved",
		LicenseExpiry:     time.Now().Add(-30 * 24 * time.Hour), // Expired 30 days ago
		InsuranceExpiry:   time.Now().Add(180 * 24 * time.Hour),
		VehicleRegExpiry:  time.Now().Add(90 * 24 * time.Hour),
		DocumentsVerified: true,
		IsSuspended:       false,
		IsRestricted:      false,
		HasActiveVehicle:  true,
	},
	SuspendedDriver: DriverProfileFixture{
		ID:                "suspended-driver-001",
		Status:            "active",
		BackgroundStatus:  "approved",
		LicenseExpiry:     time.Now().Add(365 * 24 * time.Hour),
		InsuranceExpiry:   time.Now().Add(180 * 24 * time.Hour),
		VehicleRegExpiry:  time.Now().Add(90 * 24 * time.Hour),
		DocumentsVerified: true,
		IsSuspended:       true, // SUSPENDED
		IsRestricted:      false,
		HasActiveVehicle:  true,
	},
	PendingBackground: DriverProfileFixture{
		ID:                "pending-bg-001",
		Status:            "active",
		BackgroundStatus:  "pending", // Background check not completed
		LicenseExpiry:     time.Now().Add(365 * 24 * time.Hour),
		InsuranceExpiry:   time.Now().Add(180 * 24 * time.Hour),
		VehicleRegExpiry:  time.Now().Add(90 * 24 * time.Hour),
		DocumentsVerified: true,
		IsSuspended:       false,
		IsRestricted:      false,
		HasActiveVehicle:  true,
	},
	NoVehicle: DriverProfileFixture{
		ID:                "no-vehicle-001",
		Status:            "active",
		BackgroundStatus:  "approved",
		LicenseExpiry:     time.Now().Add(365 * 24 * time.Hour),
		InsuranceExpiry:   time.Now().Add(180 * 24 * time.Hour),
		VehicleRegExpiry:  time.Now().Add(90 * 24 * time.Hour),
		DocumentsVerified: true,
		IsSuspended:       false,
		IsRestricted:      false,
		HasActiveVehicle:  false, // No vehicle assigned
	},
	RestrictedDriver: DriverProfileFixture{
		ID:                "restricted-driver-001",
		Status:            "active",
		BackgroundStatus:  "approved",
		LicenseExpiry:     time.Now().Add(365 * 24 * time.Hour),
		InsuranceExpiry:   time.Now().Add(180 * 24 * time.Hour),
		VehicleRegExpiry:  time.Now().Add(90 * 24 * time.Hour),
		DocumentsVerified: true,
		IsSuspended:       false,
		IsRestricted:      true, // Account restricted
		HasActiveVehicle:  true,
	},
}

// NewRandomDriverAvailability creates a new random driver availability fixture.
func NewRandomDriverAvailability(state string) DriverAvailabilityFixture {
	id := uuid.New().String()
	now := time.Now()
	return DriverAvailabilityFixture{
		ID:                 id,
		PartitionKey:       "cobrun:" + id,
		DriverID:           id,
		TenantID:           "cobrun",
		State:              state,
		PrevState:          "OFFLINE",
		LastGeoLat:         47.6062 + (float64(now.UnixNano()%100) / 10000),
		LastGeoLng:         -122.3321 + (float64(now.UnixNano()%100) / 10000),
		LastGeoTimestamp:   now,
		DeviceID:           "device-" + id[:8],
		SessionID:          "session-" + id[:8],
		AppVersion:         "1.0.0",
		Platform:           "ios",
		ProductEligibility: []string{"STANDARD"},
		VehicleID:          "vehicle-" + id[:8],
		VehicleType:        "sedan",
		City:               "Seattle",
		LastStateAt:        now,
		LastHeartbeat:      now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// NewRandomDriverProfile creates a new random valid driver profile fixture.
func NewRandomDriverProfile() DriverProfileFixture {
	id := uuid.New().String()
	return DriverProfileFixture{
		ID:                id,
		Status:            "active",
		BackgroundStatus:  "approved",
		LicenseExpiry:     time.Now().Add(365 * 24 * time.Hour),
		InsuranceExpiry:   time.Now().Add(180 * 24 * time.Hour),
		VehicleRegExpiry:  time.Now().Add(90 * 24 * time.Hour),
		DocumentsVerified: true,
		IsSuspended:       false,
		IsRestricted:      false,
		HasActiveVehicle:  true,
	}
}

// LocationBatchFixture represents a batch of location points for testing.
type LocationBatchFixture struct {
	DriverID       string
	IdempotencyKey string
	Points         []LocationPointFixture
}

// LocationPointFixture represents a single location point.
type LocationPointFixture struct {
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Timestamp time.Time `json:"timestamp"`
	Accuracy  float64   `json:"accuracy"`
	Speed     float64   `json:"speed"`
	Heading   float64   `json:"heading"`
}

// NewLocationBatch creates a location batch fixture for testing.
func NewLocationBatch(driverID string, count int) LocationBatchFixture {
	now := time.Now()
	points := make([]LocationPointFixture, count)

	baseLat := 47.6062
	baseLng := -122.3321

	for i := 0; i < count; i++ {
		points[i] = LocationPointFixture{
			Lat:       baseLat + float64(i)*0.0001, // Small increments
			Lng:       baseLng + float64(i)*0.0001,
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Accuracy:  10.0,
			Speed:     25.0 + float64(i),
			Heading:   90.0,
		}
	}

	return LocationBatchFixture{
		DriverID:       driverID,
		IdempotencyKey: uuid.New().String(),
		Points:         points,
	}
}
