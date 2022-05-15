package main

import (
	"github.com/skaurus/yandex-practicum-go/internal/app"
	"github.com/skaurus/yandex-practicum-go/internal/config"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
)

func main() {
	storage := storage.New(storage.Memory)
	config := config.ParseConfig()

	router := app.SetupRouter(&storage, config)
	router.Run(config.ServerAddr)
}
