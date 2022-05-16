package main

import (
	"github.com/skaurus/yandex-practicum-go/internal/app"
	"github.com/skaurus/yandex-practicum-go/internal/config"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
)

func main() {
	config := config.ParseConfig()

	var store storage.Storage
	if len(config.StorageFileName) > 0 {
		storageConnectInfo := storage.ConnectInfo{
			Filename: config.StorageFileName,
		}
		store = storage.New(storage.File, storageConnectInfo)
		defer store.Close()
	} else {
		store = storage.New(storage.Memory, storage.ConnectInfo{})
	}

	router := app.SetupRouter(&store, config)
	router.Run(config.ServerAddr)
}
