package airspace

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/paulmach/orb/geo"
	"github.com/paulmach/orb/project"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"gopkg.in/yaml.v2"
)

// Download airspace defs in yaml from https://github.com/ahsparrow/airspace
// Schema is https://github.com/ahsparrow/yaixm/blob/master/yaixm/data/schema.yaml

// Airspace definitions - similar to `airspaceResponse` but sanitised.
// github.com/golang/geo/r2

var _ = planar.Length

type Feature struct {
	ID       string
	Name     string
	Type     string
	Class    string
	Geometry []Volume
}

type Volume struct {
	ID                string
	Name              string
	Type              string
	Class             string
	Sequence          int
	Lower             float64
	Upper             float64
	ClearanceRequired bool
	Danger            bool
	// The (horizontal) shape will be either a circle or a polygon.
	// One of:
	Circle  Circle
	Polygon orb.Ring
}

type Circle struct {
	Radius float64
	Centre orb.Point
}

var (
	prohibitedAirspaceClasses = map[string]bool{
		"A": true,  // Most airways; London/Manchester TMAs.
		"B": true,  // Not used in UK
		"C": true,  // Mostly above FL195 and some airways.
		"D": true,  // Most aerodrome CTRs and CTAs. Some TMAs and lower levels of selected airways.
		"E": true,  // Scottish airways. Technically permissible (ATC clearance not required for VFR flight), but pilots encouraged to contact ATC.
		"F": false, // Not used in UK
		"G": false, // ‘Open FIR’, ATC clearance not required, radio not required.
	}

	// Not all are strictly prohibited, some are "avoid unless ...."
	prohibitedTypes = map[string]bool{
		"ATZ":  true, // Aerodrome Traffic Zone
		"AWY":  true, // Airway
		"CTA":  true, // Control Area (usually on top of a CTR).
		"CTR":  true, // Control Region
		"MATZ": true, // Military ATZ. Technically permissible.
		"P":    true, // Prohibited area
		"R":    true, // Restricted area
		"RAT":  true, // Temporary restricted area
		"RMZ":  true, // Radio mandatory zone
		"TMA":  true, // Terminal control area
		"TRA":  true, // Temporary reserved area
		"TMZ":  true, // Transponder mandatory zone
	}

	dangerTypes = map[string]bool{
		"AIAA":    true,  // Areas of intense arial activity
		"D":       true,  // Danger area
		"D_OTHER": true,  // Dangerous activity, but not a Danger area
		"DZ":      true,  // Drop Zone
		"GLIDER":  true,  // Gliding operations
		"GVS":     false, // Gas venting station
		"HIRTA":   true,  // High intensity radio transmission area
		"ILS":     false, // ILS feather
		"LASER":   true,  // Laser site.
		"NOATZ":   true,  // Non-ATZ airfield
		"UL":      true,  // Ultra-light strip
	}
)

func ClearanceRequired(f Feature) bool {
	return prohibitedAirspaceClasses[f.Class] || prohibitedTypes[f.Type]
}

func Danger(f Feature) bool {
	return dangerTypes[f.Type]
}

// This type is used to decode YAML data from https://raw.githubusercontent.com/ahsparrow/airspace/master/airspace.yaml (and equivalent).
type airspaceResponse struct {
	Airspace []struct {
		ID          string
		Name        string
		Type        string
		LocalType   string
		ControlType string
		Class       string
		Geometry    []struct {
			ID       string
			Name     string
			Class    string
			Seqno    int
			Boundary []struct {
				// One of:
				Circle struct {
					Radius string
					Centre string
				}
				Line []string
				Arc  struct {
					Dir    string
					Radius string
					Centre string
					To     string
				}
			}
			Lower string
			Upper string
		}
	}
}

type ratResponse struct {
	Name     string
	Type     string
	Geometry []struct {
		ID       string
		Name     string
		Class    string
		Seqno    int
		Boundary []struct {
			// One of:
			Circle struct {
				Radius string
				Centre string
			}
			Line []string
			Arc  struct {
				Dir    string
				Radius string
				Centre string
				To     string
			}
		}
		Lower string
		Upper string
	}
}

func Decode(data []byte) ([]Feature, error) {
	var a airspaceResponse
	if err := yaml.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	features, err := normalise(&a)
	return features, err
}

