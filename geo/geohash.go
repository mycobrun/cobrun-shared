// Package geo provides geospatial utilities.
package geo

import (
	"strings"
)

const (
	base32                = "0123456789bcdefghjkmnpqrstuvwxyz"
	maxGeohashPrecision   = 12
	bitsPerChar           = 5
)

// Geohash precision to approximate dimensions.
// Precision 1: ~5000km x 5000km
// Precision 2: ~1250km x 625km
// Precision 3: ~156km x 156km
// Precision 4: ~39km x 19.5km
// Precision 5: ~4.9km x 4.9km
// Precision 6: ~1.2km x 0.6km
// Precision 7: ~153m x 153m
// Precision 8: ~38m x 19m
// Precision 9: ~4.8m x 4.8m

// GeohashPrecisionForRadius returns the recommended precision for a given radius.
func GeohashPrecisionForRadius(radiusKm float64) int {
	switch {
	case radiusKm > 5000:
		return 1
	case radiusKm > 625:
		return 2
	case radiusKm > 78:
		return 3
	case radiusKm > 19.5:
		return 4
	case radiusKm > 2.4:
		return 5
	case radiusKm > 0.6:
		return 6
	case radiusKm > 0.076:
		return 7
	case radiusKm > 0.019:
		return 8
	default:
		return 9
	}
}

// Encode encodes a point to a geohash with the specified precision.
func Encode(p Point, precision int) string {
	if precision < 1 {
		precision = 1
	}
	if precision > maxGeohashPrecision {
		precision = maxGeohashPrecision
	}

	minLat, maxLat := -90.0, 90.0
	minLng, maxLng := -180.0, 180.0

	var hash strings.Builder
	hash.Grow(precision)

	bit := 0
	ch := 0
	isLng := true

	for hash.Len() < precision {
		if isLng {
			mid := (minLng + maxLng) / 2
			if p.Lng >= mid {
				ch |= 1 << (4 - bit)
				minLng = mid
			} else {
				maxLng = mid
			}
		} else {
			mid := (minLat + maxLat) / 2
			if p.Lat >= mid {
				ch |= 1 << (4 - bit)
				minLat = mid
			} else {
				maxLat = mid
			}
		}

		isLng = !isLng
		bit++

		if bit == bitsPerChar {
			hash.WriteByte(base32[ch])
			bit = 0
			ch = 0
		}
	}

	return hash.String()
}

// Decode decodes a geohash to a point (center of the cell).
func Decode(hash string) Point {
	bbox := DecodeBounds(hash)
	return bbox.Center()
}

// DecodeBounds decodes a geohash to its bounding box.
func DecodeBounds(hash string) BoundingBox {
	minLat, maxLat := -90.0, 90.0
	minLng, maxLng := -180.0, 180.0

	isLng := true

	for _, c := range strings.ToLower(hash) {
		idx := strings.IndexRune(base32, c)
		if idx == -1 {
			continue
		}

		for bit := 4; bit >= 0; bit-- {
			if isLng {
				mid := (minLng + maxLng) / 2
				if (idx>>bit)&1 == 1 {
					minLng = mid
				} else {
					maxLng = mid
				}
			} else {
				mid := (minLat + maxLat) / 2
				if (idx>>bit)&1 == 1 {
					minLat = mid
				} else {
					maxLat = mid
				}
			}
			isLng = !isLng
		}
	}

	return BoundingBox{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLng: minLng,
		MaxLng: maxLng,
	}
}

// Neighbors returns the 8 neighboring geohashes.
func Neighbors(hash string) []string {
	return []string{
		neighbor(hash, 1, 0),  // North
		neighbor(hash, 1, 1),  // Northeast
		neighbor(hash, 0, 1),  // East
		neighbor(hash, -1, 1), // Southeast
		neighbor(hash, -1, 0), // South
		neighbor(hash, -1, -1), // Southwest
		neighbor(hash, 0, -1),  // West
		neighbor(hash, 1, -1),  // Northwest
	}
}

