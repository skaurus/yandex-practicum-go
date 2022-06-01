package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/skaurus/yandex-practicum-go/internal/env"
	"github.com/skaurus/yandex-practicum-go/internal/storage"

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
	env := c.MustGet("env").(*env.Environment)
	logger := env.Logger
	store := c.MustGet("storage").(*storage.Storage)
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

	newID, err := (*store).Store(string(body), uniq)
	if err != nil {
		logger.Error().Err(err).Msgf("can't shorten an url [%s] by %s", string(body), uniq)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.String(http.StatusCreated, createRedirectURL(env.BaseURI, newID))
}

type APIRequest struct {
	URL string `json:"url"`
}

func APIShorten(c *gin.Context) {
	env := c.MustGet("env").(*env.Environment)
	logger := env.Logger
	store := c.MustGet("storage").(*storage.Storage)
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

	newID, err := (*store).Store(data.URL, uniq)
	if err != nil {
		logger.Error().Err(err).Msgf("can't shorten an url [%s] by %s", data.URL, uniq)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.PureJSON(http.StatusCreated, gin.H{"result": createRedirectURL(env.BaseURI, newID)})
}

func Redirect(c *gin.Context) {
	env := c.MustGet("env").(*env.Environment)
	logger := env.Logger
	store := c.MustGet("storage").(*storage.Storage)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Error().Err(err).Msgf("can't convert [%s] to int", c.Param("id"))
		c.String(http.StatusBadRequest, "wrong id")
		return
	}

	originalURL, err := (*store).GetByID(id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
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
	env := c.MustGet("env").(*env.Environment)
	logger := env.Logger
	store := c.MustGet("storage").(*storage.Storage)
	uniq := c.MustGet("uniq").(string)

	rows, err := (*store).GetAllUserUrls(uniq)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
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
			ShortURL:    createRedirectURL(env.BaseURI, row.ID),
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
	env := c.MustGet("env").(*env.Environment)
	logger := env.Logger

	// Это вынесено отдельно, потому что с пустой строкой драйвер всё равно
	// пытается подключиться, с параметрами по умолчанию (текущий юзер,
	// база = текущему юзеру, без пароля), обламывается, и светит в логи юзера
	if env.DBConn == nil {
		logger.Error().Msg("no db connection string was provided, nothing to ping")
		c.String(http.StatusInternalServerError, "")
		return
	}

	err := env.DBConn.Ping(c)
	if err != nil {
		logger.Error().Err(err).Msg("db ping failed")
		c.String(http.StatusInternalServerError, "")
		return
	}

	c.String(http.StatusOK, "")
}