// normalise converts the raw YAML airspace data into our internal Feature representation.
// This involves:
//  1. Resolving airspace type (converting "OTHER" to LocalType)
//  2. Generating IDs for features that don't have explicit IDs
//  3. Converting each geometry volume with its boundaries (circles, lines, arcs)
//  4. Classifying each feature as prohibited or danger
func normalise(a *airspaceResponse) ([]Feature, error) {
	var features []Feature
	for i, f := range a.Airspace {
		// Determine the actual airspace type
		airspaceType := resolveAirspaceType(f.Type, f.LocalType)

		// Generate or use existing ID
		featureID := resolveFeatureID(f.ID, f.Name, i)

		feat := Feature{
			ID:    featureID,
			Name:  f.Name,
			Type:  airspaceType,
			Class: f.Class,
		}

		// Process each geometry volume (a feature can have multiple volumes at different altitudes)
		for _, g := range f.Geometry {
			vol, err := processGeometry(g, feat)
			if err != nil {
				return nil, err
			}
			feat.Geometry = append(feat.Geometry, vol)
		}

		features = append(features, feat)
	}

	return features, nil
}

// resolveAirspaceType determines the actual type, using LocalType for "OTHER" and "D_OTHER" types.
func resolveAirspaceType(airspaceType, localType string) string {
	if airspaceType == "OTHER" || airspaceType == "D_OTHER" {
		return localType
	}
	return airspaceType
}

// resolveFeatureID returns the feature ID, generating one if not provided.
// Generated IDs are based on the feature name with a numeric suffix.
func resolveFeatureID(id, name string, index int) string {
	id = strings.TrimSpace(id)
	if id == "" {
		// Dropzones and similar don't have explicit IDs - generate from name
		safeName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		id = safeName + "-" + strconv.FormatInt(int64(index), 10)
	}
	return id
}

// processGeometry converts a YAML geometry definition into a Volume with parsed boundaries.
func processGeometry(g struct {
	ID       string
	Name     string
	Class    string
	Seqno    int
	Boundary []struct {
		Circle struct {
			Radius string
			Centre string
		}
		Line []string
		Arc  struct {
			Dir    string
			Radius string
			Centre string
			To     string
		}
	}
	Lower string
	Upper string
}, feat Feature) (Volume, error) {

	// Inherit ID, name, and class from parent feature if not specified
	volID := g.ID
	if volID == "" {
		volID = feat.ID
	}
	volName := g.Name
	if volName == "" {
		volName = feat.Name
	}
	volClass := g.Class
	if volClass == "" {
		volClass = feat.Class
	}

	vol := Volume{
		ID:                volID,
		Name:              volName,
		Type:              feat.Type,
		Class:             volClass,
		Sequence:          g.Seqno,
		Lower:             decodeHeight(g.Lower),
		Upper:             decodeHeight(g.Upper),
		ClearanceRequired: ClearanceRequired(feat),
		Danger:            Danger(feat),
	}

	// Process boundary definitions (can be circles, lines, or arcs)
	// currentPos tracks position for connecting arcs to previous line segments
	var currentPos orb.Point

	for _, b := range g.Boundary {
		// Circle boundary (mutually exclusive with polygon boundaries)
		if b.Circle.Radius != "" {
			var err error
			vol.Circle.Radius = decodeDistance(b.Circle.Radius)
			vol.Circle.Centre, err = parseLatLng(b.Circle.Centre)
			if err != nil {
				return Volume{}, fmt.Errorf("bad circle %v: %s", b, err)
			}
		}

		// Line segments (straight lines between points)
		for i := range b.Line {
			p, err := parseLatLng(b.Line[i])
			if err != nil {
				return Volume{}, fmt.Errorf("bad line %v: %s", b, err)
			}
			vol.Polygon = append(vol.Polygon, p)
			currentPos = p // Track position for next arc
		}

		// Arc segments (curved sections between two points around a center)
		if b.Arc.Radius != "" {
			to, err := parseLatLng(b.Arc.To)
			if err != nil {
				return Volume{}, fmt.Errorf("bad arc %v: %s", b, err)
			}
			radius := decodeDistance(b.Arc.Radius)
			centre, _ := parseLatLng(b.Arc.Centre)

			// Direction: clockwise (+1.0) or counter-clockwise (-1.0)
			dir := +1.0
			if b.Arc.Dir == "ccw" {
				dir = -1.0
			}

			// Convert arc to polygon approximation and append to boundary
			arc := arcToPolygon(centre, radius, currentPos, to, dir)
			vol.Polygon = append(vol.Polygon, arc...)
		}
	}

	return vol, nil
}

