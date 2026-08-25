package helper

import (
	"math"
	"testing"
)

func TestPointInRectangleToGeo(t *testing.T) {
	// Define a test rectangle in Singapore (approximately 500m x 500m)
	// Using Marina Bay area as an example
	topLeft := GeoPoint{Lat: 1.2850, Lng: 103.8540}
	topRight := GeoPoint{Lat: 1.2850, Lng: 103.8595}
	bottomLeft := GeoPoint{Lat: 1.2805, Lng: 103.8540}
	bottomRight := GeoPoint{Lat: 1.2805, Lng: 103.8595}

	tests := []struct {
		name        string
		x           float64
		y           float64
		width       float64
		height      float64
		expectError bool
	}{
		{
			name:        "Center of rectangle",
			x:           500,
			y:           500,
			width:       1000,
			height:      1000,
			expectError: false,
		},
		{
			name:        "Top-left corner",
			x:           0,
			y:           0,
			width:       1000,
			height:      1000,
			expectError: false,
		},
		{
			name:        "Top-right corner",
			x:           1000,
			y:           0,
			width:       1000,
			height:      1000,
			expectError: false,
		},
		{
			name:        "Bottom-left corner",
			x:           0,
			y:           1000,
			width:       1000,
			height:      1000,
			expectError: false,
		},
		{
			name:        "Bottom-right corner",
			x:           1000,
			y:           1000,
			width:       1000,
			height:      1000,
			expectError: false,
		},
		{
			name:        "Invalid x coordinate (negative)",
			x:           -10,
			y:           500,
			width:       1000,
			height:      1000,
			expectError: true,
		},
		{
			name:        "Invalid x coordinate (exceeds width)",
			x:           1100,
			y:           500,
			width:       1000,
			height:      1000,
			expectError: true,
		},
		{
			name:        "Invalid y coordinate (negative)",
			x:           500,
			y:           -10,
			width:       1000,
			height:      1000,
			expectError: true,
		},
		{
			name:        "Invalid y coordinate (exceeds height)",
			x:           500,
			y:           1100,
			width:       1000,
			height:      1000,
			expectError: true,
		},
		{
			name:        "Invalid width (zero)",
			x:           500,
			y:           500,
			width:       0,
			height:      1000,
			expectError: true,
		},
		{
			name:        "Invalid height (negative)",
			x:           500,
			y:           500,
			width:       1000,
			height:      -100,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PointInRectangleToGeo(
				topLeft, topRight, bottomLeft, bottomRight,
				tt.x, tt.y, tt.width, tt.height,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify the result is within reasonable bounds
			if result.Lat < bottomLeft.Lat || result.Lat > topLeft.Lat {
				t.Errorf("Latitude %f is out of bounds [%f, %f]",
					result.Lat, bottomLeft.Lat, topLeft.Lat)
			}

			if result.Lng < topLeft.Lng || result.Lng > topRight.Lng {
				t.Errorf("Longitude %f is out of bounds [%f, %f]",
					result.Lng, topLeft.Lng, topRight.Lng)
			}

			// For corner tests, verify we get approximately the expected corner
			epsilon := 0.0001 // Acceptable error margin
			switch tt.name {
			case "Top-left corner":
				if math.Abs(result.Lat-topLeft.Lat) > epsilon || math.Abs(result.Lng-topLeft.Lng) > epsilon {
					t.Errorf("Expected top-left corner %v, got %v", topLeft, result)
				}
			case "Top-right corner":
				if math.Abs(result.Lat-topRight.Lat) > epsilon || math.Abs(result.Lng-topRight.Lng) > epsilon {
					t.Errorf("Expected top-right corner %v, got %v", topRight, result)
				}
			case "Bottom-left corner":
				if math.Abs(result.Lat-bottomLeft.Lat) > epsilon || math.Abs(result.Lng-bottomLeft.Lng) > epsilon {
					t.Errorf("Expected bottom-left corner %v, got %v", bottomLeft, result)
				}
			case "Bottom-right corner":
				if math.Abs(result.Lat-bottomRight.Lat) > epsilon || math.Abs(result.Lng-bottomRight.Lng) > epsilon {
					t.Errorf("Expected bottom-right corner %v, got %v", bottomRight, result)
				}
			}

			t.Logf("Result: Lat=%f, Lng=%f", result.Lat, result.Lng)
		})
	}
}

func TestDistance(t *testing.T) {
	tests := []struct {
		name             string
		point1           GeoPoint
		point2           GeoPoint
		expectedDistance float64
		tolerance        float64
	}{
		{
			name:             "Same point",
			point1:           GeoPoint{Lat: 1.2850, Lng: 103.8540},
			point2:           GeoPoint{Lat: 1.2850, Lng: 103.8540},
			expectedDistance: 0,
			tolerance:        0.1,
		},
		{
			name:             "Approximately 500m apart",
			point1:           GeoPoint{Lat: 1.2850, Lng: 103.8540},
			point2:           GeoPoint{Lat: 1.2805, Lng: 103.8540},
			expectedDistance: 500,
			tolerance:        50, // 50m tolerance
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := Distance(tt.point1, tt.point2)
			diff := math.Abs(distance - tt.expectedDistance)

			if diff > tt.tolerance {
				t.Errorf("Expected distance ~%fm, got %fm (diff: %fm)",
					tt.expectedDistance, distance, diff)
			}

			t.Logf("Distance: %fm", distance)
		})
	}
}

func TestPointInRectangleToGeo_RealWorldExample(t *testing.T) {
	// Real world example: A building site in Singapore
	// Rectangle approximately 800m x 600m
	topLeft := GeoPoint{Lat: 1.3000, Lng: 103.8500}
	topRight := GeoPoint{Lat: 1.3000, Lng: 103.8572}
	bottomLeft := GeoPoint{Lat: 1.2928, Lng: 103.8500}
	bottomRight := GeoPoint{Lat: 1.2928, Lng: 103.8572}

	// Point at 1/4 from left, 1/3 from top in a 800x600 pixel image
	x := 200.0 // 1/4 of 800
	y := 200.0 // 1/3 of 600
	width := 800.0
	height := 600.0

	result, err := PointInRectangleToGeo(
		topLeft, topRight, bottomLeft, bottomRight,
		x, y, width, height,
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify result is within rectangle bounds
	if result.Lat < bottomLeft.Lat || result.Lat > topLeft.Lat {
		t.Errorf("Latitude out of bounds")
	}
	if result.Lng < topLeft.Lng || result.Lng > topRight.Lng {
		t.Errorf("Longitude out of bounds")
	}

	// Calculate expected approximate values
	expectedLat := topLeft.Lat - (topLeft.Lat-bottomLeft.Lat)*y/height
	expectedLng := topLeft.Lng + (topRight.Lng-topLeft.Lng)*x/width

	// Check if within reasonable tolerance
	epsilon := 0.0001
	if math.Abs(result.Lat-expectedLat) > epsilon {
		t.Errorf("Latitude mismatch: expected ~%f, got %f", expectedLat, result.Lat)
	}
	if math.Abs(result.Lng-expectedLng) > epsilon {
		t.Errorf("Longitude mismatch: expected ~%f, got %f", expectedLng, result.Lng)
	}

	t.Logf("Input: x=%f, y=%f (in %fx%f rectangle)", x, y, width, height)
	t.Logf("Output: Lat=%f, Lng=%f", result.Lat, result.Lng)
	t.Logf("Expected: Lat=%f, Lng=%f", expectedLat, expectedLng)
}
