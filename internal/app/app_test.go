package app

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	YA     = "https://ya.ru"
	Google = "https://google.com"
)

func TestRoutes(t *testing.T) {
	router := SetupRouter()

	type want struct {
		code           int
		body           string
		contentType    string
		locationHeader string
	}

	tests := []struct {
		name   string
		url    string
		method string
		body   string
		want   want
	}{
		{
			name:   "positive test #1",
			url:    "/",
			method: http.MethodPost,
			body:   YA,
			want: want{
				code:        201,
				body:        "http://localhost:8080/1",
				contentType: "text/plain",
			},
		},
		{
			name:   "positive test #2",
			url:    "/",
			method: http.MethodPost,
			body:   Google,
			want: want{
				code:        201,
				body:        "http://localhost:8080/2",
				contentType: "text/plain",
			},
		},
		{
			name:   "negative test #1",
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
			name:   "negative test #2",
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
			name:   "positive test #3",
			url:    "/1",
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
			name:   "positive test #4",
			url:    "/2",
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
			name:   "negative test #3",
			url:    "/3",
			method: http.MethodGet,
			body:   "",
			want: want{
				code:        400,
				body:        "wrong id",
				contentType: "text/plain",
			},
		},
		{
			name:   "api positive test #1",
			url:    "/api/shorten",
			method: http.MethodPost,
			body:   fmt.Sprintf(`{"url":"%s"}`, YA),
			want: want{
				code:        201,
				body:        `{"result":"http://localhost:8080/3"}`,
				contentType: "application/json",
			},
		},
		{
			name:   "api negative test #1",
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
			name:   "api negative test #2",
			url:    "/api/shorten",
			method: http.MethodPost,
			body:   fmt.Sprintf(`{"url":"%s"}`, ""),
			want: want{
				code:        400,
				body:        "empty url",
				contentType: "text/plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}
