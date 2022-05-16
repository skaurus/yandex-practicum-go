package storage

import (
	"encoding/json"
	"os"
)

type fileStorage struct {
	file          *os.File
	encoder       *json.Encoder
	decoder       *json.Decoder
	memoryStorage // распаршенное содержимое файла
}

func NewFileStorage(filename string) (*fileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	s := &fileStorage{
		file:          file,
		encoder:       json.NewEncoder(file),
		decoder:       json.NewDecoder(file),
		memoryStorage: memoryStorage{},
	}
	s.encoder.SetEscapeHTML(false)
	err = s.decoder.Decode(&s.memoryStorage.store)
	if err != nil {
		return nil, err
	}
	var maxID int = 0
	for n := range s.memoryStorage.store {
		if n > maxID {
			maxID = n
		}
	}
	s.memoryStorage.counter = IntPtr(maxID)
	return s, nil
}

func (s fileStorage) Shorten(u string) int {
	*s.counter++
	s.store[*s.counter] = u

	// так как при остановке проекта в GoLand не вызывается defer s.Close() в
	// main.go - приходится писать в файл при каждом сокращении ссылки
	bytes, err := json.Marshal(&s.memoryStorage.store)
	if err != nil {
		panic(err)
	}
	_, err = s.file.WriteAt(bytes, 0)
	if err != nil {
		panic(err)
	}

	return *s.counter
}

func (s fileStorage) Unshorten(id int) (string, bool) {
	url, ok := s.store[id]
	return url, ok
}

// Close не вызывается, если я просто останавливаю проект в GoLand :(
func (s fileStorage) Close() error {
	bytes, err := json.Marshal(&s.memoryStorage.store)
	if err != nil {
		return err
	}
	_, err = s.file.WriteAt(bytes, 0)
	if err != nil {
		return err
	}
	return s.file.Close()
}
