package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/skaurus/yandex-practicum-go/internal/storage"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

const (
	ErrEmptyUrl = "empty url"
)

func BodyShorten(c *gin.Context) {
	storage := c.MustGet("storage").(storage.Storage)

	url, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if len(url) == 0 {
		c.String(http.StatusBadRequest, ErrEmptyUrl)
		return
	}

	newID := storage.Shorten(string(url))
	c.String(http.StatusCreated, "http://localhost:8080/%d", newID)
}

type ApiRequest struct {
	Url string `json:"url"`
}

func ApiShorten(c *gin.Context) {
	storage := c.MustGet("storage").(storage.Storage)

	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	var data ApiRequest
	err = json.Unmarshal(body, &data)
	if err != nil {
		c.String(http.StatusBadRequest, "can't parse json")
		return
	}
	if len(data.Url) == 0 {
		c.String(http.StatusBadRequest, ErrEmptyUrl)
		return
	}

	shortenedUrl := fmt.Sprintf("http://localhost:8080/%d", storage.Shorten(data.Url))
	c.PureJSON(http.StatusCreated, gin.H{"result": shortenedUrl})
}

func Get(c *gin.Context) {
	storage := c.MustGet("storage").(storage.Storage)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "wrong id")
		return
	}
	url, ok := storage.Unshorten(id)
	if !ok {
		c.String(http.StatusBadRequest, "wrong id")
		return
	}
	c.Header("Location", url)
	c.String(http.StatusTemporaryRedirect, "")
}
