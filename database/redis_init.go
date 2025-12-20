// Package database provides Redis initialization and data structure setup utilities.
package database

import (
	"context"
	"fmt"
	"log"
	"time"
)

// RedisKeyPatterns defines the key patterns used in the application.
var RedisKeyPatterns = struct {
	// Geospatial keys for driver locations
	DriverLocations     string // driver_locations:{city}
	DriverLocationsByID string // driver_location:{driver_id}

	// Driver status
	DriverStatus       string // driver:{driver_id}:status
	ActiveDrivers      string // active_drivers:{city}
	OnlineDrivers      string // online_drivers:{city}

	// Surge pricing
	SurgeZone          string // surge:{zone_id}
	SurgeByCity        string // surge:city:{city}

	// Ride requests
	ActiveRequest      string // request:{request_id}
	RiderActiveRequest string // rider:{rider_id}:active_request

	// Driver offers
	DriverOffer        string // offer:{offer_id}
	DriverPendingOffer string // driver:{driver_id}:pending_offer

	// Rate limiting
	RateLimit          string // ratelimit:{entity_type}:{entity_id}:{action}

	// Sessions
	Session            string // session:{session_id}
	UserSessions       string // user:{user_id}:sessions

	// Caching
	GeofenceCache      string // geofence:{geofence_id}
	PriceEstimateCache string // price_estimate:{hash}
	RateCardCache      string // rate_card:{service_area_id}:{ride_type}
	UserCache          string // user:{user_id}

	// Real-time tracking
	TripTracking       string // trip:{trip_id}:tracking
	TripSubscribers    string // trip:{trip_id}:subscribers
}{
	DriverLocations:     "driver_locations:%s",
	DriverLocationsByID: "driver_location:%s",
	DriverStatus:        "driver:%s:status",
	ActiveDrivers:       "active_drivers:%s",
	OnlineDrivers:       "online_drivers:%s",
	SurgeZone:           "surge:%s",
	SurgeByCity:         "surge:city:%s",
	ActiveRequest:       "request:%s",
	RiderActiveRequest:  "rider:%s:active_request",
	DriverOffer:         "offer:%s",
	DriverPendingOffer:  "driver:%s:pending_offer",
	RateLimit:           "ratelimit:%s:%s:%s",
	Session:             "session:%s",
	UserSessions:        "user:%s:sessions",
	GeofenceCache:       "geofence:%s",
	PriceEstimateCache:  "price_estimate:%s",
	RateCardCache:       "rate_card:%s:%s",
	UserCache:           "user:%s",
	TripTracking:        "trip:%s:tracking",
	TripSubscribers:     "trip:%s:subscribers",
}

// RedisTTLs defines the TTL values for different key types.
var RedisTTLs = struct {
	DriverLocation     time.Duration
	DriverStatus       time.Duration
	ActiveDriver       time.Duration
	Surge              time.Duration
	ActiveRequest      time.Duration
	DriverOffer        time.Duration
	Session            time.Duration
	GeofenceCache      time.Duration
	PriceEstimateCache time.Duration
	RateCardCache      time.Duration
	UserCache          time.Duration
	TripTracking       time.Duration
}{
	DriverLocation:     30 * time.Second,  // Stale after 30 seconds
	DriverStatus:       5 * time.Minute,   // Refresh every 5 minutes
	ActiveDriver:       1 * time.Minute,   // Quick expiry for active lists
	Surge:              5 * time.Minute,   // Surge data refresh
	ActiveRequest:      10 * time.Minute,  // Request timeout
	DriverOffer:        15 * time.Second,  // Offer expiry
	Session:            24 * time.Hour,    // Session duration
	GeofenceCache:      1 * time.Hour,     // Geofence cache
	PriceEstimateCache: 5 * time.Minute,   // Price estimate validity
	RateCardCache:      1 * time.Hour,     // Rate card cache
	UserCache:          15 * time.Minute,  // User cache
	TripTracking:       6 * time.Hour,     // Trip tracking data
}

// RedisInitializer handles Redis initialization.
type RedisInitializer struct {
	client *RedisClient
}

// NewRedisInitializer creates a new Redis initializer.
func NewRedisInitializer(client *RedisClient) *RedisInitializer {
	return &RedisInitializer{client: client}
}

