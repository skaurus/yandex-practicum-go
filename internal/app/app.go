package app

import (
	"compress/gzip"
	"compress/zlib"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/skaurus/yandex-practicum-go/internal/config"
	"github.com/skaurus/yandex-practicum-go/internal/handlers"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
	"github.com/skaurus/yandex-practicum-go/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func SetGlobalVars(config *config.Config, storage *storage.Storage, logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("config", config)
		c.Set("storage", storage)
		c.Set("logger", logger)

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
	return w.ResponseWriter.Header()
}

// избегаем попадания заголовков в gzWriter
func (w gzWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

var gzipReader *gzip.Reader
var gzipWriter *gzip.Writer

func GzipCompression(c *gin.Context) {
	log := c.MustGet("logger").(*zerolog.Logger)

	// разжимаем запрос
	ce := c.GetHeader("Content-Encoding")
	switch {
	case ce == "gzip":
		var err error
		if gzipReader == nil {
			gzipReader, err = gzip.NewReader(c.Request.Body)
		} else {
			err = gzipReader.Reset(c.Request.Body)
		}
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		defer gzipReader.Close()
		c.Request.Body = gzipReader
	case ce == "deflate":
		// в документации написано, что io.ReadCloser, который возвращает zlib.NewReader,
		// имплементирует интерфейс Resetter с методом Reset - но кажется это не так :(
		// а без Reset смысла в кешировании глобальной переменной нет
		//err := zlibReader.Reset(c.Request.Body)
		zlibReader, err := zlib.NewReader(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
		}
		defer zlibReader.Close()
		c.Request.Body = zlibReader
	case len(ce) > 0:
		c.String(http.StatusBadRequest, "unsupported Content-Encoding")
		return
	}

	if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
		c.Next()
		return
	}

	// жмём ответ
	switch c.GetHeader("Content-Type") {
	case "application/json", "application/javascript", "text/plain", "text/html", "text/css", "text/xml":
		if gzipWriter == nil {
			var err error
			gzipWriter, err = gzip.NewWriterLevel(c.Writer, gzip.BestCompression)
			if err != nil {
				log.Fatal().Err(err)
				break
			}
		} else {
			gzipWriter.Reset(c.Writer)
		}
		defer gzipWriter.Close()

		c.Writer = gzWriter{c.Writer, gzipWriter}
		c.Header("Content-Encoding", "gzip")
	}

	c.Next()
}

const (
	uniqCookieName   = "uniq"
	uniqCookieMaxAge = 60 * 60 * 24 * 365     // seconds
	cookieSecretKey  = "carrot-james-regular" // https://edoceo.com/dev/mnemonic-password-generator
)

// SetCookies - проставляем/читаем куки
func SetCookies(c *gin.Context) {
	config := c.MustGet("config").(*config.Config)
	log := c.MustGet("logger").(*zerolog.Logger)

	var uniq string
	// блок с несколькими последовательными проверками - это способ не делать
	// вложенные один в другой if (success) { ... }
	for {
		// 1. пытаемся прочитать куку уника
		cookieValue, err := c.Cookie(uniqCookieName)
		if err != nil { // куки не было
			log.Info().Msg("no uniq cookie")
			break
		}

		// 2. пытаемся достать из куки айди и подпись
		maybeUniq, sign, found := strings.Cut(cookieValue, "-")
		if !found {
			log.Error().Msg("uniq cookie don't have separator")
			break
		}

		// 3. пытаемся расшифровать подпись куки уника
		sign1, err := hex.DecodeString(sign)
		if err != nil {
			log.Error().Msg("uniq cookie signature can't be decoded")
			break
		}

		hmacer := hmac.New(sha256.New, []byte(cookieSecretKey))
		hmacer.Write([]byte(maybeUniq))
		sign2 := hmacer.Sum(nil)
		if !hmac.Equal(sign1, sign2) {
			log.Error().Msg("uniq cookie signature is wrong")
			break
		}

		uniq = maybeUniq
		break
	}

	if len(uniq) == 0 {
		uniq = utils.RandStringN(8)
		hmacer := hmac.New(sha256.New, []byte(cookieSecretKey))
		hmacer.Write([]byte(uniq))
		sign := hmacer.Sum(nil)
		cookieValue := fmt.Sprintf("%s-%s", uniq, hex.EncodeToString(sign))
		c.SetCookie(uniqCookieName, cookieValue, uniqCookieMaxAge, "/", config.CookieDomain, false, true)
		log.Info().Msg("set uniq cookie " + cookieValue)
	}

	c.Set("uniq", uniq)

	c.Next()
}

func SetupRouter(config *config.Config, storage *storage.Storage) *gin.Engine {
	gin.DisableConsoleColor()
	f, _ := os.OpenFile(config.LogName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	gin.DefaultWriter = io.MultiWriter(f)

	zerolog.SetGlobalLevel(config.LogLevel)
	logger := zerolog.New(f).With().Timestamp().Logger()

	router := gin.Default()
	router.Use(SetGlobalVars(config, storage, &logger))
	router.Use(GzipCompression)
	router.Use(SetCookies)

	router.POST("/", handlers.BodyShorten)
	router.GET("/:id", handlers.Redirect)
	router.POST("/api/shorten", handlers.APIShorten)
	router.GET("/api/user/urls", handlers.GetAllUserURLs)

	return router
}
