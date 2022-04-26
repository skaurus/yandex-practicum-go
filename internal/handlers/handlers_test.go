package handlers

import (
	"bytes"
	"github.com/skaurus/yandex-practicum-go/internal/storage"
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

func TestCreateHandler(t *testing.T) {
	storage := storage.New(storage.Memory)
	//storage.Shorten(YA)
	//storage.Shorten(Google)
	handler := CreateHandler(storage)

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
				body:        "empty url\n",
				contentType: "text/plain",
			},
		},
		{
			name:   "negative test #2",
			url:    "/search",
			method: http.MethodPost,
			body:   "",
			want: want{
				code:        400,
				body:        "no handler defined\n",
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
				code:           400,
				body:           "wrong id\n",
				contentType:    "text/plain",
				locationHeader: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.url, bytes.NewBuffer([]byte(tt.body)))

			// создаём новый Recorder
			w := httptest.NewRecorder()
			// определяем хендлер
			h := http.HandlerFunc(handler)
			// запускаем сервер
			h.ServeHTTP(w, request)
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
