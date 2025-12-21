// Package fixtures provides test data for unit and integration tests.
package fixtures

import (
	"time"

	"github.com/google/uuid"
)

// LocationFixture represents a geographic location.
type LocationFixture struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

// TripFixture represents a test trip.
type TripFixture struct {
	ID            string          `json:"id"`
	RiderID       string          `json:"rider_id"`
	DriverID      string          `json:"driver_id"`
	Status        string          `json:"status"`
	Pickup        LocationFixture `json:"pickup"`
	Dropoff       LocationFixture `json:"dropoff"`
	VehicleType   string          `json:"vehicle_type"`
	EstimatedFare float64         `json:"estimated_fare"`
	ActualFare    float64         `json:"actual_fare"`
	Distance      float64         `json:"distance_km"`
	Duration      int             `json:"duration_minutes"`
	CreatedAt     time.Time       `json:"created_at"`
}

// Common SF locations for testing
var SFLocations = struct {
	UnionSquare     LocationFixture
	FishermansWharf LocationFixture
	SFO             LocationFixture
	GoldenGatePark  LocationFixture
	MissionDistrict LocationFixture
	SOMA            LocationFixture
}{
	UnionSquare: LocationFixture{
		Latitude:  37.7879,
		Longitude: -122.4074,
		Address:   "Union Square, San Francisco, CA 94108",
	},
	FishermansWharf: LocationFixture{
		Latitude:  37.8080,
		Longitude: -122.4177,
		Address:   "Fisherman's Wharf, San Francisco, CA 94133",
	},
	SFO: LocationFixture{
		Latitude:  37.6213,
		Longitude: -122.3790,
		Address:   "San Francisco International Airport, CA 94128",
	},
	GoldenGatePark: LocationFixture{
		Latitude:  37.7694,
		Longitude: -122.4862,
		Address:   "Golden Gate Park, San Francisco, CA 94122",
	},
	MissionDistrict: LocationFixture{
		Latitude:  37.7599,
		Longitude: -122.4148,
		Address:   "Mission District, San Francisco, CA 94110",
	},
	SOMA: LocationFixture{
		Latitude:  37.7785,
		Longitude: -122.3950,
		Address:   "SOMA, San Francisco, CA 94103",
	},
}

// TestTrips provides common trip fixtures for testing.
var TestTrips = struct {
	CompletedTrip    TripFixture
	InProgressTrip   TripFixture
	RequestedTrip    TripFixture
	CancelledTrip    TripFixture
	AirportTrip      TripFixture
}{
	CompletedTrip: TripFixture{
		ID:            "trip-completed-001",
		RiderID:       TestUsers.Rider.ID,
		DriverID:      TestUsers.Driver.ID,
		Status:        "completed",
		Pickup:        SFLocations.UnionSquare,
		Dropoff:       SFLocations.FishermansWharf,
		VehicleType:   "uberx",
		EstimatedFare: 15.50,
		ActualFare:    14.75,
		Distance:      3.2,
		Duration:      12,
		CreatedAt:     time.Now().Add(-2 * time.Hour),
	},
	InProgressTrip: TripFixture{
		ID:            "trip-inprogress-001",
		RiderID:       TestUsers.Rider.ID,
		DriverID:      TestUsers.Driver.ID,
		Status:        "in_progress",
		Pickup:        SFLocations.MissionDistrict,
		Dropoff:       SFLocations.GoldenGatePark,
		VehicleType:   "uberx",
		EstimatedFare: 22.00,
		ActualFare:    0,
		Distance:      5.8,
		Duration:      18,
		CreatedAt:     time.Now().Add(-15 * time.Minute),
	},
	RequestedTrip: TripFixture{
		ID:            "trip-requested-001",
		RiderID:       TestUsers.Rider.ID,
		DriverID:      "",
		Status:        "requested",
		Pickup:        SFLocations.SOMA,
		Dropoff:       SFLocations.UnionSquare,
		VehicleType:   "uberx",
		EstimatedFare: 12.00,
		ActualFare:    0,
		Distance:      2.1,
		Duration:      8,
		CreatedAt:     time.Now(),
	},
	CancelledTrip: TripFixture{
		ID:            "trip-cancelled-001",
		RiderID:       TestUsers.Rider.ID,
		DriverID:      TestUsers.Driver.ID,
		Status:        "cancelled",
		Pickup:        SFLocations.UnionSquare,
		Dropoff:       SFLocations.SFO,
		VehicleType:   "uberxl",
		EstimatedFare: 55.00,
		ActualFare:    5.00, // Cancellation fee
		Distance:      21.5,
		Duration:      35,
		CreatedAt:     time.Now().Add(-1 * time.Hour),
	},
	AirportTrip: TripFixture{
		ID:            "trip-airport-001",
		RiderID:       TestUsers.Rider.ID,
		DriverID:      TestUsers.Driver.ID,
		Status:        "completed",
		Pickup:        SFLocations.SFO,
		Dropoff:       SFLocations.UnionSquare,
		VehicleType:   "uberblack",
		EstimatedFare: 75.00,
		ActualFare:    78.50,
		Distance:      21.5,
		Duration:      42,
		CreatedAt:     time.Now().Add(-24 * time.Hour),
	},
}

// NewRandomTrip creates a new random trip fixture.
func NewRandomTrip(riderID, driverID string) TripFixture {
	id := uuid.New().String()
	return TripFixture{
		ID:            id,
		RiderID:       riderID,
		DriverID:      driverID,
		Status:        "requested",
		Pickup:        SFLocations.UnionSquare,
		Dropoff:       SFLocations.MissionDistrict,
		VehicleType:   "uberx",
		EstimatedFare: 18.50,
		ActualFare:    0,
		Distance:      4.2,
		Duration:      15,
		CreatedAt:     time.Now(),
	}
}
