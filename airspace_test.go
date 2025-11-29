package airspace

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var data = `
airspace:
- name: ABERDEEN CTA
  id: aberdeen-cta
  type: CTA
  class: D
  geometry:
  - seqno: 1
    upper: FL115
    lower: 1500 ft
    boundary:
    - line:
      - 572153N 0015835W
      - 572100N 0015802W
      - 572100N 0023356W
    - arc:
        dir: cw
        radius: 10 nm
        centre: 571834N 0021602W
        to: 572153N 0015835W
  - seqno: 2
    upper: FL115
    lower: 1500 ft
    boundary:
    - line:
      - 571522N 0015428W
      - 570845N 0015019W
    - arc:
        dir: cw
        radius: 10 nm
        centre: 570531N 0020740W
        to: 570214N 0022458W
    - line:
      - 570850N 0022913W
    - arc:
        dir: ccw
        radius: 10 nm
        centre: 571207N 0021152W
        to: 571522N 0015428W
  - seqno: 3
    upper: FL115
    lower: 3000 ft
    boundary:
    - line:
      - 572100N 0023356W
      - 570015N 0025056W
      - 565433N 0023557W
      - 565533N 0020635W
    - arc:
        dir: cw
        radius: 10 nm
        centre: 570531N 0020740W
        to: 570214N 0022458W
    - line:
      - 571520N 0023326W
    - arc:
        dir: cw
        radius: 10 nm
        centre: 571834N 0021602W
        to: 572100N 0023356W


`

func TestDecode(t *testing.T) {
	features, err := Decode([]byte(data))
	require.NoError(t, err)
	assert.Equal(t, "aberdeen-cta", features[0].ID)
	assert.Equal(t, "D", features[0].Class)
	assert.Equal(t, 3, len(features[0].Geometry))
	assert.Equal(t, 11500.0, features[0].Geometry[0].Upper)
	assert.Equal(t, 1500.0, features[0].Geometry[0].Lower)
	assert.Equal(t, 3, len(features[0].Geometry))
	assert.Equal(t, Circle{}, features[0].Geometry[0].Circle)
	assert.Equal(t, 19, len(features[0].Geometry[0].Polygon))
}

func TestDecodeInvalidYAML(t *testing.T) {
	// Test that invalid YAML is properly rejected
	invalidYAML := []byte(`this is not: valid: yaml: {{{`)
	_, err := Decode(invalidYAML)
	assert.Error(t, err, "Should return error for invalid YAML")
	assert.Contains(t, err.Error(), "failed to unmarshal YAML", "Error should mention YAML unmarshaling")
}

func TestDecodeEmptyData(t *testing.T) {
	// Test that empty data doesn't cause a panic
	features, err := Decode([]byte(""))
	require.NoError(t, err, "Empty YAML should not error")
	assert.Empty(t, features, "Should return empty feature list")
}

func TestDownload(t *testing.T) {
	// Verify real-life data exists and can be parsed correctly.
	url := `https://raw.githubusercontent.com/ahsparrow/airspace/master/airspace.yaml`
	a, err := Load(url)
	require.NoError(t, err)

	assert.Greater(t, len(a), 600)
}

