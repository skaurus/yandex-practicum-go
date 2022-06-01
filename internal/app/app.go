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
	"strings"

	"github.com/skaurus/yandex-practicum-go/internal/env"
	"github.com/skaurus/yandex-practicum-go/internal/handlers"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
	"github.com/skaurus/yandex-practicum-go/internal/utils"

	"github.com/gin-gonic/gin"
)

func SetGlobalVars(env *env.Environment, store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("env", env)
		c.Set("storage", store)

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
	env := c.MustGet("env").(*env.Environment)
	logger := env.Logger

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
				logger.Fatal().Err(err)
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
	env := c.MustGet("env").(*env.Environment)
	logger := env.Logger

	var uniq string
	// блок с несколькими последовательными проверками - это способ не делать
	// вложенные один в другой if (success) { ... }
	for {
		// 1. пытаемся прочитать куку уника
		cookieValue, err := c.Cookie(uniqCookieName)
		if err != nil { // куки не было
			logger.Info().Msg("no uniq cookie")
			break
		}

		// 2. пытаемся достать из куки айди и подпись
		// Cut появился только в go 1.18 ((
		//maybeUniq, sign, found := strings.Cut(cookieValue, "-")
		parts := strings.SplitN(cookieValue, "-", 2)
		maybeUniq, sign := parts[0], parts[1]
		if len(sign) == 0 {
			logger.Error().Msg("uniq cookie don't have separator")
			break
		}

		// 3. пытаемся расшифровать подпись куки уника
		sign1, err := hex.DecodeString(sign)
		if err != nil {
			logger.Error().Msg("uniq cookie signature can't be decoded")
			break
		}

		hmacer := hmac.New(sha256.New, []byte(cookieSecretKey))
		hmacer.Write([]byte(maybeUniq))
		sign2 := hmacer.Sum(nil)
		if !hmac.Equal(sign1, sign2) {
			logger.Error().Msg("uniq cookie signature is wrong")
			break
		}

		uniq = maybeUniq
		// не уверен, что обман go vet - хорошая практика, но зато весело.
		// а проблема в том, что go vet выполнение цикла гарантированно один
		// раз считает признаком бага. этот обман - попытка сказать "я точно
		// знаю, что делаю и беру на себя ответственность". мб есть другой
		// способ это сказать, или другой способ сэкономить вложенность?
		if uniq == maybeUniq {
			break
		}
	}

	if len(uniq) == 0 {
		uniq = utils.RandStringN(8)
		hmacer := hmac.New(sha256.New, []byte(cookieSecretKey))
		hmacer.Write([]byte(uniq))
		sign := hmacer.Sum(nil)
		cookieValue := fmt.Sprintf("%s-%s", uniq, hex.EncodeToString(sign))
		c.SetCookie(
			uniqCookieName, cookieValue, uniqCookieMaxAge, "/",
			env.Config.CookieDomain, false, true,
		)
		logger.Info().Msg("set uniq cookie " + cookieValue)
	}

	c.Set("uniq", uniq)

	c.Next()
}

func SetupRouter(env *env.Environment, store *storage.Storage) *gin.Engine {
	gin.DisableConsoleColor()
	gin.DefaultWriter = io.MultiWriter(env.LogFile)

	router := gin.Default()
	router.Use(SetGlobalVars(env, store))
	router.Use(GzipCompression)
	router.Use(SetCookies)

	router.POST("/", handlers.BodyShorten)
	router.GET("/:id", handlers.Redirect)
	router.POST("/api/shorten", handlers.APIShorten)
	router.GET("/api/user/urls", handlers.GetAllUserURLs)
	router.GET("/ping", handlers.Ping)

	return router
}
