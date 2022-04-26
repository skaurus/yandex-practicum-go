package handlers

import (
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/skaurus/yandex-practicum-go/internal/storage"
)

func CreateHandler(store storage.Storage) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" && r.Method == http.MethodPost:
			url, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			if len(url) == 0 {
				http.Error(w, "empty url", 400)
				return
			}
			newId := store.Shorten(string(url))
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("http://localhost:8080/" + strconv.Itoa(newId)))
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
			url, ok := store.Unshorten(id)
			if !ok {
				http.Error(w, "wrong id", 400)
				return
			}
			w.Header().Set("Location", url)
			w.WriteHeader(http.StatusTemporaryRedirect)
			w.Write([]byte(""))
		default:
			http.Error(w, "no handler defined", 400)
		}
	}
}
