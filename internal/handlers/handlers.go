package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/skaurus/yandex-practicum-go/internal/config"
	"github.com/skaurus/yandex-practicum-go/internal/storage"

	"github.com/gin-gonic/gin"
	//jsoniter "github.com/json-iterator/go"
)

const (
	ErrEmptyURL = "empty url"
)

func createRedirectURL(baseURI *url.URL, newID int) string {
	u, _ := url.Parse(fmt.Sprintf("./%d", newID))
	return baseURI.ResolveReference(u).String()
}

func BodyShorten(c *gin.Context) {
	storage := c.MustGet("storage").(*storage.Storage)
	config := c.MustGet("config").(*config.Config)
	uniq := c.MustGet("uniq").(string)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if len(body) == 0 {
		c.String(http.StatusBadRequest, ErrEmptyURL)
		return
	}

	newID := (*storage).Store(string(body), uniq)
	c.String(http.StatusCreated, createRedirectURL(config.BaseURI, newID))
}

type APIRequest struct {
	URL string `json:"url"`
}

func APIShorten(c *gin.Context) {
	storage := c.MustGet("storage").(*storage.Storage)
	config := c.MustGet("config").(*config.Config)
	uniq := c.MustGet("uniq").(string)

	// с использованием этой библиотеки не проходили тесты Практикума
	//var json = jsoniter.ConfigCompatibleWithStandardLibrary

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	var data APIRequest
	err = json.Unmarshal(body, &data)
	if err != nil {
		c.String(http.StatusBadRequest, "can't parse json")
		return
	}
	if len(data.URL) == 0 {
		c.String(http.StatusBadRequest, ErrEmptyURL)
		return
	}

	newID := (*storage).Store(data.URL, uniq)
	c.PureJSON(http.StatusCreated, gin.H{"result": createRedirectURL(config.BaseURI, newID)})
}

func Redirect(c *gin.Context) {
	storage := c.MustGet("storage").(*storage.Storage)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "wrong id")
		return
	}
	originalURL, ok := (*storage).GetByID(id)
	if !ok {
		c.String(http.StatusBadRequest, "wrong id")
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
	storage := c.MustGet("storage").(*storage.Storage)
	config := c.MustGet("config").(*config.Config)
	uniq := c.MustGet("uniq").(string)

	ids := (*storage).GetAllIDsFromUser(uniq)

	answer := make(allUserURLs, len(ids))
	for _, id := range ids {
		originalURL, ok := (*storage).GetByID(id)
		if !ok {
			continue
		}
		answer = append(answer, userURLRow{
			ShortURL:    createRedirectURL(config.BaseURI, id),
			OriginalURL: originalURL,
		})
	}
	if len(answer) == 0 {
		c.String(http.StatusNoContent, "")
		return
	}
	c.PureJSON(http.StatusOK, answer)
}
