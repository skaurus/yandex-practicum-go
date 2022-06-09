package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strings"

	"github.com/skaurus/yandex-practicum-go/internal/env"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
	"github.com/skaurus/yandex-practicum-go/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	uniqCookieName   = "uniq"
	uniqCookieMaxAge = 60 * 60 * 24 * 365     // seconds
	cookieSecretKey  = "carrot-james-regular" // https://edoceo.com/dev/mnemonic-password-generator
)

var hmacer hash.Hash

// middlewareSetCookies - проставляем/читаем куки
func (app App) middlewareSetCookies(c *gin.Context) {
	logger := app.env.Logger

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
		if hmacer == nil {
			hmacer = hmac.New(sha256.New, []byte(cookieSecretKey))
		}
		hmacer.Reset()
		hmacer.Write([]byte(uniq))
		sign := hmacer.Sum(nil)
		cookieValue := fmt.Sprintf("%s-%s", uniq, hex.EncodeToString(sign))
		c.SetCookie(
			uniqCookieName, cookieValue, uniqCookieMaxAge, "/",
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
	router.GET("/ping", app.handlerPing)

	return router
}
