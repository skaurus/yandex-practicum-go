package main

import (
	"github.com/skaurus/yandex-practicum-go/internal/app"
	"github.com/skaurus/yandex-practicum-go/internal/config"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
)

func main() {
	storageConnectInfo := storage.ConnectInfo{
		Filename: "storage.json",
	}
	storage := storage.New(storage.File, storageConnectInfo)
	defer storage.Close()

	config := config.ParseConfig()

	router := app.SetupRouter(&storage, config)
	router.Run(config.ServerAddr)
}
