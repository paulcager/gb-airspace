# gb-airspace

[![Build and Push Docker Image](https://github.com/paulcager/gb-airspace/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/paulcager/gb-airspace/actions/workflows/docker-publish.yml)

A Go library and REST server providing UK airspace information for paraglider and hang glider pilots.

This project parses and serves airspace data from the [ahsparrow/airspace](https://github.com/ahsparrow/airspace)
project, making it easy to query airspace restrictions programmatically or via HTTP API.

## ⚠️ IMPORTANT DISCLAIMER ⚠️

**THIS SOFTWARE IS FOR INFORMATIONAL AND PLANNING PURPOSES ONLY.**

- This software provides NO WARRANTY of accuracy, completeness, or fitness for any purpose
- Airspace data may be outdated, incomplete, or incorrect
- You are SOLELY responsible for ensuring compliance with all applicable airspace regulations
- ALWAYS verify airspace information with official sources (NOTAMs, AIS, charts) before flight
- ALWAYS maintain situational awareness and comply with all aviation regulations
- The authors and contributors accept ABSOLUTELY NO LIABILITY for airspace violations, accidents, incidents, or any
  other consequences arising from use of this software
- **If you fly into restricted airspace, that's on you, not us**

By using this software, you acknowledge that flight safety is YOUR responsibility.

## Features

- **Go Library**: Import and use airspace data directly in your Go applications
- **REST API Server**: Query airspace via HTTP endpoints
- **Point Queries**: Find all airspace volumes containing a specific lat/lon coordinate
- **Feature Lookup**: Retrieve specific airspace features by ID
- **SVG Generation**: Generate visual representations of UK airspace
- **Airspace Classification**: Automatic classification of prohibited vs danger areas
- **Geometric Operations**: Handle circles, polygons, and arc boundaries

## Installation

### As a Library

```bash
go get github.com/paulcager/gb-airspace
```

### As a Server

```bash
# Install the server binary
go install github.com/paulcager/gb-airspace/cmd/serve-airspace@latest

# Or build from source
git clone https://github.com/paulcager/gb-airspace
cd gb-airspace
go build ./cmd/serve-airspace
```

### Using Docker

#### Pre-built Images

Pull and run the latest pre-built image from GitHub Container Registry:

```bash
docker pull ghcr.io/paulcager/gb-airspace:latest
docker run -p 9092:9092 ghcr.io/paulcager/gb-airspace:latest
```

Available tags:
- `latest` - Latest build from master branch
- `v1.2.3` - Specific version tags
- `master` - Latest master branch build

#### Building Locally

```bash
docker build -t gb-airspace .
docker run -p 9092:9092 gb-airspace
```

## Usage

### As a Go Library

```go
package main

import (
	"fmt"
	"github.com/paulcager/gb-airspace"
	"github.com/paulmach/orb"
)

func main() {
	// Load airspace data from URL
	features, err := airspace.Load("https://raw.githubusercontent.com/ahsparrow/airspace/master/airspace.yaml")
	if err != nil {
		panic(err)
	}

	// Or load from a local file
	features, err = airspace.LoadFile("airspace.yaml")
	if err != nil {
		panic(err)
	}

	// Convert to map for lookups
	featureMap := make(map[string]airspace.Feature)
	for _, f := range features {
		featureMap[f.ID] = f
	}

	// Find airspace at a specific point (lat, lon)
	point := orb.Point{-0.1, 51.5} // London: lon, lat
	volumes := airspace.EnclosingVolumes(point, featureMap)

	for _, v := range volumes {
		fmt.Printf("Inside %s (%s) - %s Class %s, %.0f-%.0f ft\n",
			v.Name, v.ID, v.Type, v.Class, v.Lower, v.Upper)
		if v.ClearanceRequired {
			fmt.Println("  ⚠️  ATC clearance required")
		}
		if v.Danger {
			fmt.Println("  ⚠️  Danger area - avoid or use caution")
		}
	}
}
```

### As a REST Server

Start the server:

```bash
# Default: port 9092, latest airspace data from GitHub
./serve-airspace

# Custom port
./serve-airspace --port 8080

# Custom airspace data URL
./serve-airspace --airspace-url https://example.com/custom-airspace.yaml

# Custom local file
./serve-airspace --airspace-url file:///path/to/airspace.yaml
```

## REST API

Base URL: `http://localhost:9092/v4/airspace/`

### Get All Airspace Features

```bash
GET /v4/airspace/all
```

Returns all airspace features as a JSON object keyed by feature ID.

**Example:**

```bash
curl http://localhost:9092/v4/airspace/all
```

### Query by Lat/Lon

```bash
GET /v4/airspace/?latlon=LAT,LON
```

Returns an array of airspace volumes that contain the specified point.

**Example:**

```bash
# Check airspace over London
curl "http://localhost:9092/v4/airspace/?latlon=51.5,-0.1"
```

**Response:**

```json
[
  {
    "ID": "london-ctr",
    "Name": "LONDON CTR",
    "Type": "CTR",
    "Class": "D",
    "Sequence": 1,
    "Lower": 0,
    "Upper": 2500,
    "ClearanceRequired": true,
    "Danger": false,
    "Circle": {
      "Radius": 27780,
      "Centre": [
        -0.461389,
        51.4775
      ]
    },
    "Polygon": null
  }
]
```

### Get Specific Feature by ID

```bash
GET /v4/airspace/?name=FEATURE_ID
```

Returns a single airspace feature.

**Example:**

```bash
curl "http://localhost:9092/v4/airspace/?name=london-ctr"
```

## Airspace Data

### Data Source

Airspace data is sourced from the [ahsparrow/airspace](https://github.com/ahsparrow/airspace) project, which provides
regularly updated UK airspace definitions in YAML format following
the [yaixm schema](https://github.com/ahsparrow/yaixm/blob/master/yaixm/data/schema.yaml).

### Updating Airspace Data

The server fetches airspace data on startup. To get the latest data:

#### Option 1: Restart the Server

Simply restart the `serve-airspace` process. It will fetch fresh data from GitHub on startup.

```bash
# Kill and restart the server
killall serve-airspace
./serve-airspace
```

#### Option 2: Use a Local File

Download the latest data manually and point the server to it:

```bash
# Download latest data
wget https://raw.githubusercontent.com/ahsparrow/airspace/master/airspace.yaml

# Start server with local file
./serve-airspace --airspace-url file://$(pwd)/airspace.yaml
```

#### Option 3: Schedule Automatic Updates

Use a process manager like systemd or supervisor to restart the service periodically, or use a cron job:

```bash
# Example cron job to restart daily at 2 AM
0 2 * * * systemctl restart gb-airspace
```

### Checking Last Update Date

The airspace data itself doesn't include a version or date field. To determine when the data was last updated:

#### Check GitHub Commit History

Visit the [airspace repository](https://github.com/ahsparrow/airspace) and check the commit history for `airspace.yaml`:

```
https://github.com/ahsparrow/airspace/commits/master/airspace.yaml
```

#### Check HTTP Headers

When the server fetches data, you can check the `Last-Modified` header:

```bash
curl -I https://raw.githubusercontent.com/ahsparrow/airspace/master/airspace.yaml
```

#### Check Server Startup Logs

The server logs when it successfully loads data. Check your logs for the startup timestamp.

## Data Format

### Airspace Classes

- **A, B, C, D, E**: Controlled airspace (clearance required for VFR flight)
- **F**: Not used in UK
- **G**: Open FIR (no clearance required)

### Airspace Types

**Prohibited (clearance required):**

- CTR (Control Region)
- CTA (Control Area)
- TMA (Terminal Control Area)
- ATZ (Aerodrome Traffic Zone)
- MATZ (Military ATZ)
- P (Prohibited)
- R (Restricted)
- RAT (Temporary Restricted Area)
- And others...

**Danger Areas (advisory):**

- D (Danger Area)
- DZ (Drop Zone)
- GLIDER (Gliding Operations)
- AIAA (Area of Intense Aerial Activity)
- And others...

### Altitudes

- **SFC**: Surface (0 feet)
- **FL115**: Flight level (115 × 100 = 11,500 feet)
- **3000 ft**: Feet above mean sea level

## Development

### Building

```bash
# Build the library and server
go build ./...

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Running Tests

```bash
# All tests
go test ./...

# Verbose output
go test -v ./...

# Specific test
go test -run TestDecode

# Include live data download test
go test -v -run TestDownload
```

### Docker Build

```bash
# Build image
docker build -t gb-airspace .

# Run container
docker run -p 9092:9092 gb-airspace

# Run with custom port
docker run -p 8080:9092 -e PORT=9092 gb-airspace
```

## Dependencies

- [paulmach/orb](https://github.com/paulmach/orb) - Geospatial types and operations
- [gopkg.in/yaml.v2](https://gopkg.in/yaml.v2) - YAML parsing
- [spf13/pflag](https://github.com/spf13/pflag) - Command-line flags
- [stretchr/testify](https://github.com/stretchr/testify) - Testing utilities
- [paulcager/go-http-middleware](https://github.com/paulcager/go-http-middleware) - HTTP middleware

## License

See [LICENSE](LICENSE) file for details.

## Credits

Airspace data provided by the [ahsparrow/airspace](https://github.com/ahsparrow/airspace) project, derived from official
UK CAA airspace definitions.

