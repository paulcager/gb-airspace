package main

import (
	"encoding/json"
	"fmt"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/project"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	airspace "github.com/paulcager/gb-airspace"
	"github.com/paulcager/go-http-middleware"
	"github.com/paulmach/orb/planar"
	flag "github.com/spf13/pflag"
)

const (
	apiVersion = "v4"
)

var (
	port     string
	dataURL  string
	features []airspace.Feature
	_        = planar.RingContains
)

func main() {
	flag.StringVarP(&port, "port", "p", ":9092", "Port to listen on")
	flag.StringVarP(&dataURL, "airspace-url", "u", "https://gitlab.com/ahsparrow/airspace/-/raw/master/airspace.yaml", "airspace.yaml URL")
	flag.Parse()

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	var err error
	features, err = airspace.Load(dataURL)
	if err != nil {
		panic(err)
	}

	server := makeHTTPServer(port)
	log.Fatal(server.ListenAndServe())
}

func makeHTTPServer(listenPort string) *http.Server {
	middleware.EnablePrometheus()

	http.Handle(
		"/"+apiVersion+"/airspace/",
		middleware.MakeLoggingHandler(http.HandlerFunc(handle)))

	log.Println("Starting HTTP server on " + listenPort)

	s := &http.Server{
		ReadHeaderTimeout: 20 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       10 * time.Minute,
		Handler:           http.DefaultServeMux,
		Addr:              listenPort,
	}

	return s
}

func handle(w http.ResponseWriter, r *http.Request) {
	latLonStr := r.URL.Path[len("/"+apiVersion+"/airspace/"):]
	parts := strings.Split(latLonStr, ",")
	if len(parts) != 2 {
		handleError(w, r, latLonStr, nil)
		return
	}

	lat, err1 := strconv.ParseFloat(parts[0], 64)
	lon, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil {
		handleError(w, r, latLonStr, nil)
		return
	}

	point := orb.Point{lat, lon}

	enclosingVolumes := make([]airspace.Volume, 0)
	for _, f := range features {
		for _, v := range f.Geometry {
			if isEnclosedBy(point, v) {
				enclosingVolumes = append(enclosingVolumes, v)
			}
		}
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(enclosingVolumes); err != nil {
		log.Printf("Failed to write response: %s", err)
		w.WriteHeader(http.StatusBadGateway)
	}
}

func isEnclosedBy(p orb.Point, vol airspace.Volume) bool {
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

func handleError(w http.ResponseWriter, _ *http.Request, str string, err error) {
	var s string
	if err != nil {
		s = fmt.Sprintf("Invalid request: %q (%s)\n", str, err)
	} else {
		s = fmt.Sprintf("Invalid request: %q\n", str)
	}
	http.Error(w, s, http.StatusBadRequest)
}
