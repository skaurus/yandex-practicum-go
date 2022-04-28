package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
	"io"
	"net/http"
	"strconv"
)

func Post(c *gin.Context) {
	storage := c.MustGet("storage").(storage.Storage)

	url, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if len(url) == 0 {
		c.String(http.StatusBadRequest, "empty url")
		return
	}
	newID := storage.Shorten(string(url))
	c.String(http.StatusCreated, "http://localhost:8080/%d", newID)
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
