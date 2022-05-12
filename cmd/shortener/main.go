package main

import (
	"github.com/skaurus/yandex-practicum-go/internal/app"
)

func main() {
	router := app.SetupRouter()
	router.Run()
}
