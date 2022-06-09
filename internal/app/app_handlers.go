package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/skaurus/yandex-practicum-go/internal/storage"

	"github.com/gin-gonic/gin"
)

const (
	ErrEmptyURL = "empty url"
)

var ErrDuplicate = errors.New("url is duplicate")
var ErrDuplicateNotFound = errors.New("url is duplicate, but couldn't be found")

func (app App) createRedirectURL(newID int) string {
	u, _ := url.Parse(fmt.Sprintf("./%d", newID))
	return app.env.BaseURI.ResolveReference(u).String()
}

func (app App) storeOneURL(c *gin.Context, url string, addedBy string) (int, error) {
	logger := app.env.Logger

	newID, err := app.storage.Store(c, url, addedBy)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			alreadyURL, err := app.storage.GetByURL(c, url)
			if err == nil {
				logger.Warn().Msgf("url [%s] is duplicated, original is [%d]", url, alreadyURL.ID)
				return newID, ErrDuplicate
			} else {
				logger.Error().Err(err).Msgf("url [%s] is duplicated, but can't find original", url)
				return newID, ErrDuplicateNotFound
			}
		}
		logger.Error().Err(err).Msgf("can't shorten an url [%s] by %s", url, addedBy)
		return 0, err
	}

	return newID, nil
}

func (app App) handlerBodyShorten(c *gin.Context) {
	logger := app.env.Logger
	uniq := c.MustGet("uniq").(string)

	bodyB, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("can't read request body")
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	body := string(bodyB)
	if len(body) == 0 {
		logger.Warn().Msg("empty body")
		c.String(http.StatusBadRequest, ErrEmptyURL)
		return
	}

	newID, err := app.storeOneURL(c, body, uniq)
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			c.String(http.StatusConflict, app.createRedirectURL(newID))
			return
		}
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.String(http.StatusCreated, app.createRedirectURL(newID))
}

type APIRequest struct {
	URL string `json:"url"`
}

func (app App) handlerAPIShorten(c *gin.Context) {
	logger := app.env.Logger
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

	newID, err := app.storeOneURL(c, data.URL, uniq)
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			c.PureJSON(http.StatusConflict, gin.H{"result": app.createRedirectURL(newID)})
			return
		}
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.PureJSON(http.StatusCreated, gin.H{"result": app.createRedirectURL(newID)})
}

type apiBatchResponseRow struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type APIBatchResponse []apiBatchResponseRow

func (app App) handlerAPIShortenBatch(c *gin.Context) {
	logger := app.env.Logger
	uniq := c.MustGet("uniq").(string)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("can't read request body")
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	var data storage.StoreBatchRequest
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Error().Err(err).Msg("can't parse body")
		c.String(http.StatusBadRequest, "can't parse json")
		return
	}
	if len(data) == 0 {
		logger.Warn().Msg("empty batch")
		c.String(http.StatusBadRequest, "empty batch")
		return
	}

	rows, err := app.storage.StoreBatch(c, &data, uniq)
	if err != nil {
		logger.Error().Err(err).Msgf("can't shorten an url [%s] by %s", data, uniq)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	answer := make(APIBatchResponse, 0, len(*rows))
	for _, row := range *rows {
		answer = append(answer, apiBatchResponseRow{
			CorrelationID: row.CorrelationID,
			ShortURL:      app.createRedirectURL(row.ID),
		})
	}

	c.PureJSON(http.StatusCreated, answer)
}

func (app App) handlerRedirect(c *gin.Context) {
	logger := app.env.Logger

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Error().Err(err).Msgf("can't convert [%s] to int", c.Param("id"))
		c.String(http.StatusBadRequest, "wrong id")
		return
	}

	originalURL, err := app.storage.GetByID(c, id)
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

func (app App) handlerGetAllUserURLs(c *gin.Context) {
	logger := app.env.Logger
	uniq := c.MustGet("uniq").(string)

	rows, err := app.storage.GetAllUserUrls(c, uniq)
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
			ShortURL:    app.createRedirectURL(row.ID),
			OriginalURL: row.OriginalURL,
		})
	}

	if len(answer) == 0 {
		c.String(http.StatusNoContent, "")
		return
	}
	c.PureJSON(http.StatusOK, answer)
}

func (app App) handlerPing(c *gin.Context) {
	logger := app.env.Logger

	// Это вынесено отдельно, потому что с пустой строкой драйвер всё равно
	// пытается подключиться, с параметрами по умолчанию (текущий юзер,
	// база = текущему юзеру, без пароля), обламывается, и светит в логи юзера
	if app.env.DBConn == nil {
		logger.Error().Msg("no db connection string was provided, nothing to ping")
		c.String(http.StatusInternalServerError, "")
		return
	}

	err := app.env.DBConn.Ping(c)
	if err != nil {
		logger.Error().Err(err).Msg("db ping failed")
		c.String(http.StatusInternalServerError, "")
		return
	}

	c.String(http.StatusOK, "")
}
