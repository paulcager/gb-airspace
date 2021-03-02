package main

import (
	"encoding/json"
	"fmt"
	"github.com/paulmach/orb"
	"log"
	"net/http"
	"os"
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
	features map[string]airspace.Feature
)

func main() {
	flag.StringVarP(&port, "port", "p", ":9092", "Port to listen on")
	flag.StringVarP(&dataURL, "airspace-url", "u", "https://gitlab.com/ahsparrow/airspace/-/raw/master/airspace.yaml", "airspace.yaml URL")
	flag.Parse()

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	featureList, err := airspace.Load(dataURL)
	if err != nil {
		panic(err)
	}

	features = make(map[string]airspace.Feature, len(featureList))
	for _, f := range featureList {
		if _, ok := features[f.ID]; ok {
			log.Printf("Duplicate feature ID %q. Lookups will be undefined", f.ID)
		}
		features[f.ID]=f
	}

	out, _ := os.Create("/tmp/pc1.txt")
	for i, f := range features {
		for j, v := range f.Geometry {
			fmt.Fprintln(out, i, j, "<", v.ID, ">", v.Type, v.Sequence, v.Circle.Radius == 0, v.Lower, v.Upper)
		}
	}
	out.Close()

	server := makeHTTPServer(port)
	log.Fatal(server.ListenAndServe())
}

func makeHTTPServer(listenPort string) *http.Server {
	middleware.EnablePrometheus()

	http.Handle(
		"/"+apiVersion+"/airspace/all",
		middleware.MakeLoggingHandler(http.HandlerFunc(handleRequestAll)))

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
	values := r.URL.Query()
	latLon := strings.TrimSpace(values.Get("latlon"))
	name := strings.TrimSpace(values.Get("name"))

	if name != "" {
		handleNamedRequest(w, r, name)
		return
	}

	if latLon != "" {
		handleLatlonRequest(w, r, latLon)
		return
	}

	http.Error(w, "Invalid request", http.StatusBadRequest)
}

func handleRequestAll(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(features)
	if err != nil {
		log.Println("handleRequestAll:", err)
		http.Error(w, fmt.Sprintf("JSON encoding error: %s", err), http.StatusInternalServerError)
	}
}

func handleNamedRequest(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	f,ok := features[id]
	if !ok {
		log.Printf("Did not find feature %q\n", id)
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(f)
	if err != nil {
		log.Println("handleNamedRequest(" + id + "):", err)
		http.Error(w, fmt.Sprintf("JSON encoding error: %s", err), http.StatusInternalServerError)
	}
}

func handleLatlonRequest(w http.ResponseWriter, r *http.Request, latLonStr string) {
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
