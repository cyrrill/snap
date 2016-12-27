package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type Payload struct {
	Body          string
	StatusCode    int
	ContentType   string
	ContentLength string
}

var storage = make(map[string]Payload)

var base string

func main() {
	base = os.Args[1]
	mux := http.NewServeMux()
	mux.HandleFunc("/", requestHandler)
	http.ListenAndServe(":8000", mux)
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.RequestURI()
	hash := getMD5Hash(uri)
	payload, ok := storage[hash]
	if !ok {
		payload, ok = getSource(uri)
		storage[hash] = payload
	}
	w.Header().Add("Content-Type", payload.ContentType)
	w.Header().Add("Content-Length", payload.ContentLength)
	w.Header().Add("Pragma", "public")
	w.Header().Add("Cache-Control", "max-age=84097, public")

	io.WriteString(w, payload.Body)
}

func getSource(uri string) (Payload, bool) {
	resp, err := http.Get(base + uri)
	if err == nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		payload := Payload{
			string(body),
			resp.StatusCode,
			resp.Header.Get("Content-Type"),
			resp.Header.Get("Content-Length"),
		}
		return payload, err == nil
	}
	return Payload{}, false
}

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