// arcToPolygon converts an arc boundary into a polygon approximation.
// Arcs are defined by a centre point, radius, start point, end point, and direction.
// The arc is approximated using line segments every 10 degrees.
//
// Parameters:
//   - centre: The center point of the arc
//   - radius: The radius in meters
//   - initialPoint: Where the arc starts (must be on the circle)
//   - to: Where the arc ends (must be on the circle)
//   - dir: Direction of sweep (+1.0 for clockwise, -1.0 for counter-clockwise)
//
// Returns a LineString of points approximating the arc, including the final point.
func arcToPolygon(centre orb.Point, radius float64, initialPoint orb.Point, to orb.Point, dir float64) orb.LineString {
	// Calculate bearings from centre to start and end points
	initialAngleDeg := geo.Bearing(centre, initialPoint)
	finalAngleDeg := geo.Bearing(centre, to)

	// Normalize angles to ensure we sweep in the correct direction
	// For clockwise (dir > 0), if final < initial, add 360° to final
	// For counter-clockwise (dir < 0), if final > initial, add 360° to initial
	if dir > 0 {
		// Clockwise: sweep from initialAngle to finalAngle in positive direction
		if finalAngleDeg < initialAngleDeg {
			finalAngleDeg += 360
		}
	} else {
		// Counter-clockwise: sweep from initialAngle to finalAngle in negative direction
		if finalAngleDeg > initialAngleDeg {
			initialAngleDeg += 360
		}
	}

	// Generate points along the arc every 10 degrees
	var poly orb.LineString
	for a := initialAngleDeg; dir*a < dir*finalAngleDeg; a += dir * 10 {
		point := destinationPoint(centre, a, radius)
		poly = append(poly, point)
	}

	// Always include the exact end point to close the arc precisely
	poly = append(poly, to)

	return poly
}

func toRadians(angle float64) float64 {
	return math.Pi / 180.0 * angle
}

func toDegrees(angle float64) float64 {
	return 180.0 / math.Pi * angle
}

// destinationPoint calculates a destination point given a start point, bearing, and distance.
// This uses the haversine formula for great circle calculations on a sphere.
//
// Parameters:
//   - start: The starting point (lat/lon)
//   - bearing: The bearing in degrees (0° = North, 90° = East, etc.)
//   - distance: The distance to travel in meters
//
// Returns the destination point in WGS84 coordinates.
//
// This is used to generate points along arcs at specific angles from the arc's center.
func destinationPoint(start orb.Point, bearing float64, distance float64) orb.Point {
	// Convert distance to angular distance (radians) by dividing by Earth's radius
	angularDistance := distance / orb.EarthRadius

	// Convert all angles to radians for trigonometric calculations
	bearingRadians := toRadians(bearing)
	lat1 := toRadians(start.Lat())
	lon1 := toRadians(start.Lon())

	// Calculate destination latitude using spherical trigonometry
	// Formula: lat2 = asin(sin(lat1)*cos(d) + cos(lat1)*sin(d)*cos(bearing))
	sinLat2 := math.Sin(lat1)*math.Cos(angularDistance) +
		math.Cos(lat1)*math.Sin(angularDistance)*math.Cos(bearingRadians)
	lat2 := math.Asin(sinLat2)

	// Calculate destination longitude
	// Formula: lon2 = lon1 + atan2(sin(bearing)*sin(d)*cos(lat1), cos(d) - sin(lat1)*sin(lat2))
	x := math.Cos(angularDistance) - math.Sin(lat1)*sinLat2
	y := math.Sin(bearingRadians) * math.Sin(angularDistance) * math.Cos(lat1)
	lon2 := lon1 + math.Atan2(y, x)

	// Convert back to degrees and return as Point{lon, lat}
	return orb.Point{toDegrees(lon2), toDegrees(lat2)}
}

