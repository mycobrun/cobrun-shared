// Package cosmosdb provides Cosmos DB document schemas.
package cosmosdb

import (
	"time"
)

// ============================================================================
// TRIPS CONTAINER
// ============================================================================

// Trip represents a trip document in Cosmos DB.
type Trip struct {
	ID        string `json:"id"`
	RequestID string `json:"request_id"`
	RiderID   string `json:"rider_id"` // Partition key
	DriverID  string `json:"driver_id"`
	VehicleID string `json:"vehicle_id"`
	RideType  string `json:"ride_type"`
	Status    string `json:"status"`

	// Locations
	PickupLocation  GeoPoint  `json:"pickup_location"`
	PickupAddress   string    `json:"pickup_address,omitempty"`
	DropoffLocation GeoPoint  `json:"dropoff_location"`
	DropoffAddress  string    `json:"dropoff_address,omitempty"`
	CurrentLocation *GeoPoint `json:"current_location,omitempty"`

	// Route
	RoutePolyline  string     `json:"route_polyline,omitempty"`
	EstimatedRoute []GeoPoint `json:"estimated_route,omitempty"`
	ActualRoute    []GeoPoint `json:"actual_route,omitempty"`

	// Distance & Duration
	EstimatedDistanceMeters float64 `json:"estimated_distance_meters"`
	EstimatedDurationSecs   int     `json:"estimated_duration_seconds"`
	ActualDistanceMeters    float64 `json:"actual_distance_meters,omitempty"`
	ActualDurationSecs      int     `json:"actual_duration_seconds,omitempty"`

	// ETAs
	PickupETASecs  int `json:"pickup_eta_seconds,omitempty"`
	DropoffETASecs int `json:"dropoff_eta_seconds,omitempty"`

	// Pricing
	EstimatedFare   float64        `json:"estimated_fare"`
	ActualFare      float64        `json:"actual_fare,omitempty"`
	SurgeMultiplier float64        `json:"surge_multiplier"`
	Currency        string         `json:"currency"`
	FareBreakdown   *FareBreakdown `json:"fare_breakdown,omitempty"`

	// Payment
	PaymentMethodID string `json:"payment_method_id"`
	PaymentStatus   string `json:"payment_status,omitempty"`
	PaymentID       string `json:"payment_id,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	EnRouteAt   *time.Time `json:"en_route_at,omitempty"`
	ArrivedAt   *time.Time `json:"arrived_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`

	// Cancellation
	CancelledBy        string  `json:"cancelled_by,omitempty"`
	CancellationReason string  `json:"cancellation_reason,omitempty"`
	CancellationFee    float64 `json:"cancellation_fee,omitempty"`

	// Ratings
	RiderRating   *float64 `json:"rider_rating,omitempty"`
	DriverRating  *float64 `json:"driver_rating,omitempty"`
	RiderComment  string   `json:"rider_comment,omitempty"`
	DriverComment string   `json:"driver_comment,omitempty"`

	// Metadata
	PromoCode     string  `json:"promo_code,omitempty"`
	PromoDiscount float64 `json:"promo_discount,omitempty"`
	Notes         string  `json:"notes,omitempty"`

	// TTL (optional, -1 means no expiry)
	TTL int32 `json:"ttl,omitempty"`
}

// FareBreakdown represents the breakdown of a trip fare.
type FareBreakdown struct {
	BaseFare      float64 `json:"base_fare"`
	DistanceFare  float64 `json:"distance_fare"`
	TimeFare      float64 `json:"time_fare"`
	SurgeFare     float64 `json:"surge_fare,omitempty"`
	TollFees      float64 `json:"toll_fees,omitempty"`
	ServiceFee    float64 `json:"service_fee"`
	PromoDiscount float64 `json:"promo_discount,omitempty"`
	Tips          float64 `json:"tips,omitempty"`
	Total         float64 `json:"total"`
}

// ============================================================================
// RIDE REQUESTS CONTAINER
// ============================================================================

