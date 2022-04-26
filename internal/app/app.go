package app

import (
	"log"
	"net/http"

	"github.com/skaurus/yandex-practicum-go/internal/handlers"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
)

func Serve() {
	storage := storage.New(storage.Memory)
	http.HandleFunc("/", handlers.CreateHandler(storage))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
