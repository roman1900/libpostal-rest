package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/gorilla/mux"
	expand "github.com/openvenues/gopostal/expand"
	parser "github.com/openvenues/gopostal/parser"
)

type Request struct {
	Query string `json:"query"`
}

type SplitAddress struct {
	house          string
	category       string
	near           string
	house_number   string
	road           string
	unit           string
	level          string
	staircase      string
	entrance       string
	po_box         string
	postcode       string
	suburb         string
	city_district  string
	city           string
	island         string
	state_district string
	state          string
	country_region string
	country        string
	world_region   string
}

func main() {
	host := os.Getenv("LISTEN_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("LISTEN_PORT")
	if port == "" {
		port = "8484"
	}
	listenSpec := fmt.Sprintf("%s:%s", host, port)

	certFile := os.Getenv("SSL_CERT_FILE")
	keyFile := os.Getenv("SSL_KEY_FILE")

	router := mux.NewRouter()
	router.HandleFunc("/health", HealthHandler).Methods("GET")
	router.HandleFunc("/expand", ExpandHandler).Methods("POST")
	router.HandleFunc("/parser", ParserHandler).Methods("POST")

	s := &http.Server{Addr: listenSpec, Handler: router}
	go func() {
		if certFile != "" && keyFile != "" {
			fmt.Printf("listening on https://%s\n", listenSpec)
			s.ListenAndServeTLS(certFile, keyFile)
		} else {
			fmt.Printf("listening on http://%s\n", listenSpec)
			s.ListenAndServe()
		}
	}()

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	<-stop
	fmt.Println("\nShutting down the server...")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	s.Shutdown(ctx)
	fmt.Println("Server stopped")
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func ExpandHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")

	var req Request

	q, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(q, &req)

	expansions := expand.ExpandAddress(req.Query)

	expansionThing, _ := xml.Marshal(expansions)
	w.Write(expansionThing)
}

func ParserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")

	var req Request
	var res SplitAddress
	q, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(q, &req)
	xr := reflect.ValueOf(&res)
	parsed := parser.ParseAddress(req.Query)
	for _, parsed_component := range parsed {
		f := reflect.Indirect(xr).FieldByName(parsed_component.Label)
		f.SetString(parsed_component.Value)
	}
	parseThing, _ := xml.Marshal(res)
	w.Write(parseThing)
}
