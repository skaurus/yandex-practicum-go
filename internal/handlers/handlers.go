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

func BodyShorten(c *gin.Context) {
	storage := c.MustGet("storage").(*storage.Storage)
	config := c.MustGet("config").(*config.Config)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if len(body) == 0 {
		c.String(http.StatusBadRequest, ErrEmptyURL)
		return
	}

	newID := (*storage).Shorten(string(body))
	u, _ := url.Parse(fmt.Sprintf("./%d", newID))

	c.String(http.StatusCreated, config.BaseURI.ResolveReference(u).String())
}

type APIRequest struct {
	URL string `json:"url"`
}

func APIShorten(c *gin.Context) {
	storage := c.MustGet("storage").(*storage.Storage)
	config := c.MustGet("config").(*config.Config)

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

	newID := (*storage).Shorten(string(data.URL))
	u, _ := url.Parse(fmt.Sprintf("./%d", newID))

	c.PureJSON(http.StatusCreated, gin.H{"result": config.BaseURI.ResolveReference(u).String()})
}

func Get(c *gin.Context) {
	storage := c.MustGet("storage").(*storage.Storage)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "wrong id")
		return
	}
	url, ok := (*storage).Unshorten(id)
	if !ok {
		c.String(http.StatusBadRequest, "wrong id")
		return
	}
	c.Header("Location", url)
	c.String(http.StatusTemporaryRedirect, "")
}
