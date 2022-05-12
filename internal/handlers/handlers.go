package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/skaurus/yandex-practicum-go/internal/storage"

	"github.com/gin-gonic/gin"
	//jsoniter "github.com/json-iterator/go"
)

const (
	ErrEmptyURL = "empty url"
)

func BodyShorten(c *gin.Context) {
	storage := c.MustGet("storage").(storage.Storage)

	url, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if len(url) == 0 {
		c.String(http.StatusBadRequest, ErrEmptyURL)
		return
	}

	newID := storage.Shorten(string(url))
	c.String(http.StatusCreated, "http://localhost:8080/%d", newID)
}

type APIRequest struct {
	URL string `json:"url"`
}

func APIShorten(c *gin.Context) {
	storage := c.MustGet("storage").(storage.Storage)

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

	shortenedURL := fmt.Sprintf("http://localhost:8080/%d", storage.Shorten(data.Url))
	c.PureJSON(http.StatusCreated, gin.H{"result": shortenedURL})
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