// RideRequest represents a ride request document.
type RideRequest struct {
	ID              string   `json:"id"`
	RiderID         string   `json:"rider_id"` // Partition key
	PickupLocation  GeoPoint `json:"pickup_location"`
	PickupAddress   string   `json:"pickup_address,omitempty"`
	DropoffLocation GeoPoint `json:"dropoff_location"`
	DropoffAddress  string   `json:"dropoff_address,omitempty"`
	RideType        string   `json:"ride_type"`
	Status          string   `json:"status"`

	// Scheduling
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	IsScheduled bool       `json:"is_scheduled"`

	// Matching
	MatchedDriverID string     `json:"matched_driver_id,omitempty"`
	MatchedAt       *time.Time `json:"matched_at,omitempty"`
	SearchAttempts  int        `json:"search_attempts"`
	CurrentRadius   float64    `json:"current_radius"`

	// Pricing
	EstimatedFare   float64 `json:"estimated_fare"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	Currency        string  `json:"currency"`

	// ETA
	PickupETASecs    int     `json:"pickup_eta_seconds"`
	TripDurationSecs int     `json:"trip_duration_seconds"`
	TripDistanceM    float64 `json:"trip_distance_meters"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ExpiresAt time.Time `json:"expires_at"`

	// Metadata
	PromoCode       string `json:"promo_code,omitempty"`
	PaymentMethodID string `json:"payment_method_id,omitempty"`
	Notes           string `json:"notes,omitempty"`

	// TTL
	TTL int32 `json:"ttl,omitempty"`
}

// ============================================================================
// DRIVER OFFERS CONTAINER
// ============================================================================

// DriverOffer represents an offer sent to a driver.
type DriverOffer struct {
	ID            string     `json:"id"`
	RequestID     string     `json:"request_id"`
	DriverID      string     `json:"driver_id"` // Partition key
	Status        string     `json:"status"`
	PickupETASecs int        `json:"pickup_eta_seconds"`
	DistanceM     float64    `json:"distance_meters"`
	Fare          float64    `json:"fare"`
	ExpiresAt     time.Time  `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	RespondedAt   *time.Time `json:"responded_at,omitempty"`

	// TTL - offers expire automatically
	TTL int32 `json:"ttl"`
}

// ============================================================================
// DRIVER LOCATIONS CONTAINER
// ============================================================================

// DriverLocation represents a driver's current location.
type DriverLocation struct {
	ID         string    `json:"id"`
	DriverID   string    `json:"driver_id"` // Partition key
	Location   GeoPoint  `json:"location"`
	Heading    float64   `json:"heading"`
	Speed      float64   `json:"speed"`
	Accuracy   float64   `json:"accuracy"`
	Status     string    `json:"status"`
	City       string    `json:"city,omitempty"`
	Geohash    string    `json:"geohash,omitempty"`
	TripID     string    `json:"trip_id,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
	ReceivedAt time.Time `json:"received_at"`

	// TTL - stale locations expire
	TTL int32 `json:"ttl,omitempty"`
}

// ============================================================================
// LOCATION HISTORY CONTAINER
// ============================================================================

// LocationHistory represents a historical location entry.
type LocationHistory struct {
	ID        string    `json:"id"`
	DriverID  string    `json:"driver_id"` // Partition key
	Location  GeoPoint  `json:"location"`
	Heading   float64   `json:"heading"`
	Speed     float64   `json:"speed"`
	TripID    string    `json:"trip_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`

	// TTL - history expires after retention period
	TTL int32 `json:"ttl,omitempty"`
}

// ============================================================================
// EVENTS CONTAINER
// ============================================================================

// Event represents a domain event.
type Event struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // Partition key
	EntityID   string                 `json:"entity_id"`
	EntityType string                 `json:"entity_type"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Metadata   *EventMetadata         `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`

	// TTL - events expire after retention period
	TTL int32 `json:"ttl,omitempty"`
}

// EventMetadata contains event metadata.
type EventMetadata struct {
	Source      string `json:"source"`
	Version     string `json:"version"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// ============================================================================
// COMMON TYPES
// ============================================================================

// GeoPoint represents a geographic point.
type GeoPoint struct {
	Type        string    `json:"type"` // Always "Point" for GeoJSON
	Coordinates []float64 `json:"coordinates"` // [longitude, latitude]
}

// NewGeoPoint creates a new GeoPoint from latitude and longitude.
func NewGeoPoint(lat, lng float64) GeoPoint {
	return GeoPoint{
		Type:        "Point",
		Coordinates: []float64{lng, lat}, // GeoJSON format: [lng, lat]
	}
}

// Lat returns the latitude.
func (g GeoPoint) Lat() float64 {
	if len(g.Coordinates) >= 2 {
		return g.Coordinates[1]
	}
	return 0
}

// Lng returns the longitude.
func (g GeoPoint) Lng() float64 {
	if len(g.Coordinates) >= 1 {
		return g.Coordinates[0]
	}
	return 0
}

