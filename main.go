package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

// Type used to store network responses in local cache
type Payload struct {
	Body          string
	StatusCode    int
	ContentType   string
	ContentLength string
}

// Contains local cache
var storage = make(map[string]Payload)

// Base URI of remote source
var base string

// Local port to serve from
var port = "80"

// Main function, read arguments and serve
func main() {
	getArgs()
	mux := http.NewServeMux()
	mux.HandleFunc("/", requestHandler)
	http.ListenAndServe(":"+port, handlers.CompressHandler(mux))
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

// HTTP handler checks local cache, else uses source
func requestHandler(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.RequestURI()
	io.WriteString(os.Stdout, uri+"\n")
	hash := getMD5Hash(uri)
	payload, ok := storage[hash]
	hit := "HIT"
	if !ok {
		hit = "MISS"
		payload, ok = getSource(uri)
		storage[hash] = payload
	}
	w.Header().Add("Content-Type", payload.ContentType)
	w.Header().Add("Content-Length", payload.ContentLength)
	w.Header().Add("Pragma", "public")
	w.Header().Add("Cache-Control", "max-age=84097, public")
	w.Header().Add("X-Snap", hit)
	io.WriteString(w, payload.Body)
}

// Gets remote content, minifies, and stores in cache
func getSource(uri string) (Payload, bool) {
	resp, err := http.Get(base + uri)
	if err == nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		minBody := minifyBody(body)
		payload := Payload{
			minBody,
			resp.StatusCode,
			resp.Header.Get("Content-Type"),
			string(len(minBody)),
		}
		return payload, err == nil
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
