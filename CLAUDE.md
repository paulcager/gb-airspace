# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go library and REST server for UK airspace information, specifically designed for paraglider and hang glider
pilots. It processes airspace definitions from YAML format and provides both a library API and HTTP REST endpoints to
query airspace data.

The project consumes airspace data from https://github.com/ahsparrow/airspace (YAML format following the yaixm schema).

## Development Commands

### Building

```bash
# Build all commands
go build ./...

# Build the server binary
go build ./cmd/serve-airspace

# Build static binary (as used in Docker)
CGO_ENABLED=0 go install ./...
```

### Testing

```bash
# Run all tests
go test ./...

# Run specific test
go test -run TestDecode

# Run tests with verbose output
go test -v ./...
```

### Running the Server

```bash
# Run with default settings (port 9092)
go run ./cmd/serve-airspace/serve-airspace.go

# Run with custom port
go run ./cmd/serve-airspace/serve-airspace.go --port 8080

# Run with custom airspace URL
go run ./cmd/serve-airspace/serve-airspace.go --airspace-url https://path/to/airspace.yaml
```

### Docker

```bash
# Build Docker image
docker build -t gb-airspace .

# Run container
docker run -p 9092:9092 gb-airspace
```

## Architecture

### Core Components

**airspace.go** - Main library containing:

- `Feature` and `Volume` types representing airspace features with 3D geometry (horizontal shape + vertical bounds)
- YAML parsing from the ahsparrow/airspace schema into normalized internal structures
- Geometric calculations for points, circles, polygons, and arcs
- `EnclosingVolumes()` - finds all airspace volumes that contain a given lat/lon point
- Airspace classification logic distinguishing prohibited vs danger areas

**svg.go** - SVG generation for visual representation of airspace:

- Projects UK airspace (lat/lon bounds: 49.5-59°N, 6.5°W-2°E) onto a flat SVG canvas
- Uses Mercator-style projection centered on the Peak District (53.35°N) to minimize distortion
- Color-codes airspace by type and altitude (red for ground-level restrictions, green/blue for higher)

**cmd/serve-airspace/serve-airspace.go** - HTTP REST server:

- Loads airspace data on startup and keeps it in memory as a map[string]Feature indexed by feature ID
- Exposes v4 API endpoints (see REST API section below)
- Uses go-http-middleware for logging and Prometheus metrics

### Key Data Structures

**Feature**: Top-level airspace entity with ID, name, type, class, and one or more Volume geometries

- Examples: CTR (Control Region), CTA (Control Area), ATZ (Aerodrome Traffic Zone), etc.

**Volume**: A 3D airspace volume with:

- Horizontal geometry: either a Circle (center + radius) or Polygon (list of lat/lon points)
- Vertical bounds: Lower and Upper in feet (or flight levels converted to feet)
- Classification: ClearanceRequired (prohibited) vs Danger (advisory)

### Geometry Handling

The library converts YAML boundary definitions into normalized geometric shapes:

- **Lines**: Direct lat/lon points → polygon vertices
- **Arcs**: Converted to polygon approximations (10-degree segments) between two points around a center
- **Circles**: Stored as center point + radius in meters

Arc processing (`arcToPolygon`) handles both clockwise and counterclockwise arcs, computing bearing angles and
generating intermediate points along the arc.

### Coordinate System Notes

- External API uses WGS84 lat/lon (Point{lon, lat} - note lon comes first in orb.Point)
- Internal distance calculations project to Mercator for planar geometry operations
- Heights: Flight levels (FLxxx) converted to feet × 100, "SFC" = 0 feet
- Distances: Input in nautical miles, converted to meters (1 nm = 1852 m)

### Airspace Classification

**Prohibited Airspace** (clearance required):

- Airspace classes: A, B, C, D, E (not F or G)
- Types: ATZ, AWY, CTA, CTR, MATZ, P (prohibited), R (restricted), RAT, RMZ, TMA, TRA, TMZ

**Danger Areas** (advisory):

- Types: AIAA, D, D_OTHER, DZ (drop zones), GLIDER, HIRTA, LASER, NOATZ, UL (ultralight strips)

## REST API

Base URL: `/v4/airspace/`

### Endpoints

**GET /v4/airspace/all**

- Returns all airspace features as JSON
- Response: `map[string]Feature` (keyed by feature ID)

**GET /v4/airspace/?latlon=LAT,LON**

- Query airspace at a specific point
- Example: `/v4/airspace/?latlon=51.5,-0.1`
- Response: Array of `Volume` objects that enclose the point

**GET /v4/airspace/?name=FEATURE_ID**

- Retrieve a specific airspace feature by ID
- Example: `/v4/airspace/?name=aberdeen-cta`
- Response: Single `Feature` object or 404

### CORS

The server sets `Access-Control-Allow-Origin: *` for named requests to allow cross-origin access.

## Dependencies

- **paulmach/orb**: Geospatial types and operations (Point, Ring, planar geometry, projections)
- **gopkg.in/yaml.v2**: YAML parsing for airspace data
- **stretchr/testify**: Test assertions
- **spf13/pflag**: Command-line flag parsing
- **paulcager/go-http-middleware**: HTTP logging and Prometheus integration

## Important Implementation Details

### Lat/Lon Parsing

The `parseLatLng` function expects coordinates in the specific format from the YAML data:

- Format: "DDMMSSN DDDMMSSX" (e.g., "502257N 0033739W")
- 16 characters total with space separator at position 7
- Degrees, minutes, seconds for both latitude and longitude
- N/S for latitude, E/W for longitude

### Height Encoding

All heights are normalized to feet:

- "SFC" (surface) → 0
- "FL115" (flight level) → 11,500 feet
- "3000 ft" → 3,000 feet

The code filters out very high airspace (>10,000 feet) for SVG rendering as it's not relevant to paragliding/hang
gliding.

### Feature ID Generation

Most features have explicit IDs from the YAML data. For features without IDs (like drop zones), the code generates one
by combining the lowercase name (spaces replaced with hyphens) and a numeric index.

### Memory Model

The server loads all airspace data into memory on startup and stores it in a map for O(1) lookup by ID. For the lat/lon
query endpoint, it iterates through all features and volumes to find enclosing geometries (no spatial index).
