package main

import (
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	shorts := make(map[int][]byte)
	counter := 0

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" && r.Method == http.MethodPost:
			url, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			counter++
			shorts[counter] = url
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(strconv.Itoa(counter)))
		case r.Method == http.MethodGet:
			match, err := regexp.MatchString(`^/[0-9]+$`, r.URL.Path)
			if err != nil || !match {
				http.Error(w, "wrong url", 400)
				return
			}
			id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/"))
			if err != nil {
				http.Error(w, "can't parse id", 400)
				return
			}
			url, ok := shorts[id]
			if !ok {
				http.Error(w, "wrong id", 400)
				return
			}
			w.Header().Set("Location", string(url))
			w.WriteHeader(http.StatusTemporaryRedirect)
			w.Write([]byte(""))
		default:
			http.Error(w, "no handler defined", 400)
		}
	})
	http.ListenAndServe(":8080", nil)
}
