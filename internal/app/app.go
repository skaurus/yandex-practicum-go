package app

import (
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"strings"

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

type gzWriter struct {
	gin.ResponseWriter
	Writer io.Writer
}

func (w gzWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w gzWriter) WriteString(s string) (int, error) {
	return w.Writer.Write([]byte(s))
}

// избегаем попадания заголовков в gzWriter
func (w gzWriter) Header() http.Header {
	w.ResponseWriter.Header()
}

// избегаем попадания заголовков в gzWriter
func (w gzWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func GzipCompression(c *gin.Context) {
	if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
		c.Next()
		return
	}
	switch c.GetHeader("Content-Type") {
	case "application/json", "application/javascript", "text/plain", "text/html", "text/css", "text/xml":
		gz, err := gzip.NewWriterLevel(c.Writer, gzip.BestCompression)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		defer gz.Close()

		c.Writer = gzWriter{c.Writer, gz}
		c.Header("Content-Encoding", "gzip")
		c.Next()
		return
	default:
		c.Next()
		return
	}
}

func SetupRouter(storage *storage.Storage, config *config.Config) *gin.Engine {
	gin.DisableConsoleColor()
	f, _ := os.Create(config.LogName)
	gin.DefaultWriter = io.MultiWriter(f)

	router := gin.Default()
	router.Use(AddStorage(storage))
	router.Use(AddConfig(config))
	router.Use(GzipCompression)

	router.POST("/", handlers.BodyShorten)
	router.GET("/:id", handlers.Get)
	router.POST("/api/shorten", handlers.APIShorten)

	return router
}
