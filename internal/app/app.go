package app

import (
	"io"
	"os"

	"github.com/skaurus/yandex-practicum-go/internal/config"
	"github.com/skaurus/yandex-practicum-go/internal/handlers"
	"github.com/skaurus/yandex-practicum-go/internal/storage"

	"github.com/gin-gonic/gin"
)

func AddStorage(storage *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("storage", storage)

		c.Next()
	}
}

func AddConfig(config *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("config", config)

		c.Next()
	}
}

func SetupRouter(storage *storage.Storage, config *config.Config) *gin.Engine {
	gin.DisableConsoleColor()
	f, _ := os.Create(config.LogName)
	gin.DefaultWriter = io.MultiWriter(f)

	router := gin.Default()
	router.Use(AddStorage(storage))
	router.Use(AddConfig(config))

	router.POST("/", handlers.BodyShorten)
	router.GET("/:id", handlers.Get)
	router.POST("/api/shorten", handlers.APIShorten)

	return router
}
