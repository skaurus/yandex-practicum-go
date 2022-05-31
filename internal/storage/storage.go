package storage

import (
	"errors"

	"github.com/skaurus/yandex-practicum-go/internal/config"
)

type Storage interface {
	Store(string, string) (int, error)
	GetByID(int) (string, error)
	GetAllUserUrls(string) (shortenedURLs, error)
	Close() error
}

var ErrNotFound = errors.New("not found")

// New - в декларации метода указано, что он возвращает тип Storage;
// при этом значения, которые возвращает return - это на самом деле ссылки;
// но всё же в вызывающем коде (main.go) мы получаем value, а не ссылки.
// Что не оптимально по скорости. Но если попробовать изменить тип в
// декларации, компилятор будет ругаться, что мы возвращаем не тот тип:
// cannot use ... (type *memoryStorage) as the type *Storage
// TODO: Как быть?
func New(config *config.Config) Storage {
	if len(config.DBConnectString) > 0 {
		storage, err := NewDBStorage(config)
		if err != nil {
			panic(err)
		}
		return storage
	} else if len(config.StorageFileName) > 0 {
		storage, err := NewFileStorage(config.StorageFileName)
		if err != nil {
			panic(err)
		}
		return storage
	} else {
		return NewMemoryStorage()
	}
}