// NeighborsWithCenter returns the center hash plus its 8 neighbors.
func NeighborsWithCenter(hash string) []string {
	return append([]string{hash}, Neighbors(hash)...)
}

var (
	neighborTable = map[bool]map[int]string{
		false: { // Even
			1:  "p0r21436x8zb9dcf5h7kjnmqesgutwvy", // North
			-1: "14365h7k9dcfesgujnmqp0r2twvyx8zb", // South
			0:  "bc01fg45238967deuvhjyznpkmstqrwx", // East/West varies
		},
		true: { // Odd
			1:  "bc01fg45238967deuvhjyznpkmstqrwx", // North
			-1: "238967debc01fg45telekmstqrwxuvhjyznp", // South
			0:  "p0r21436x8zb9dcf5h7kjnmqesgutwvy", // East/West varies
		},
	}
	borderTable = map[bool]map[int]string{
		false: {
			1:  "prxz",
			-1: "028b",
			0:  "bcfguvyz",
		},
		true: {
			1:  "bcfguvyz",
			-1: "0145hjnp",
			0:  "prxz",
		},
	}
)

func neighbor(hash string, latDir, lngDir int) string {
	if hash == "" {
		return ""
	}

	// Start with the original hash
	result := hash

	// Handle latitude direction
	if latDir != 0 {
		result = adjacentHash(result, latDir, false)
	}

	// Handle longitude direction
	if lngDir != 0 {
		result = adjacentHash(result, lngDir, true)
	}

	return result
}

func adjacentHash(hash string, dir int, isLng bool) string {
	if hash == "" {
		return ""
	}

	lastChar := hash[len(hash)-1]
	parent := hash[:len(hash)-1]
	isOdd := len(hash)%2 == 1

	// Determine which table to use based on direction
	tableDir := dir
	if isLng {
		tableDir = 0
	}

	// Check if we need to recurse to parent
	borders := borderTable[isOdd != isLng][tableDir]
	if strings.ContainsRune(borders, rune(lastChar)) && parent != "" {
		parent = adjacentHash(parent, dir, isLng)
	}

	// Find neighbor character
	neighbors := neighborTable[isOdd != isLng][tableDir]
	idx := strings.IndexByte(base32, lastChar)
	if idx == -1 || idx >= len(neighbors) {
		return hash
	}

	return parent + string(neighbors[idx])
}

// GeohashRange represents a range of geohashes for efficient queries.
type GeohashRange struct {
	Prefix   string
	Hashes   []string
	Precision int
}

// CoverBoundingBox returns geohashes that cover a bounding box.
func CoverBoundingBox(bbox BoundingBox, precision int) []string {
	var hashes []string
	seen := make(map[string]bool)

	// Sample points across the bounding box
	latStep := (bbox.MaxLat - bbox.MinLat) / 10
	lngStep := (bbox.MaxLng - bbox.MinLng) / 10

	for lat := bbox.MinLat; lat <= bbox.MaxLat; lat += latStep {
		for lng := bbox.MinLng; lng <= bbox.MaxLng; lng += lngStep {
			hash := Encode(Point{Lat: lat, Lng: lng}, precision)
			if !seen[hash] {
				seen[hash] = true
				hashes = append(hashes, hash)
			}
		}
	}

	return hashes
}

// CoverRadius returns geohashes that cover a circular area.
func CoverRadius(center Point, radiusKm float64) []string {
	precision := GeohashPrecisionForRadius(radiusKm)
	centerHash := Encode(center, precision)

	// Get center and neighbors
	hashes := NeighborsWithCenter(centerHash)

	// For larger radii, we may need more hashes
	if radiusKm > 5 {
		// Add additional layer of neighbors
		extraHashes := make(map[string]bool)
		for _, h := range hashes {
			extraHashes[h] = true
			for _, n := range Neighbors(h) {
				extraHashes[n] = true
			}
		}

		hashes = make([]string, 0, len(extraHashes))
		for h := range extraHashes {
			hashes = append(hashes, h)
		}
	}

	return hashes
}
