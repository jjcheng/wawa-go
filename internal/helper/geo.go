package helper

import (
	"fmt"
	"math"
)

// GeoPoint represents a geographic coordinate
type GeoPoint struct {
	Lat float64
	Lng float64
}

// PointInRectangleToGeo converts a point (x, y) inside a rectangle to geographic coordinates (lat, lng).
// The rectangle is defined by 4 corner points in geographic coordinates.
//
// Parameters:
//   - topLeft: Top-left corner of the rectangle (highest lat, lowest lng)
//   - topRight: Top-right corner of the rectangle (highest lat, highest lng)
//   - bottomLeft: Bottom-left corner of the rectangle (lowest lat, lowest lng)
//   - bottomRight: Bottom-right corner of the rectangle (lowest lat, highest lng)
//   - x: X coordinate within the rectangle (0 to width)
//   - y: Y coordinate within the rectangle (0 to height)
//   - width: Width of the rectangle in the coordinate system
//   - height: Height of the rectangle in the coordinate system
//
// Returns:
//   - GeoPoint: The geographic coordinates of the point
//   - error: Error if the input is invalid
//
// The function uses bilinear interpolation to map the x,y coordinates to lat,lng.
// The rectangle should be small (within 1km) for accurate results.
func PointInRectangleToGeo(topLeft, topRight, bottomLeft, bottomRight GeoPoint, x, y, width, height float64) (GeoPoint, error) {
	// Validate input
	if width <= 0 || height <= 0 {
		return GeoPoint{}, fmt.Errorf("width and height must be positive")
	}

	if x < 0 || x > width {
		return GeoPoint{}, fmt.Errorf("x coordinate must be between 0 and width")
	}

	if y < 0 || y > height {
		return GeoPoint{}, fmt.Errorf("y coordinate must be between 0 and height")
	}

	// Normalize x and y to [0, 1] range
	normalizedX := x / width
	normalizedY := y / height

	// Bilinear interpolation
	// First, interpolate along the top edge (between topLeft and topRight)
	topLat := topLeft.Lat + normalizedX*(topRight.Lat-topLeft.Lat)
	topLng := topLeft.Lng + normalizedX*(topRight.Lng-topLeft.Lng)

	// Then, interpolate along the bottom edge (between bottomLeft and bottomRight)
	bottomLat := bottomLeft.Lat + normalizedX*(bottomRight.Lat-bottomLeft.Lat)
	bottomLng := bottomLeft.Lng + normalizedX*(bottomRight.Lng-bottomLeft.Lng)

	// Finally, interpolate between the top and bottom points
	resultLat := topLat + normalizedY*(bottomLat-topLat)
	resultLng := topLng + normalizedY*(bottomLng-topLng)

	return GeoPoint{
		Lat: resultLat,
		Lng: resultLng,
	}, nil
}

// Distance calculates the distance between two geographic points in meters using the Haversine formula
func Distance(point1, point2 GeoPoint) float64 {
	const earthRadius = 6371000 // Earth's radius in meters

	// Convert latitude and longitude to radians
	lat1Rad := point1.Lat * math.Pi / 180
	lat2Rad := point2.Lat * math.Pi / 180
	deltaLatRad := (point2.Lat - point1.Lat) * math.Pi / 180
	deltaLngRad := (point2.Lng - point1.Lng) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLatRad/2)*math.Sin(deltaLatRad/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLngRad/2)*math.Sin(deltaLngRad/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadius * c
	return distance
}
