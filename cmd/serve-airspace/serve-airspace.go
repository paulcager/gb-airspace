package main

import (
	"encoding/json"
	"fmt"
	"github.com/paulmach/orb"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	airspace "github.com/paulcager/gb-airspace"
	"github.com/paulcager/go-http-middleware"
	flag "github.com/spf13/pflag"
)

const (
	apiVersion = "v4"
)

var (
	port     string
	dataURL  string
	features []airspace.Feature
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

	point := orb.Point{lon, lat}
	enclosingVolumes := airspace.EnclosingVolumes(point, features)

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(enclosingVolumes); err != nil {
		log.Printf("Failed to write response: %s", err)
		w.WriteHeader(http.StatusBadGateway)
	}
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
