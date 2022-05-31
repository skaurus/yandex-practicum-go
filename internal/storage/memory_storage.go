package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type memoryStorage struct {
	// Это странное решение - что counter это ссылка - нужно, чтобы при передаче
	// по ссылке инстанса memoryStorage поле counter в этом инстансе не копировалось,
	// а оставалось общим (нужно в тестах, чтобы общий storage у двух разных router
	// работал корректно)
	counter   *int
	idToURLs  map[int]string
	userToIDs map[string][]int
}

// IntPtr - хелпер, чтобы легко было создавать ссылки на literal инты.
// Пример использования - memoryStorage{ counter: IntPtr(0) }
// Был вариант красивого решения с дженериками, но в тестах старый Go.
func IntPtr(x int) *int {
	return &x
}

func NewMemoryStorage() *memoryStorage {
	return &memoryStorage{IntPtr(0), make(map[int]string), make(map[string][]int)}
}

func (s memoryStorage) Store(u string, by string) (int, error) {
	*s.counter++
	s.idToURLs[*s.counter] = u
	s.userToIDs[by] = append(s.userToIDs[by], *s.counter)
	return *s.counter, nil
}

func (s memoryStorage) GetByID(id int) (string, error) {
	url, ok := s.idToURLs[id]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

func (s memoryStorage) GetAllIDsFromUser(by string) ([]int, error) {
	ids, ok := s.userToIDs[by]
	var err error
	if !ok {
		err = errors.New(utils.StorageErrNotFound)
	}
	return ids, err
}

func (s memoryStorage) Close() error {
	return nil
}