// parseLatLng converts airspace coordinate strings to WGS84 lat/lon points.
// Expected format: "DDMMSSN DDDMMSSX" where:
//   - DD/DDD = degrees (2 digits for lat, 3 for lon)
//   - MM = minutes (2 digits)
//   - SS = seconds (2 digits)
//   - N/S = hemisphere for latitude
//   - E/W = hemisphere for longitude
// Example: "502257N 0033739W" = 50°22'57"N 003°37'39"W
func parseLatLng(str string) (orb.Point, error) {
	const expectedFormat = "502257N 0033739W"
	formatError := fmt.Errorf("bad point: %#q, must be in format %q (degrees,minutes,seconds)", str, expectedFormat)

	// Validate length and space separator
	if len(str) != 16 || str[7] != ' ' {
		return orb.Point{}, formatError
	}

	// Parse latitude (DDMMSSN)
	latDeg, err1 := strconv.ParseUint(str[0:2], 10, 64)
	latMin, err2 := strconv.ParseUint(str[2:4], 10, 64)
	latSec, err3 := strconv.ParseUint(str[4:6], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return orb.Point{}, formatError
	}

	// Convert to decimal degrees (60 minutes/degree, 3600 seconds/degree)
	lat := float64(latDeg) + float64(latMin)/60.0 + float64(latSec)/3600.0

	// Apply hemisphere
	hemisphereNS := str[6]
	if hemisphereNS == 'S' {
		lat = -lat
	} else if hemisphereNS != 'N' {
		return orb.Point{}, formatError
	}

	// Parse longitude (DDDMMSSX)
	lonDeg, err1 := strconv.ParseUint(str[8:11], 10, 64)
	lonMin, err2 := strconv.ParseUint(str[11:13], 10, 64)
	lonSec, err3 := strconv.ParseUint(str[13:15], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return orb.Point{}, formatError
	}

	// Convert to decimal degrees
	lon := float64(lonDeg) + float64(lonMin)/60.0 + float64(lonSec)/3600.0

	// Apply hemisphere
	hemisphereEW := str[15]
	if hemisphereEW == 'W' {
		lon = -lon
	} else if hemisphereEW != 'E' {
		return orb.Point{}, formatError
	}

	// Note: orb.Point is {lon, lat} - longitude comes first!
	return orb.Point{lon, lat}, nil
}

func decodeHeight(h string) float64 {
	h = strings.ToUpper(strings.TrimSpace(h))
	if h == "" || h == "SFC" {
		return 0
	}

	if strings.HasPrefix(h, "FL") {
		// Flight level.
		f, err := strconv.ParseFloat(h[2:], 64)
		if err != nil {
			log.Printf("Could not parse flight levele %#q: %s\n", h, err)
		}
		return f * 100 // Standard pressure and so on.
	}

	h = strings.TrimSpace(strings.TrimSuffix(h, "FT"))
	f, err := strconv.ParseFloat(h, 64)
	if err != nil {
		log.Printf("Could not parse height %#q: %s\n", h, err)
	}
	return f
}

func decodeDistance(d string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSuffix(d, " nm"), 64)
	if err != nil {
		log.Printf("Invalid distance %#q: %s\n", d, err)
	}
	return nautMilesToMeters(f)
}

func nautMilesToMeters(nm float64) float64 {
	return nm * 1852
}

func metersToDegreesOfLat(m float64) float64 {
	return m / 1852 / degToNautMileY
}
func degreesOfLatToMeters(d float64) float64 {
	return d * 1852 * degToNautMileY
}
func degreesOfLngToMeters(d float64) float64 {
	return d * 1852 * degToNautMileX
}
func metersToDegreesOfLng(m float64) float64 {
	return m / 1852 / degToNautMileX
}

func Load(url string) ([]Feature, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return Decode(b)
}

func LoadFile(fileName string) ([]Feature, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return Decode(b)
}

func EnclosingVolumes(point orb.Point, features map[string]Feature) []Volume {
	enclosingVolumes := make([]Volume, 0)
	for _, f := range features {
		for _, v := range f.Geometry {
			if isEnclosedBy(point, v) {
				enclosingVolumes = append(enclosingVolumes, v)
			}
		}
	}

	return enclosingVolumes
}

func isEnclosedBy(p orb.Point, vol Volume) bool {
	if vol.Circle.Radius != 0 {
		projectedCentre := project.Point(vol.Circle.Centre, project.WGS84.ToMercator)
		projectedPoint := project.Point(p, project.WGS84.ToMercator)
		if planar.Distance(projectedPoint, projectedCentre) <= vol.Circle.Radius {
			return true
		}
	}
	if len(vol.Polygon) > 0 {
		if planar.RingContains(vol.Polygon, p) {
			return true
		}
	}

	return false
}

// https://developers.google.com/maps/documentation/javascript/overlays
// https://www.w3.org/Graphics/SVG/IG/resources/svgprimer.html#scale
// https://www.doc-developpement-durable.org/file/Projets-informatiques/cours-&-manuels-informatiques/htm-html-xml-ccs/Building%20Web%20Applications%20with%20SVG.pdf
// See https://eloquentjavascript.net/17_canvas.html
// http://jsfiddle.net/w1t1j2a1/
// https://en.wikipedia.org/wiki/Quadtree
