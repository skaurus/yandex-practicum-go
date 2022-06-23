package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

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
				return alreadyURL.ID, ErrDuplicate
			} else {
				logger.Error().Err(err).Msgf("url [%s] is duplicated, but can't find original", url)
				return 0, ErrDuplicateNotFound
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

// handlerRedirect - ищет сокращённый урл по параметру id;
// если урл найден - делает редирект со статусом 307,
// при запросе удалённого URL возвращает статус 410,
// а при остальных ошибках - отвечает статусом 400.
func (app App) handlerRedirect(c *gin.Context) {
	logger := app.env.Logger

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Error().Err(err).Msgf("can't convert [%s] to int", c.Param("id"))
		c.String(http.StatusBadRequest, "wrong id")
		return
	}

	shortURL, err := app.storage.GetByID(c, id)
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

	if shortURL.IsDeleted {
		c.String(http.StatusGone, "")
		return
	}

	c.Header("Location", shortURL.OriginalURL)
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

func (app App) deleteURLs(ctx context.Context, ids []int) error {
	err := app.storage.DeleteByIDMulti(ctx, ids)
	if err != nil {
		app.env.Logger.Error().Err(err).Msgf("can't delete urls %v", ids)
	}
	return err
}

const (
	deleteURLsBatchSize  = 10
	deleteURLsTimeWindow = time.Duration(1e9 * 10)
)

var deleteURLCh chan int = make(chan int, deleteURLsBatchSize)

// queueURLForDelete - добавляет айди урла в очередь на удаление.
func queueURLForDelete(id int) {
	deleteURLCh <- id
}

// deleteQueuedURLs - удаляет урлы со всеми айдишниками, что накопились в
// deleteCh, либо когда он заполнен, либо по таймеру. Таймер после этого
// в любом случае запускается заново.
func (app App) deleteQueuedURLs() {
	var buffer []int
	var ctx context.Context
	var cancel context.CancelFunc
	clearBuffer := func() {
		buffer = make([]int, 0, deleteURLsBatchSize)
		app.env.Logger.Info().Msg("clearBuffer")
	}
	startTimer := func() {
		if cancel != nil {
			cancel()
		}
		ctx, cancel = context.WithTimeout(context.Background(), deleteURLsTimeWindow)
		app.env.Logger.Info().Msg("startTimer")
	}
	clearBuffer()
	startTimer()
	defer cancel() // I guess should never happen

	for {
		select {
		case id := <-deleteURLCh:
			buffer = append(buffer, id)
			if len(buffer) >= deleteURLsBatchSize {
				app.env.Logger.Info().Msgf("batch >= %d", deleteURLsBatchSize)
				err := app.deleteURLs(context.Background(), buffer)
				if err == nil {
					clearBuffer()
					startTimer()
				}
			}
		case <-ctx.Done():
			app.env.Logger.Info().Msg("batch timeout")
			startTimer()
			if len(buffer) > 0 {
				app.env.Logger.Info().Msg("batch > 0")
				err := app.deleteURLs(context.Background(), buffer)
				if err == nil {
					clearBuffer()
				}
			}
		}
	}
}

// handlerDeleteURLs - асинхронный хендлер; он принимает в теле запроса
// список айди урлов на удаление в виде JSON с массивом строк в body.
// В случае успешного добавления задания в очередь должен возвращать
// HTTP-статус 202 Accepted.
// Фактический результат удаления может происходить позже — каким-либо
// образом оповещать пользователя об успешности или неуспешности не нужно.
// Успешно удалить URL может пользователь, его создавший.
func (app App) handlerDeleteURLs(c *gin.Context) {
	logger := app.env.Logger
	uniq := c.MustGet("uniq").(string)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("can't read request body")
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	var sids []string
	err = json.Unmarshal(body, &sids)
	if err != nil {
		logger.Error().Err(err).Msgf("can't parse body %s", body)
		c.String(http.StatusBadRequest, "can't parse json")
		return
	}
	ids := make([]int, 0, len(sids))
	for _, v := range sids {
		id, err := strconv.Atoi(v)
		if err != nil {
			logger.Error().Err(err).Msgf("can't convert [%s] to int", v)
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		logger.Warn().Msg("empty batch")
		c.String(http.StatusBadRequest, "empty batch")
		return
	}

	// тут бы запрашивать не все сразу, а пачками например по 100;
	// плюс повесить ограничение на максимальное число урлов, что
	// можно удалить за раз
	shortenedURLs, err := app.storage.GetByIDMulti(c, ids)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			logger.Warn().Msgf("can't find ids %s", body)
			c.String(http.StatusBadRequest, "wrong ids")
		} else {
			logger.Error().Err(err).Msgf("can't find ids %s", body)
			c.String(http.StatusBadRequest, err.Error())
		}
		return
	}

	hasSomethingToDelete := false
	for _, shortURL := range shortenedURLs {
		if shortURL.AddedBy != uniq {
			continue
		}

		queueURLForDelete(shortURL.ID)
		hasSomethingToDelete = true
	}

	if hasSomethingToDelete {
		c.String(http.StatusAccepted, "")
		return
	} else {
		// намеренно не буду показывать ошибку. ишь чо удумал, чужие урлы удалять
		c.String(http.StatusBadRequest, "")
		return
	}
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
