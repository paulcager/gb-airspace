package airspace

import (
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"text/template"

	"github.com/kr/pretty"
)

const (
	minLat               = 49.5
	maxLat               = 59
	minLon               = -6.5
	maxLon               = 2
	peakLat              = 53.35  // The latitude of the Peak District, used to centre the projection's distortion.
	maxInterestingHeight = 10_000 // We are not interested in restrictions > 10,000 feet
)

var (
	// One minute of latitude is always one nautical mile.
	degToNautMileY = 60.0
	// One minute of longitude depends on your latitude. This will be different at the N and S of the map,
	// so use the Peak District as the logical centre of the map.
	//
	// This does not affect where Google Maps will place the objects we draw, but *does* affect how circles
	// etc appear once Maps has stretched our image to cover the map's canvas.
	degToNautMileX = 60.0 * math.Cos(math.Pi/180.0*peakLat)

	heightNautMiles = (maxLat - minLat) * degToNautMileY
	widthNautMiles  = (maxLon - minLon) * degToNautMileX
)

func ToSVG(features []Feature, w io.Writer) error {
	// The image has origin (0,0) in NW corner. All dimensions are in nautical miles.

	params := map[string]interface{}{
		"minLat":   minLat,
		"maxLat":   maxLat,
		"minLon":   minLon,
		"maxLon":   maxLon,
		"height":   heightNautMiles,
		"width":    widthNautMiles,
		"features": features,
	}

	t := template.Must(template.New("airspace").Funcs(funcMap).Parse(tmplt))
	return t.Execute(w, params)
	//_=t
	//_=params
	//w.Write([]byte(test1))
	//return nil
}

var funcMap = template.FuncMap{
	// x converts a longitude to nautical miles from the origin
	"x": xPos,
	// y converts a longitude to nautical miles from the origin
	"y":             yPos,
	"d":             decodeDistance,
	"pretty":        func(obj interface{}) string { return pretty.Sprint(obj) },
	"height":        height,
	"colourise":     colourise,
	"isInteresting": func(h string) bool { return height(h) <= maxInterestingHeight },
	//"polygon":       polygon,
}

func xPos(x float64) float64 { return (x - minLon) * degToNautMileX }
func yPos(y float64) float64 { return (maxLat - y) * degToNautMileY }

func height(h string) float64 {
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

func chooseColour(featureType string, class string, h float64) (string, float64) {
	if !prohibitedAirspaceClasses[class] {
		return "black", 0.05
	}

	switch {
	case h == 0:
		return "red", 0.25
	case h < 1000:
		return "red", 0.1
	case h < 3000:
		return "green", 0.1
	case h < 5000:
		return "blue", 0.1
	default:
		return "#000000", 0.1
	}
}

// TODO - pass all of these in seems silly.
//   type: OTHER
//  localtype: MATZ
//  controltype: MILITARY.

func colourise(featureType string, class string, hStr string) string {
	colour, opacity := chooseColour(featureType, class, height(hStr))
	return fmt.Sprintf(`fill="%s" opacity="%f"`, colour, opacity)
}

//func polygon(featureType string, class string, h string, bounds []Boundary) string {
//	colour, opacity := chooseColour(featureType, class, height(h))
//	b := new(bytes.Buffer)
//	b.WriteString(`<path d="`)
//	segs := 0
//	for _, bound := range bounds {
//		if len(bound.Line) > 0 {
//			for _, p := range bound.Line {
//				if segs == 0 {
//					fmt.Fprintf(b, "M %f %f ", xPos(p.X()), yPos(p.Y()))
//				} else {
//					fmt.Fprintf(b, "L %f %f ", xPos(p.X()), yPos(p.Y()))
//				}
//				segs++
//			}
//		}
//		arc := bound.Arc
//		if arc.Radius != "" {
//			radius := distance(arc.Radius)
//			_ = radius
//			//fmt.Fprintf(b,`A %f %f 0 0 0 0 %f %f`, radius, radius, xPos(arc.To.X()), yPos(arc.To.Y()))
//			fmt.Fprintf(b,`L %f %f`, xPos(arc.To.X()), yPos(arc.To.Y()))
//		}
//	}
//	fmt.Fprintf(b, `Z" fill="none" stroke="%s" stroke-opacity="%f" stroke-width="0.25" />`, colour, opacity)
//	return b.String()
//}

const tmplt = `
{{ $ := . }}
<svg viewBox="0 0 {{.width}} {{.height}}" preserveAspectRatio="none" xmlns="http://www.w3.org/2000/svg">
{{range .features -}}
	{{- $feature := . -}}
	{{range .Geometry -}}
		{{ $volume := . -}}
		{{- if isInteresting .Lower -}}
			<!-- {{.ID}} {{.Name}} {{.Class}} {{.Lower}} -->
			{{range .Boundary -}}
				{{if ne "" .Circle.Centre.Text -}}
					<circle cx="{{x .Circle.Centre.X}}" cy="{{y .Circle.Centre.Y}}" r="{{d .Circle.Radius}}" {{colourise $feature.Type $volume.Class $volume.Lower}}/>
				{{- end -}}
			{{- end }}
			{{- polygon feature.Type .Class .Lower .Boundary -}}
		{{- end -}}
	{{end -}}
{{end}}
<!-- <rect x="0" y="0" height="{{.height}}" width="{{.width}}" fill="none" stroke="#000000" stroke-width="5"/> -->
</svg>
`
const test1 = `

<svg viewBox="0 0 304.4318715140466 570" preserveAspectRatio="none" xmlns="http://www.w3.org/2000/svg">
<!-- aberdeen-cta ABERDEEN CTA D 1500 ft -->
                        <path d="
                            M 161.881533 97.776923
                            L 162.336114 99.000000
                            L 140.699870 99.000000
							A 0.1 0.1 0 1 161.881533 97.776923
                            "
                            fill="none" stroke="green" stroke-opacity="0.100000" stroke-width="0.25" /><!-- aberdeen-cta ABERDEEN CTA D 1500 ft -->
						<circle cx="161.881533" cy="97.776923" r="1" fill="yellow" opacity="0.5"/>
						<circle cx="162.336114" cy="99" r="1" fill="green" opacity="0.5"/>
						<circle cx="140.699870" cy="99" r="1" fill="red" opacity="0.5"/>

</svg>
`

// A 10.000000 10.000000 0 1 161.881533 97.776923
//                            L 161.881533 97.776923
