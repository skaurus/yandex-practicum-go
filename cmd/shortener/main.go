package main

import (
	"math/rand"
	"time"

	"github.com/skaurus/yandex-practicum-go/internal/app"
	"github.com/skaurus/yandex-practicum-go/internal/config"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	config := config.ParseConfig()

	var store storage.Storage
	if len(config.StorageFileName) > 0 {
		storageConnectInfo := storage.ConnectInfo{
			Filename: config.StorageFileName,
		}
		store = storage.New(storage.File, storageConnectInfo)
	} else {
		store = storage.New(storage.Memory, storage.ConnectInfo{})
	}

	router := app.SetupRouter(config, &store)
	err := router.Run(config.ServerAddr)
	if err != nil {
		panic(err)
	}
}