// Initialize sets up Redis for the application.
func (ri *RedisInitializer) Initialize(ctx context.Context) error {
	// Verify connection
	if err := ri.client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connection verified successfully")

	// Initialize default service areas for location tracking
	defaultCities := []string{
		"sanfrancisco",
		"losangeles",
		"newyork",
		"chicago",
		"seattle",
		"austin",
		"denver",
		"miami",
	}

	// Ensure geo keys exist for each city
	for _, city := range defaultCities {
		key := fmt.Sprintf(RedisKeyPatterns.DriverLocations, city)
		// Just ensure the key exists by checking it
		_, _ = ri.client.client.Exists(ctx, key).Result()
	}

	log.Printf("Redis initialization completed for %d cities", len(defaultCities))
	return nil
}

// DriverStatusData represents driver status stored in Redis.
type DriverStatusData struct {
	Status       string    `json:"status"`
	VehicleID    string    `json:"vehicle_id,omitempty"`
	TripID       string    `json:"trip_id,omitempty"`
	LastLocation string    `json:"last_location,omitempty"` // "lat,lng"
	UpdatedAt    time.Time `json:"updated_at"`
}

// SurgeZoneData represents surge zone data stored in Redis.
type SurgeZoneData struct {
	ZoneID           string    `json:"zone_id"`
	Multiplier       float64   `json:"multiplier"`
	Demand           int       `json:"demand"`
	Supply           int       `json:"supply"`
	DemandSupplyRatio float64  `json:"demand_supply_ratio"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ActiveRequestData represents active request data stored in Redis.
type ActiveRequestData struct {
	RequestID   string    `json:"request_id"`
	RiderID     string    `json:"rider_id"`
	Status      string    `json:"status"`
	PickupLat   float64   `json:"pickup_lat"`
	PickupLng   float64   `json:"pickup_lng"`
	RideType    string    `json:"ride_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// OfferData represents offer data stored in Redis.
type OfferData struct {
	OfferID   string    `json:"offer_id"`
	RequestID string    `json:"request_id"`
	DriverID  string    `json:"driver_id"`
	Fare      float64   `json:"fare"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionData represents session data stored in Redis.
type SessionData struct {
	SessionID  string    `json:"session_id"`
	UserID     string    `json:"user_id"`
	UserType   string    `json:"user_type"`
	DeviceID   string    `json:"device_id,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// DriverLocationService provides Redis-based driver location operations.
type DriverLocationService struct {
	client *RedisClient
}

// NewDriverLocationService creates a new driver location service.
func NewDriverLocationService(client *RedisClient) *DriverLocationService {
	return &DriverLocationService{client: client}
}

// UpdateLocation updates a driver's location.
func (s *DriverLocationService) UpdateLocation(ctx context.Context, driverID, city string, lat, lng float64) error {
	key := fmt.Sprintf(RedisKeyPatterns.DriverLocations, city)
	return s.client.GeoAdd(ctx, key, lng, lat, driverID)
}

// GetNearbyDrivers finds drivers near a location.
func (s *DriverLocationService) GetNearbyDrivers(ctx context.Context, city string, lat, lng, radiusKm float64) ([]GeoResult, error) {
	key := fmt.Sprintf(RedisKeyPatterns.DriverLocations, city)
	return s.client.GeoRadius(ctx, key, lng, lat, radiusKm, "km", true, true, true, 50)
}

// RemoveDriver removes a driver from location tracking.
func (s *DriverLocationService) RemoveDriver(ctx context.Context, driverID, city string) error {
	key := fmt.Sprintf(RedisKeyPatterns.DriverLocations, city)
	return s.client.ZRem(ctx, key, driverID)
}

// SetDriverOnline marks a driver as online.
func (s *DriverLocationService) SetDriverOnline(ctx context.Context, driverID, city string) error {
	key := fmt.Sprintf(RedisKeyPatterns.OnlineDrivers, city)
	return s.client.SAdd(ctx, key, driverID)
}

// SetDriverOffline marks a driver as offline.
func (s *DriverLocationService) SetDriverOffline(ctx context.Context, driverID, city string) error {
	// Remove from online drivers set
	key := fmt.Sprintf(RedisKeyPatterns.OnlineDrivers, city)
	_ = s.client.SRem(ctx, key, driverID)

	// Remove from location geo set
	return s.RemoveDriver(ctx, driverID, city)
}

// GetOnlineDriverCount returns the number of online drivers in a city.
func (s *DriverLocationService) GetOnlineDriverCount(ctx context.Context, city string) (int64, error) {
	key := fmt.Sprintf(RedisKeyPatterns.OnlineDrivers, city)
	return s.client.SCard(ctx, key)
}

