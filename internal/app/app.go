package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strings"
	"time"

	"github.com/skaurus/yandex-practicum-go/internal/env"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
	"github.com/skaurus/yandex-practicum-go/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	uniqCookieName   = "uniq"
	uniqCookieMaxAge = time.Duration(1e9 * 60 * 60 * 24 * 365) // seconds
	cookieSecretKey  = "carrot-james-regular"                  // https://edoceo.com/dev/mnemonic-password-generator
)

var hmacer hash.Hash

// middlewareSetCookies - проставляем/читаем куки
func (app App) middlewareSetCookies(c *gin.Context) {
	logger := app.env.Logger

	var uniq string
	// блок с несколькими последовательными проверками - это способ не делать
	// вложенные один в другой if (success) { ... }
	// range написан так, чтобы for был выполнен ровно один раз
	for range []int{1} {
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
	}

	if len(uniq) == 0 {
		uniq = utils.RandStringN(8)
		if hmacer == nil {
			hmacer = hmac.New(sha256.New, []byte(cookieSecretKey))
		}
		hmacer.Reset()
		hmacer.Write([]byte(uniq))
		sign := hmacer.Sum(nil)
		cookieValue := fmt.Sprintf("%s-%s", uniq, hex.EncodeToString(sign))
		c.SetCookie(
			uniqCookieName, cookieValue, int(uniqCookieMaxAge.Seconds()), "/",
			app.env.Config.CookieDomain, false, true,
		)
		logger.Info().Msg("set uniq cookie " + cookieValue)
	}

	c.Set("uniq", uniq)

	c.Next()
}

type App struct {
	env     env.Environment
	storage storage.Storage
}

func SetupRouter(env env.Environment, store storage.Storage) *gin.Engine {
	gin.DisableConsoleColor()
	gin.DefaultWriter = io.MultiWriter(env.LogFile)

	app := App{
		env,
		store,
	}

	router := gin.Default()
	router.Use(app.middlewareGzipCompression)
	router.Use(app.middlewareSetCookies)

	router.POST("/", app.handlerBodyShorten)
	router.GET("/:id", app.handlerRedirect)
	router.POST("/api/shorten", app.handlerAPIShorten)
	router.POST("/api/shorten/batch", app.handlerAPIShortenBatch)
	router.GET("/api/user/urls", app.handlerGetAllUserURLs)
	router.DELETE("/api/user/urls", app.handlerDeleteURLs)
	go app.deleteQueuedURLs() // background job
	router.GET("/ping", app.handlerPing)

	return router
}
