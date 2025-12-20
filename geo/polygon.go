// Package geo provides geospatial utilities.
package geo

import (
	"encoding/json"
	"math"
)

// Polygon represents a geographic polygon.
type Polygon struct {
	Points []Point `json:"points"`
}

// NewPolygon creates a new polygon from points.
func NewPolygon(points []Point) *Polygon {
	return &Polygon{Points: points}
}

// Contains checks if a point is inside the polygon using ray casting algorithm.
func (p *Polygon) Contains(point Point) bool {
	if len(p.Points) < 3 {
		return false
	}

	inside := false
	n := len(p.Points)

	j := n - 1
	for i := 0; i < n; i++ {
		pi := p.Points[i]
		pj := p.Points[j]

		if ((pi.Lat > point.Lat) != (pj.Lat > point.Lat)) &&
			(point.Lng < (pj.Lng-pi.Lng)*(point.Lat-pi.Lat)/(pj.Lat-pi.Lat)+pi.Lng) {
			inside = !inside
		}
		j = i
	}

	return inside
}

// BoundingBox returns the bounding box of the polygon.
func (p *Polygon) BoundingBox() BoundingBox {
	if len(p.Points) == 0 {
		return BoundingBox{}
	}

	minLat, maxLat := p.Points[0].Lat, p.Points[0].Lat
	minLng, maxLng := p.Points[0].Lng, p.Points[0].Lng

	for _, pt := range p.Points[1:] {
		if pt.Lat < minLat {
			minLat = pt.Lat
		}
		if pt.Lat > maxLat {
			maxLat = pt.Lat
		}
		if pt.Lng < minLng {
			minLng = pt.Lng
		}
		if pt.Lng > maxLng {
			maxLng = pt.Lng
		}
	}

	return BoundingBox{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLng: minLng,
		MaxLng: maxLng,
	}
}

// Centroid calculates the centroid of the polygon.
func (p *Polygon) Centroid() Point {
	if len(p.Points) == 0 {
		return Point{}
	}

	var sumLat, sumLng float64
	for _, pt := range p.Points {
		sumLat += pt.Lat
		sumLng += pt.Lng
	}

	n := float64(len(p.Points))
	return Point{
		Lat: sumLat / n,
		Lng: sumLng / n,
	}
}

// Area calculates the approximate area of the polygon in square kilometers.
// Uses the Shoelace formula with Earth's radius for approximation.
func (p *Polygon) Area() float64 {
	if len(p.Points) < 3 {
		return 0
	}

	n := len(p.Points)
	var area float64

	for i := 0; i < n; i++ {
		j := (i + 1) % n
		xi := degreesToRadians(p.Points[i].Lng)
		yi := degreesToRadians(p.Points[i].Lat)
		xj := degreesToRadians(p.Points[j].Lng)
		yj := degreesToRadians(p.Points[j].Lat)

		area += xi*yj - xj*yi
	}

	// Convert to km^2 using average radius
	area = math.Abs(area) / 2.0
	areaKm2 := area * EarthRadiusKm * EarthRadiusKm

	return areaKm2
}

// Perimeter calculates the perimeter of the polygon in kilometers.
func (p *Polygon) Perimeter() float64 {
	if len(p.Points) < 2 {
		return 0
	}

	var perimeter float64
	n := len(p.Points)

	for i := 0; i < n; i++ {
		j := (i + 1) % n
		perimeter += HaversineDistance(p.Points[i], p.Points[j])
	}

	return perimeter
}

// IsValid checks if the polygon is valid (at least 3 points, closed).
func (p *Polygon) IsValid() bool {
	if len(p.Points) < 3 {
		return false
	}

	// Check all points are valid
	for _, pt := range p.Points {
		if !pt.IsValid() {
			return false
		}
	}

	return true
}

// GeoJSON support

// GeoJSONPolygon represents a GeoJSON polygon.
type GeoJSONPolygon struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

// ToGeoJSON converts the polygon to GeoJSON format.
func (p *Polygon) ToGeoJSON() GeoJSONPolygon {
	coords := make([][]float64, len(p.Points)+1)

	for i, pt := range p.Points {
		coords[i] = []float64{pt.Lng, pt.Lat} // GeoJSON uses [lng, lat]
	}
	// Close the polygon
	if len(p.Points) > 0 {
		coords[len(p.Points)] = []float64{p.Points[0].Lng, p.Points[0].Lat}
	}

	return GeoJSONPolygon{
		Type:        "Polygon",
		Coordinates: [][][]float64{coords},
	}
}

// FromGeoJSON creates a Polygon from GeoJSON.
func FromGeoJSON(gj GeoJSONPolygon) *Polygon {
	if len(gj.Coordinates) == 0 || len(gj.Coordinates[0]) == 0 {
		return &Polygon{}
	}

	ring := gj.Coordinates[0]
	points := make([]Point, 0, len(ring)-1)

	for i := 0; i < len(ring)-1; i++ { // Skip last point (closing point)
		if len(ring[i]) >= 2 {
			points = append(points, Point{
				Lat: ring[i][1], // GeoJSON uses [lng, lat]
				Lng: ring[i][0],
			})
		}
	}

	return &Polygon{Points: points}
}

// ToJSON serializes the polygon to JSON.
func (p *Polygon) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// PolygonFromJSON deserializes a polygon from JSON.
func PolygonFromJSON(data []byte) (*Polygon, error) {
	var p Polygon
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Geofence operations

// Geofence represents a named geographic boundary.
type Geofence struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"` // service_area, surge_zone, no_pickup, etc.
	Polygon  *Polygon `json:"polygon"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Contains checks if a point is within the geofence.
func (g *Geofence) Contains(point Point) bool {
	if g.Polygon == nil {
		return false
	}
	return g.Polygon.Contains(point)
}

// GeofenceCollection is a collection of geofences.
type GeofenceCollection struct {
	Geofences []*Geofence
}

// NewGeofenceCollection creates a new collection.
func NewGeofenceCollection() *GeofenceCollection {
	return &GeofenceCollection{
		Geofences: make([]*Geofence, 0),
	}
}

// Add adds a geofence to the collection.
func (gc *GeofenceCollection) Add(gf *Geofence) {
	gc.Geofences = append(gc.Geofences, gf)
}

// FindContaining returns all geofences containing a point.
func (gc *GeofenceCollection) FindContaining(point Point) []*Geofence {
	var result []*Geofence
	for _, gf := range gc.Geofences {
		if gf.Contains(point) {
			result = append(result, gf)
		}
	}
	return result
}

// FindByType returns geofences of a specific type containing a point.
func (gc *GeofenceCollection) FindByType(point Point, geofenceType string) []*Geofence {
	var result []*Geofence
	for _, gf := range gc.Geofences {
		if gf.Type == geofenceType && gf.Contains(point) {
			result = append(result, gf)
		}
	}
	return result
}

// IsInServiceArea checks if a point is in any service area.
func (gc *GeofenceCollection) IsInServiceArea(point Point) bool {
	for _, gf := range gc.Geofences {
		if gf.Type == "service_area" && gf.Contains(point) {
			return true
		}
	}
	return false
}
