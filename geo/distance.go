// Package geo provides geospatial utilities.
package geo

import (
	"math"
)

const (
	// EarthRadiusKm is the Earth's radius in kilometers.
	EarthRadiusKm = 6371.0
	// EarthRadiusMiles is the Earth's radius in miles.
	EarthRadiusMiles = 3958.8
	// MetersPerKm converts kilometers to meters.
	MetersPerKm = 1000.0
	// FeetPerMile converts miles to feet.
	FeetPerMile = 5280.0
)

// Point represents a geographic coordinate.
type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// NewPoint creates a new Point.
func NewPoint(lat, lng float64) Point {
	return Point{Lat: lat, Lng: lng}
}

// IsValid checks if the point has valid coordinates.
func (p Point) IsValid() bool {
	return p.Lat >= -90 && p.Lat <= 90 && p.Lng >= -180 && p.Lng <= 180
}

// HaversineDistance calculates the great-circle distance between two points
// using the Haversine formula. Returns distance in kilometers.
func HaversineDistance(p1, p2 Point) float64 {
	lat1 := degreesToRadians(p1.Lat)
	lat2 := degreesToRadians(p2.Lat)
	deltaLat := degreesToRadians(p2.Lat - p1.Lat)
	deltaLng := degreesToRadians(p2.Lng - p1.Lng)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusKm * c
}

// HaversineDistanceMeters returns distance in meters.
func HaversineDistanceMeters(p1, p2 Point) float64 {
	return HaversineDistance(p1, p2) * MetersPerKm
}

// HaversineDistanceMiles returns distance in miles.
func HaversineDistanceMiles(p1, p2 Point) float64 {
	return HaversineDistance(p1, p2) * 0.621371
}

// Bearing calculates the initial bearing from p1 to p2.
// Returns bearing in degrees (0-360, where 0 is North).
func Bearing(p1, p2 Point) float64 {
	lat1 := degreesToRadians(p1.Lat)
	lat2 := degreesToRadians(p2.Lat)
	deltaLng := degreesToRadians(p2.Lng - p1.Lng)

	x := math.Sin(deltaLng) * math.Cos(lat2)
	y := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(deltaLng)

	bearing := math.Atan2(x, y)
	bearingDegrees := radiansToDegrees(bearing)

	// Normalize to 0-360
	return math.Mod(bearingDegrees+360, 360)
}

// DestinationPoint calculates the destination point given start point,
// bearing (in degrees), and distance (in kilometers).
func DestinationPoint(start Point, bearing, distanceKm float64) Point {
	lat1 := degreesToRadians(start.Lat)
	lng1 := degreesToRadians(start.Lng)
	bearingRad := degreesToRadians(bearing)

	angularDist := distanceKm / EarthRadiusKm

	lat2 := math.Asin(
		math.Sin(lat1)*math.Cos(angularDist) +
			math.Cos(lat1)*math.Sin(angularDist)*math.Cos(bearingRad),
	)

	lng2 := lng1 + math.Atan2(
		math.Sin(bearingRad)*math.Sin(angularDist)*math.Cos(lat1),
		math.Cos(angularDist)-math.Sin(lat1)*math.Sin(lat2),
	)

	// Normalize longitude to -180 to 180
	lng2 = math.Mod(lng2+3*math.Pi, 2*math.Pi) - math.Pi

	return Point{
		Lat: radiansToDegrees(lat2),
		Lng: radiansToDegrees(lng2),
	}
}

// Midpoint calculates the midpoint between two points.
func Midpoint(p1, p2 Point) Point {
	lat1 := degreesToRadians(p1.Lat)
	lat2 := degreesToRadians(p2.Lat)
	lng1 := degreesToRadians(p1.Lng)
	deltaLng := degreesToRadians(p2.Lng - p1.Lng)

	bx := math.Cos(lat2) * math.Cos(deltaLng)
	by := math.Cos(lat2) * math.Sin(deltaLng)

	lat3 := math.Atan2(
		math.Sin(lat1)+math.Sin(lat2),
		math.Sqrt((math.Cos(lat1)+bx)*(math.Cos(lat1)+bx)+by*by),
	)
	lng3 := lng1 + math.Atan2(by, math.Cos(lat1)+bx)

	return Point{
		Lat: radiansToDegrees(lat3),
		Lng: radiansToDegrees(lng3),
	}
}

// BoundingBox calculates a bounding box around a center point
// with a given radius in kilometers.
type BoundingBox struct {
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLng float64 `json:"min_lng"`
	MaxLng float64 `json:"max_lng"`
}

// BoundingBoxFromPoint creates a bounding box around a point.
func BoundingBoxFromPoint(center Point, radiusKm float64) BoundingBox {
	// Approximate degrees per km at different latitudes
	latDelta := radiusKm / 111.0 // ~111 km per degree of latitude
	lngDelta := radiusKm / (111.0 * math.Cos(degreesToRadians(center.Lat)))

	return BoundingBox{
		MinLat: center.Lat - latDelta,
		MaxLat: center.Lat + latDelta,
		MinLng: center.Lng - lngDelta,
		MaxLng: center.Lng + lngDelta,
	}
}

// Contains checks if a point is within the bounding box.
func (bb BoundingBox) Contains(p Point) bool {
	return p.Lat >= bb.MinLat && p.Lat <= bb.MaxLat &&
		p.Lng >= bb.MinLng && p.Lng <= bb.MaxLng
}

// Center returns the center point of the bounding box.
func (bb BoundingBox) Center() Point {
	return Point{
		Lat: (bb.MinLat + bb.MaxLat) / 2,
		Lng: (bb.MinLng + bb.MaxLng) / 2,
	}
}

// FindNearestPoints returns points within a given radius of a center point.
func FindNearestPoints(center Point, points []Point, radiusKm float64) []Point {
	var nearest []Point

	// First, use bounding box for quick filtering
	bbox := BoundingBoxFromPoint(center, radiusKm)

	for _, p := range points {
		if bbox.Contains(p) {
			// Then verify with actual distance
			if HaversineDistance(center, p) <= radiusKm {
				nearest = append(nearest, p)
			}
		}
	}

	return nearest
}

// Helper functions

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func radiansToDegrees(radians float64) float64 {
	return radians * 180 / math.Pi
}
