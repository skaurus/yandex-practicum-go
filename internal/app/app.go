package app

import (
	"io"
	"os"

	"github.com/skaurus/yandex-practicum-go/internal/handlers"
	"github.com/skaurus/yandex-practicum-go/internal/storage"

	"github.com/gin-gonic/gin"
)

func AddStore() gin.HandlerFunc {
	store := storage.New(storage.Memory)
	return func(c *gin.Context) {
		c.Set("storage", store)

		c.Next()
	}
}

func SetupRouter() *gin.Engine {
	gin.DisableConsoleColor()
	f, _ := os.Create("app.log")
	gin.DefaultWriter = io.MultiWriter(f)

	router := gin.Default()
	router.Use(AddStore())

	router.POST("/", handlers.Post)
	router.GET("/:id", handlers.Get)

	return router
}
