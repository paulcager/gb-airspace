package airspace

import (
	"github.com/paulcager/osgridref"
	"github.com/paulmach/orb"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)
import "github.com/stretchr/testify/assert"

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

func TestDownload(t *testing.T) {
	// Verify real-life data exists and can be parsed correctly.
	url := `https://gitlab.com/ahsparrow/airspace/-/raw/master/airspace.yaml`
	a, err := Load(url)
	require.NoError(t, err)

	assert.Greater(t, len(a), 600)
}

func Test_arcToPolygon(t *testing.T) {
	type args struct {
		centre       orb.Point
		radius       float64
		initialPoint orb.Point
		to           orb.Point
		dir          float64
	}
	tests := []struct {
		name string
		args args
		want []osgridref.LatLon
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := arcToPolygon(tt.args.centre, tt.args.radius, tt.args.initialPoint, tt.args.to, tt.args.dir); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("arcToPolygon() = %v, want %v", got, tt.want)
			}
		})
	}
}
