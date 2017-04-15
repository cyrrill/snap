package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/patrickmn/go-cache"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

// Type used to store network responses in local cache
type Payload struct {
	Body        string
	StatusCode  int
	ContentType string
}

// Contains local cache
var c *cache.Cache

// Base URI of remote source
var base string

// Local port to serve from
var port = "80"

// Main function, read arguments and serve
func main() {

	getArgs()

	log.Print("Initializing Cache")
	c = cache.New(6*time.Hour, 1*time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("/", requestHandler)

	//  Start HTTP
	log.Print("Handling HTTP")
	go startHTTP(mux)

	//  Start HTTPS
	log.Print("Handling HTTPS")
	startHTTPS(mux)
}

// Reads base URI and port
func getArgs() {
	base = os.Args[1]
	if len(base) == 0 {
		panic("No source")
	}
	if len(os.Args) >= 3 {
		port = string(os.Args[2])
	}
}

// Listens via HTTP on specified parameter port
func startHTTP(mux *http.ServeMux) {
	err_http := http.ListenAndServe(":"+port, handlers.CompressHandler(mux))
	if err_http != nil {
		log.Fatal("Web server (HTTP): ", err_http)
	}
}

// Listens via TLS on fixed port 443
func startHTTPS(mux *http.ServeMux) {
	err_https := http.ListenAndServeTLS(
		":443",
		"ssl/public.crt",
		"ssl/private.key",
		mux,
	)
	if err_https != nil {
		log.Fatal("Web server (HTTPS): ", err_https)
	}
}

// HTTP handler checks local cache, else uses source
func requestHandler(w http.ResponseWriter, r *http.Request) {
	var (
		found   bool
		payload Payload
	)

	start := time.Now()

	// Success flag initialized to false
	hit := "MISS"

	// Get Request URI and output to STDOUT
	uri := r.URL.RequestURI()

	// Only cache GET requests
	if strings.ToUpper(r.Method) == "GET" {

		// Calculate a hash value to use as a shorter key for the URIs
		hash := getMD5Hash(uri)

		// Attempt to retrieve hash key from cache
		stored, found := c.Get(hash)

		if found {
			hit = "HIT"
			// If the the value is in cache, assign to return value
			payload = stored.(Payload)
		} else {
			// Else get it from source
			payload, found = getSource(uri)
			if found {
				c.Set(hash, payload, cache.NoExpiration)
			}
		}

	} else {
		payload, found = getSource(uri)
	}

	if found {
		// Set HTTP headers
		w.Header().Add("Content-Type", payload.ContentType)
		w.Header().Add("Pragma", "public")
		w.Header().Add("Cache-Control", "max-age=84097, public")
		w.Header().Add("X-Snap", hit)

		// Write payload to HTTP response body
		io.WriteString(w, payload.Body)
	} else {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	elapsed := time.Since(start)

	log.Print(uri + " (" + elapsed.String() + ")")
}

// Gets remote content, minifies, and stores in cache
func getSource(uri string) (Payload, bool) {
	resp, err := http.Get(base + uri)
	if err == nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		payload := Payload{
			minifyBody(body),
			resp.StatusCode,
			resp.Header.Get("Content-Type"),
		}
		return payload, err == nil
	} else {
		panic(err)
	}
	return Payload{}, false
}

// Minfies the body of a request
func minifyBody(body []byte) string {
	var b bytes.Buffer
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	mw := m.Writer("text/html", &b)
	if _, err := mw.Write(body); err != nil {
		panic(err)
	}
	if err := mw.Close(); err != nil {
		panic(err)
	}
	return string(b.Bytes())
}

// MD5 helper method
func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