func Test_arcToPolygon(t *testing.T) {
	// Use a simple center point for testing
	centre := orb.Point{0.0, 51.0} // 0°E, 51°N (roughly London)
	radius := 10000.0              // 10km in meters

	t.Run("Clockwise 90-degree arc", func(t *testing.T) {
		// Arc from North to East (90 degrees clockwise)
		initialPoint := destinationPoint(centre, 0, radius)   // North
		to := destinationPoint(centre, 90, radius)            // East
		result := arcToPolygon(centre, radius, initialPoint, to, 1.0)

		// Should generate points every 10 degrees: 0, 10, 20, 30, 40, 50, 60, 70, 80, plus final point
		assert.GreaterOrEqual(t, len(result), 9, "Should have at least 9 points for 90-degree arc")
		assert.LessOrEqual(t, len(result), 11, "Should have at most 11 points for 90-degree arc")

		// Last point should be the exact 'to' point
		assert.Equal(t, to, result[len(result)-1], "Last point should match 'to' point exactly")

		// First point should be near the initial point
		assert.InDelta(t, initialPoint.Lat(), result[0].Lat(), 0.01, "First point should be near initial latitude")
		assert.InDelta(t, initialPoint.Lon(), result[0].Lon(), 0.01, "First point should be near initial longitude")
	})

	t.Run("Counter-clockwise 90-degree arc", func(t *testing.T) {
		// Arc from North to West (90 degrees counter-clockwise)
		initialPoint := destinationPoint(centre, 0, radius)   // North
		to := destinationPoint(centre, 270, radius)           // West
		result := arcToPolygon(centre, radius, initialPoint, to, -1.0)

		// Should generate points every 10 degrees
		assert.GreaterOrEqual(t, len(result), 9, "Should have at least 9 points for 90-degree arc")
		assert.LessOrEqual(t, len(result), 11, "Should have at most 11 points for 90-degree arc")

		// Last point should be the exact 'to' point
		assert.Equal(t, to, result[len(result)-1], "Last point should match 'to' point exactly")
	})

	t.Run("Clockwise 180-degree arc", func(t *testing.T) {
		// Arc from North to South (180 degrees clockwise)
		initialPoint := destinationPoint(centre, 0, radius)   // North
		to := destinationPoint(centre, 180, radius)           // South
		result := arcToPolygon(centre, radius, initialPoint, to, 1.0)

		// Should generate points every 10 degrees: 18 intermediate points + final
		assert.GreaterOrEqual(t, len(result), 18, "Should have at least 18 points for 180-degree arc")
		assert.LessOrEqual(t, len(result), 20, "Should have at most 20 points for 180-degree arc")

		// Last point should be the exact 'to' point
		assert.Equal(t, to, result[len(result)-1], "Last point should match 'to' point exactly")
	})

	t.Run("Small clockwise arc (30 degrees)", func(t *testing.T) {
		// Small arc from bearing 45 to bearing 75
		initialPoint := destinationPoint(centre, 45, radius)
		to := destinationPoint(centre, 75, radius)
		result := arcToPolygon(centre, radius, initialPoint, to, 1.0)

		// Should have at least a few points (45, 55, 65, 75)
		assert.GreaterOrEqual(t, len(result), 3, "Should have at least 3 points for 30-degree arc")
		assert.LessOrEqual(t, len(result), 5, "Should have at most 5 points for 30-degree arc")

		// Last point should be exact 'to' point
		assert.Equal(t, to, result[len(result)-1], "Last point should match 'to' point exactly")
	})

	t.Run("Clockwise arc crossing 0 degrees", func(t *testing.T) {
		// Arc from 350 degrees to 10 degrees (crossing north)
		initialPoint := destinationPoint(centre, 350, radius)
		to := destinationPoint(centre, 10, radius)
		result := arcToPolygon(centre, radius, initialPoint, to, 1.0)

		// Should wrap around correctly
		assert.GreaterOrEqual(t, len(result), 2, "Should have at least 2 points")
		assert.LessOrEqual(t, len(result), 5, "Should have at most 5 points for 20-degree arc")

		// Last point should be exact 'to' point
		assert.Equal(t, to, result[len(result)-1], "Last point should match 'to' point exactly")
	})

	t.Run("Counter-clockwise arc crossing 0 degrees", func(t *testing.T) {
		// Arc from 10 degrees to 350 degrees (counter-clockwise, crossing north)
		initialPoint := destinationPoint(centre, 10, radius)
		to := destinationPoint(centre, 350, radius)
		result := arcToPolygon(centre, radius, initialPoint, to, -1.0)

		// Should wrap around correctly
		assert.GreaterOrEqual(t, len(result), 2, "Should have at least 2 points")
		assert.LessOrEqual(t, len(result), 5, "Should have at most 5 points for 20-degree arc")

		// Last point should be exact 'to' point
		assert.Equal(t, to, result[len(result)-1], "Last point should match 'to' point exactly")
	})

	t.Run("Large radius arc", func(t *testing.T) {
		// Test with a large radius (50km) - just verify the arc is generated correctly
		largeRadius := 50000.0
		initialPoint := destinationPoint(centre, 0, largeRadius)
		to := destinationPoint(centre, 45, largeRadius)
		result := arcToPolygon(centre, largeRadius, initialPoint, to, 1.0)

		// Should still generate points every 10 degrees (0, 10, 20, 30, 40, plus final at 45)
		assert.GreaterOrEqual(t, len(result), 4, "Should have at least 4 points for 45-degree arc")
		assert.LessOrEqual(t, len(result), 7, "Should have at most 7 points for 45-degree arc")

		// Last point should be exact 'to' point
		assert.Equal(t, to, result[len(result)-1], "Last point should match 'to' point exactly")

		// First point should be near the initial point
		assert.InDelta(t, initialPoint.Lat(), result[0].Lat(), 0.01, "First point should be near initial latitude")
		assert.InDelta(t, initialPoint.Lon(), result[0].Lon(), 0.01, "First point should be near initial longitude")
	})

	t.Run("Nearly complete circle clockwise", func(t *testing.T) {
		// Arc from 10 degrees to 5 degrees clockwise (355 degree arc)
		initialPoint := destinationPoint(centre, 10, radius)
		to := destinationPoint(centre, 5, radius)
		result := arcToPolygon(centre, radius, initialPoint, to, 1.0)

		// Should have many points (approximately 355/10 = 35 points)
		assert.GreaterOrEqual(t, len(result), 34, "Should have at least 34 points for 355-degree arc")
		assert.LessOrEqual(t, len(result), 37, "Should have at most 37 points for 355-degree arc")

		// Last point should be exact 'to' point
		assert.Equal(t, to, result[len(result)-1], "Last point should match 'to' point exactly")
	})
}

