package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/skaurus/yandex-practicum-go/internal/env"
)

// shortenedURL - "строка" с данными; будем энкодить его в файле как JSON,
// но сэкономим на повторении ключей: https://eagain.net/articles/go-json-array-to-struct/
// будущее описание строки таблицы в базе данных, скорее всего
type shortenedURL struct {
	ID          int
	OriginalURL string
	AddedBy     string
	IsDeleted   bool
}
type shortenedURLs []*shortenedURL

type Storage interface {
	Store(context.Context, string, string) (int, error)
	StoreBatch(context.Context, *StoreBatchRequest, string) (*StoreBatchResponse, error)
	GetByID(context.Context, int) (*shortenedURL, error)
	GetByIDMulti(context.Context, []int) (shortenedURLs, error)
	GetByURL(context.Context, string) (*shortenedURL, error)
	GetAllUserUrls(context.Context, string) (shortenedURLs, error)
	DeleteByID(context.Context, int) error
	DeleteByIDMulti(context.Context, []int) error
	Close() error
}

// storeBatch* используются в "/api/shorten/batch" (handlers.handlerAPIShortenBatch)
type storeBatchRequestRecord struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type StoreBatchRequest []storeBatchRequestRecord

type storeBatchResponseRecord struct {
	CorrelationID string
	ID            int
}

type StoreBatchResponse []storeBatchResponseRecord

const errNotFound = "not found"

type Error struct {
	text string
	err  error
}

func (e *Error) Error() string {
	return e.text
}
func (e *Error) Unwrap() error {
	return e.err
}
func (e *Error) Is(target error) bool {
	return e.Error() == target.Error()
}

func newError(errorText string, originalError error) error {
	return &Error{errorText, originalError}
}

var ErrNotFound = newError(errNotFound, nil)

func (s shortenedURL) MarshalJSON() ([]byte, error) {
	IsDeleted := "false"
	if s.IsDeleted {
		IsDeleted = "true"
	}
	return []byte(fmt.Sprintf(`[%d,"%s","%s",%s]`, s.ID, s.OriginalURL, s.AddedBy, IsDeleted)), nil
}

func (s shortenedURLs) MarshalJSON() ([]byte, error) {
	var rowJSON []byte
	var err error

	// https://stackoverflow.com/a/1766304/320345
	// интересно, что вариант с strings.Builder в 1.5 раза быстрее на конкатенировании
	// 10 слайсов []byte, с 50 уже чуть медленнее, и дальше отставание нарастает
	var bytesBuffer bytes.Buffer

	bytesBuffer.WriteString("[")
	for _, v := range s {
		rowJSON, err = v.MarshalJSON()
		if err != nil {
			return nil, err
		}
		bytesBuffer.Write(rowJSON)
		bytesBuffer.WriteString(",")
	}
	if len(s) > 0 { // отрезаем лишнюю запятую
		bytesBuffer.Truncate(bytesBuffer.Len() - 1)
	}
	bytesBuffer.WriteString("]")

	return bytesBuffer.Bytes(), nil
}

func (s *shortenedURL) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&s.ID, &s.OriginalURL, &s.AddedBy, &s.IsDeleted}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf("wrong number of fields in Notification: %d != %d", g, e)
	}
	return nil
}

// самонадеянно выглядит отсутствие error в ответе. возможна ли она тут и
// что вообще ей считать - хороший вопрос, оставшийся за скобками
func (s memoryStorage) memoryToRows() *shortenedURLs {
	// понятно, что скорее всего такой длины не хватит, но как стартовая точка...
	rows := make(shortenedURLs, 0, len(s.userToIDs))
	for _, row := range s.idToURLs {
		rows = append(rows, row)
	}

	return &rows
}

// New - в декларации метода указано, что он возвращает тип Storage;
// при этом значения, которые возвращает return - это на самом деле ссылки;
// но всё же в вызывающем коде (main.go) мы получаем value, а не ссылки.
// Что не оптимально по скорости. Но если попробовать изменить тип в
// декларации, компилятор будет ругаться, что мы возвращаем не тот тип:
// cannot use ... (type *memoryStorage) as the type *Storage
// TODO: Как быть?
func New(env env.Environment) Storage {
	if len(env.Config.DBConnectString) > 0 {
		storage, err := NewDBStorage(env)
		if err != nil {
			panic(err)
		}
		return storage
	} else if len(env.Config.StorageFileName) > 0 {
		storage, err := NewFileStorage(env)
		if err != nil {
			panic(err)
		}
		return storage
	} else {
		return NewMemoryStorage()
	}
}
