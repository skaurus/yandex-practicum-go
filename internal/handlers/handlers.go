package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"net/url"
	"strconv"

	configpkg "github.com/skaurus/yandex-practicum-go/internal/config"
	storagepkg "github.com/skaurus/yandex-practicum-go/internal/storage"

	"github.com/gin-gonic/gin"
)

const (
	ErrEmptyURL = "empty url"
)

func createRedirectURL(baseURI *url.URL, newID int) string {
	u, _ := url.Parse(fmt.Sprintf("./%d", newID))
	return baseURI.ResolveReference(u).String()
}

func BodyShorten(c *gin.Context) {
	storage := c.MustGet("storage").(*storagepkg.Storage)
	config := c.MustGet("config").(*configpkg.Config)
	logger := c.MustGet("logger").(*zerolog.Logger)
	uniq := c.MustGet("uniq").(string)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("can't read request body")
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if len(body) == 0 {
		logger.Warn().Msg("empty body")
		c.String(http.StatusBadRequest, ErrEmptyURL)
		return
	}

	newID, err := (*storage).Store(string(body), uniq)
	if err != nil {
		logger.Error().Err(err).Msgf("can't shorten an url [%s] by %s", string(body), uniq)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.String(http.StatusCreated, createRedirectURL(config.BaseURI, newID))
}

type APIRequest struct {
	URL string `json:"url"`
}

func APIShorten(c *gin.Context) {
	storage := c.MustGet("storage").(*storagepkg.Storage)
	config := c.MustGet("config").(*configpkg.Config)
	logger := c.MustGet("logger").(*zerolog.Logger)
	uniq := c.MustGet("uniq").(string)

	// с использованием этой библиотеки не проходили тесты Практикума
	//var json = jsoniter.ConfigCompatibleWithStandardLibrary

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("can't read request body")
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	var data APIRequest
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Error().Err(err).Msg("can't parse body")
		c.String(http.StatusBadRequest, "can't parse json")
		return
	}
	if len(data.URL) == 0 {
		logger.Warn().Msg("empty url")
		c.String(http.StatusBadRequest, ErrEmptyURL)
		return
	}

	newID, err := (*storage).Store(data.URL, uniq)
	if err != nil {
		logger.Error().Err(err).Msgf("can't shorten an url [%s] by %s", data.URL, uniq)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.PureJSON(http.StatusCreated, gin.H{"result": createRedirectURL(config.BaseURI, newID)})
}

func Redirect(c *gin.Context) {
	storage := c.MustGet("storage").(*storagepkg.Storage)
	logger := c.MustGet("logger").(*zerolog.Logger)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Error().Err(err).Msgf("can't convert [%s] to int", c.Param("id"))
		c.String(http.StatusBadRequest, "wrong id")
		return
	}

	originalURL, err := (*storage).GetByID(id)
	if err != nil {
		if errors.Is(err, storagepkg.ErrNotFound) {
			logger.Warn().Msgf("can't find id [%d]", id)
			c.String(http.StatusBadRequest, "wrong id")
		} else {
			logger.Error().Err(err).Msgf("can't find id [%d]", id)
			c.String(http.StatusBadRequest, err.Error())
		}
		return
	}

	c.Header("Location", originalURL)
	c.String(http.StatusTemporaryRedirect, "")
}

type userURLRow struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
type allUserURLs []userURLRow

func GetAllUserURLs(c *gin.Context) {
	storage := c.MustGet("storage").(*storagepkg.Storage)
	config := c.MustGet("config").(*configpkg.Config)
	logger := c.MustGet("logger").(*zerolog.Logger)
	uniq := c.MustGet("uniq").(string)

	rows, err := (*storage).GetAllUserUrls(uniq)
	if err != nil {
		if errors.Is(err, storagepkg.ErrNotFound) {
			// это валидный кейс, просто ответим 204
			logger.Warn().Msgf("can't find urls for user [%s]", uniq)
		} else {
			logger.Error().Err(err).Msgf("can't find urls for user [%s]", uniq)
			c.String(http.StatusBadRequest, err.Error())
			return
		}
	}

	answer := make(allUserURLs, 0, len(rows))
	for _, row := range rows {
		answer = append(answer, userURLRow{
			ShortURL:    createRedirectURL(config.BaseURI, row.ID),
			OriginalURL: row.OriginalURL,
		})
	}

	if len(answer) == 0 {
		c.String(http.StatusNoContent, "")
		return
	}
	c.PureJSON(http.StatusOK, answer)
}

func Ping(c *gin.Context) {
	config := c.MustGet("config").(*configpkg.Config)
	logger := c.MustGet("logger").(*zerolog.Logger)

	// Это вынесено отдельно, потому что с пустой строкой драйвер всё равно
	// пытается подключиться, с параметрами по умолчанию (текущий юзер,
	// база = текущему юзеру, без пароля), обламывается, и светит в логи юзера
	if len(config.DBConnectString) == 0 {
		logger.Error().Msg("no db connection string was provided, nothing to ping")
		c.String(http.StatusInternalServerError, "")
		return
	}

	conn, err := pgx.Connect(context.Background(), config.DBConnectString)
	if err != nil {
		logger.Error().Err(err).Send()
		c.String(http.StatusInternalServerError, "")
		return
	}
	defer conn.Close(context.Background())

	c.String(http.StatusOK, "")
}