// TestParseLatLng verifies coordinate parsing, especially the fix for the seconds conversion bug.
// Previously used 2600.0 instead of 3600.0, causing incorrect coordinates.
func TestParseLatLng(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLon  float64
		wantLat  float64
		wantErr  bool
	}{
		{
			name:    "Basic North/West coordinate",
			input:   "502257N 0033739W",
			wantLat: 50.0 + 22.0/60.0 + 57.0/3600.0, // 50.3825°N
			wantLon: -(3.0 + 37.0/60.0 + 39.0/3600.0), // -3.6275°W (003°37'39"W)
			wantErr: false,
		},
		{
			name:    "North/East coordinate",
			input:   "513045N 0001530E",
			wantLat: 51.0 + 30.0/60.0 + 45.0/3600.0, // 51.5125°N
			wantLon: 0.0 + 15.0/60.0 + 30.0/3600.0,   // 0.2583°E
			wantErr: false,
		},
		{
			name:    "South/East coordinate",
			input:   "012345S 1234500E",
			wantLat: -(1.0 + 23.0/60.0 + 45.0/3600.0), // -1.3958°S
			wantLon: 123.0 + 45.0/60.0 + 0.0/3600.0,    // 123.75°E
			wantErr: false,
		},
		{
			name:    "Exact degrees (no minutes or seconds)",
			input:   "500000N 0000000W",
			wantLat: 50.0,
			wantLon: 0.0,
			wantErr: false,
		},
		{
			name:    "Aberdeen CTA point from test data",
			input:   "572153N 0015835W",
			wantLat: 57.0 + 21.0/60.0 + 53.0/3600.0,
			wantLon: -(1.0 + 58.0/60.0 + 35.0/3600.0),
			wantErr: false,
		},
		{
			name:    "Invalid length",
			input:   "5020N 00330W",
			wantErr: true,
		},
		{
			name:    "Missing space separator",
			input:   "502257N00033739W",
			wantErr: true,
		},
		{
			name:    "Invalid hemisphere (X instead of N/S)",
			input:   "502257X 0033739W",
			wantErr: true,
		},
		{
			name:    "Invalid hemisphere (Z instead of E/W)",
			input:   "502257N 0033739Z",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLatLng(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			// Check longitude and latitude with small tolerance for floating point
			assert.InDelta(t, tt.wantLon, got.Lon(), 0.000001, "Longitude mismatch")
			assert.InDelta(t, tt.wantLat, got.Lat(), 0.000001, "Latitude mismatch")
		})
	}
}

