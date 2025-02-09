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
	"strings"
	"time"

	"github.com/gorilla/mux"
	expand "github.com/openvenues/gopostal/expand"
	parser "github.com/openvenues/gopostal/parser"
)

type Request struct {
	Query string `json:"query"`
}

type SplitAddress struct {
	House          string
	Category       string
	Near           string
	House_number   string
	Road           string
	Unit           string
	Level          string
	Staircase      string
	Entrance       string
	Po_box         string
	Postcode       string
	Suburb         string
	City_district  string
	City           string
	Island         string
	State_district string
	State          string
	Country_region string
	Country        string
	World_region   string
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
	opt := expand.GetDefaultExpansionOptions()
	opt.Languages = []string{"en"}
	// https://github.com/openvenues/libpostal/issues/302#issuecomment-358268077
	// Use AddressToponym to ignore some of the possible expansions like "calle"
	//which mostly apply to streets. st was expanding to Saint
	opt.AddressComponents = expand.AddressToponym
	expansions := expand.ExpandAddressOptions(req.Query, opt)

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
	opt := parser.ParserOptions{Country: "au", Language: "en"}
	parsed := parser.ParseAddressOptions(req.Query, opt)
	for _, parsed_component := range parsed {
		f := reflect.Indirect(xr).FieldByName(strings.Title(parsed_component.Label))
		f.SetString(parsed_component.Value)
	}
	parseThing, _ := xml.Marshal(res)
	w.Write(parseThing)
}
