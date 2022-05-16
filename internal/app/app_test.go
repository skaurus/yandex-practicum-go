package app

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/skaurus/yandex-practicum-go/internal/config"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
)

const (
	YA     = "https://ya.ru"
	Google = "https://google.com"
)

func TestRoutes(t *testing.T) {
	storage := storage.New(storage.Memory, storage.ConnectInfo{})
	config := config.ParseConfig()
	router := SetupRouter(&storage, config)

	configWithAnotherBase := *config
	configWithAnotherBase.BaseAddr = "https://ya.us/s/" // yet another url shortener
	configWithAnotherBase.BaseURI, _ = url.Parse(configWithAnotherBase.BaseAddr)

	routerWithAnotherBase := SetupRouter(&storage, &configWithAnotherBase)
	originalRouter := router

	type want struct {
		code           int
		body           string
		contentType    string
		locationHeader string
	}

	counter := 0
	inc := func(i *int) int { *i++; return *i }

	tests := []struct {
		name   string
		url    string
		method string
		body   string
		pre    func()
		post   func()
		want   want
	}{
		{
			name:   "shorting YA via body POST",
			url:    "/",
			method: http.MethodPost,
			body:   YA,
			want: want{
				code:        201,
				body:        fmt.Sprintf("http://localhost:8080/%d", inc(&counter)),
				contentType: "text/plain",
			},
		},
		{
			name:   fmt.Sprintf("fetching just shorted url /%d", counter),
			url:    fmt.Sprintf("/%d", counter),
			method: http.MethodGet,
			body:   "",
			want: want{
				code:           307,
				body:           "",
				contentType:    "text/plain",
				locationHeader: YA,
			},
		},
		{
			name:   "shorting YA via body POST with different base addr",
			url:    "/",
			method: http.MethodPost,
			body:   YA,
			pre: func() {
				router = routerWithAnotherBase
			},
			post: func() {
				router = originalRouter
			},
			want: want{
				code:        201,
				body:        fmt.Sprintf("https://ya.us/s/%d", inc(&counter)),
				contentType: "text/plain",
			},
		},
		{
			name:   fmt.Sprintf("fetching just shorted url /%d", counter),
			url:    fmt.Sprintf("/%d", counter),
			method: http.MethodGet,
			body:   "",
			want: want{
				code:           307,
				body:           "",
				contentType:    "text/plain",
				locationHeader: YA,
			},
		},
		{
			name:   "shorting Google via body POST",
			url:    "/",
			method: http.MethodPost,
			body:   Google,
			want: want{
				code:        201,
				body:        fmt.Sprintf("http://localhost:8080/%d", inc(&counter)),
				contentType: "text/plain",
			},
		},
		{
			name:   fmt.Sprintf("fetching just shorted url /%d", counter),
			url:    fmt.Sprintf("/%d", counter),
			method: http.MethodGet,
			body:   "",
			want: want{
				code:           307,
				body:           "",
				contentType:    "text/plain",
				locationHeader: Google,
			},
		},
		{
			name:   "shorting empty url via body POST",
			url:    "/",
			method: http.MethodPost,
			body:   "",
			want: want{
				code:        400,
				body:        "empty url",
				contentType: "text/plain",
			},
		},
		{
			name:   "fetching wrong url",
			url:    "/search",
			method: http.MethodGet,
			body:   "",
			want: want{
				code:        400,
				body:        "wrong id",
				contentType: "text/plain",
			},
		},
		{
			name:   "fetching non-existing url",
			url:    "/100",
			method: http.MethodGet,
			body:   "",
			want: want{
				code:        400,
				body:        "wrong id",
				contentType: "text/plain",
			},
		},
		{
			name:   "shorting YA via api POST",
			url:    "/api/shorten",
			method: http.MethodPost,
			body:   fmt.Sprintf(`{"url":"%s"}`, YA),
			want: want{
				code:        201,
				body:        fmt.Sprintf(`{"result":"http://localhost:8080/%d"}`, inc(&counter)),
				contentType: "application/json",
			},
		},
		{
			name:   "shorting via api POST with wrong json",
			url:    "/api/shorten",
			method: http.MethodPost,
			body:   YA,
			want: want{
				code:        400,
				body:        "can't parse json",
				contentType: "text/plain",
			},
		},
		{
			name:   "shorting via api POST with empty url",
			url:    "/api/shorten",
			method: http.MethodPost,
			body:   fmt.Sprintf(`{"url":"%s"}`, ""),
			want: want{
				code:        400,
				body:        "empty url",
				contentType: "text/plain",
			},
		},
		{
			name:   fmt.Sprintf("fetching just shorted url /%d", counter),
			url:    fmt.Sprintf("/%d", counter),
			method: http.MethodGet,
			body:   "",
			want: want{
				code:           307,
				body:           "",
				contentType:    "text/plain",
				locationHeader: YA,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pre != nil {
				tt.pre()
			}

			request := httptest.NewRequest(tt.method, tt.url, bytes.NewBuffer([]byte(tt.body)))

			// создаём новый Recorder
			w := httptest.NewRecorder()
			// запускаем сервер
			router.ServeHTTP(w, request)
			res := w.Result()

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if !assert.NoError(t, err, "can read body") {
				return
			}

			assert.Equal(t, tt.want.code, res.StatusCode)
			if tt.want.contentType == "application/json" {
				assert.JSONEq(t, tt.want.body, string(body))
			} else {
				assert.Equal(t, tt.want.body, string(body))
			}

			if len(tt.want.locationHeader) > 0 {
				assert.Equal(t, tt.want.locationHeader, res.Header.Get("Location"))
			}

			if tt.post != nil {
				tt.post()
			}
		})
	}
}