// TestDecodeHeight verifies altitude parsing for various formats
func TestDecodeHeight(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{"Surface", "SFC", 0},
		{"Surface lowercase", "sfc", 0},
		{"Empty string", "", 0},
		{"Flight level 115", "FL115", 11500},
		{"Flight level lowercase", "fl85", 8500},
		{"Feet with suffix", "1500 ft", 1500},
		{"Feet with uppercase suffix", "1500 FT", 1500},
		{"Feet without suffix", "3000", 3000},
		{"High flight level", "FL195", 19500},
		{"Low altitude", "500 ft", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeHeight(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDecodeDistance verifies distance parsing from nautical miles to meters
func TestDecodeDistance(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{"10 nautical miles", "10 nm", 18520}, // 10 * 1852
		{"5 nautical miles", "5 nm", 9260},    // 5 * 1852
		{"1 nautical mile", "1 nm", 1852},
		{"Fractional nm", "0.5 nm", 926},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeDistance(tt.input)
			assert.InDelta(t, tt.want, got, 0.1)
		})
	}
}

// TestResolveAirspaceType verifies type resolution for OTHER and D_OTHER
func TestResolveAirspaceType(t *testing.T) {
	tests := []struct {
		name          string
		airspaceType  string
		localType     string
		expectedType  string
	}{
		{"Normal type", "CTR", "IGNORED", "CTR"},
		{"OTHER uses LocalType", "OTHER", "MATZ", "MATZ"},
		{"D_OTHER uses LocalType", "D_OTHER", "GVS", "GVS"},
		{"Regular type ignores LocalType", "CTA", "IGNORED", "CTA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAirspaceType(tt.airspaceType, tt.localType)
			assert.Equal(t, tt.expectedType, got)
		})
	}
}

// TestResolveFeatureID verifies ID generation for features without explicit IDs
func TestResolveFeatureID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		featName string
		index    int
		expected string
	}{
		{"Explicit ID used", "london-ctr", "LONDON CTR", 0, "london-ctr"},
		{"Whitespace trimmed", "  aberdeen-cta  ", "ABERDEEN CTA", 0, "aberdeen-cta"},
		{"Generated from name", "", "Drop Zone Alpha", 5, "drop-zone-alpha-5"},
		{"Generated with different index", "", "Test Area", 42, "test-area-42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFeatureID(tt.id, tt.featName, tt.index)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestClearanceRequired verifies airspace classification
func TestClearanceRequired(t *testing.T) {
	tests := []struct {
		name     string
		feature  Feature
		expected bool
	}{
		{"Class D requires clearance", Feature{Class: "D", Type: "CTR"}, true},
		{"CTR requires clearance", Feature{Class: "G", Type: "CTR"}, true},
		{"Class G does not require clearance", Feature{Class: "G", Type: "OTHER"}, false},
		{"P (prohibited) requires clearance", Feature{Class: "", Type: "P"}, true},
		{"R (restricted) requires clearance", Feature{Class: "", Type: "R"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClearanceRequired(tt.feature)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestDanger verifies danger area classification
func TestDanger(t *testing.T) {
	tests := []struct {
		name     string
		feature  Feature
		expected bool
	}{
		{"D (danger) is danger", Feature{Type: "D"}, true},
		{"DZ (drop zone) is danger", Feature{Type: "DZ"}, true},
		{"GLIDER is danger", Feature{Type: "GLIDER"}, true},
		{"GVS is not danger", Feature{Type: "GVS"}, false},
		{"ILS is not danger", Feature{Type: "ILS"}, false},
		{"CTR is not danger", Feature{Type: "CTR"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Danger(tt.feature)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestDestinationPoint verifies spherical geometry calculations
func TestDestinationPoint(t *testing.T) {
	// Test with known values - going 1000m due north from a point
	start := orb.Point{0.0, 50.0} // 0°E, 50°N

	// Going north (bearing 0°) should increase latitude
	north := destinationPoint(start, 0, 1000)
	assert.Greater(t, north.Lat(), start.Lat(), "Going north should increase latitude")
	assert.InDelta(t, start.Lon(), north.Lon(), 0.0001, "Going north should not change longitude much")

	// Going east (bearing 90°) should increase longitude
	east := destinationPoint(start, 90, 1000)
	assert.Greater(t, east.Lon(), start.Lon(), "Going east should increase longitude")
	assert.InDelta(t, start.Lat(), east.Lat(), 0.0001, "Going east should not change latitude much")

	// Going south (bearing 180°) should decrease latitude
	south := destinationPoint(start, 180, 1000)
	assert.Less(t, south.Lat(), start.Lat(), "Going south should decrease latitude")

	// Going west (bearing 270°) should decrease longitude
	west := destinationPoint(start, 270, 1000)
	assert.Less(t, west.Lon(), start.Lon(), "Going west should decrease longitude")
}
