package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/skaurus/yandex-practicum-go/internal/env"
	"io"
	"os"
)

type fileStorage struct {
	file          *os.File
	encoder       *json.Encoder
	decoder       *json.Decoder
	memoryStorage // распаршенное содержимое файла
}

func NewFileStorage(env *env.Environment) (*fileStorage, error) {
	filename := env.Config.StorageFileName
	// проверка консистентности перед стартом
	_, err := os.OpenFile(filename+".new", os.O_RDONLY, 0644)
	if err == nil {
		return nil, fmt.Errorf(`
WARNING - probably, temporary backup file still exists.
It means that last shutdown was unsuccessful
and some shortened urls are probably lost.
To stop seeing this message and start - move that file somewhere
(and maybe look at it later for clues why that happened)`)
	}

	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	s := &fileStorage{
		file:    file,
		encoder: json.NewEncoder(file),
		decoder: json.NewDecoder(file),
		// переиспользуем код хранилища данных в памяти, чтобы хранить
		// распаршенный файл, что без всяких накладных расходов даёт
		// удобный, совместимый интерфейс
		memoryStorage: *NewMemoryStorage(),
	}
	s.encoder.SetEscapeHTML(false)

	var rows shortenedURLs
	err = s.decoder.Decode(&rows)
	if err != nil && err != io.EOF {
		return nil, err
	}

	var maxID int
	for _, row := range rows {
		if row.ID > maxID {
			maxID = row.ID
		}
		s.memoryStorage.idToURLs[row.ID] = row.OriginalURL
		s.memoryStorage.userToIDs[row.AddedBy] = append(s.memoryStorage.userToIDs[row.AddedBy], row.ID)
	}
	s.memoryStorage.counter = IntPtr(maxID)

	return s, nil
}

func (s *fileStorage) Store(ctx context.Context, u string, by string) (int, error) {
	// переиспользуем апи, имеем всегда актуальное состояние базы в памяти
	return s.memoryStorage.Store(ctx, u, by)
}

// TODO: Поискать, можно ли как-то без бойлерплейта сказать коду все
// TODO: "неопределённые" методы пробовать искать в своём поле memoryStorage.
// TODO: То есть что-то вроде объявления наследования
func (s *fileStorage) GetByID(ctx context.Context, id int) (string, error) {
	return s.memoryStorage.GetByID(ctx, id)
}

func (s *fileStorage) GetByURL(ctx context.Context, url string) (shortenedURL, error) {
	return s.memoryStorage.GetByURL(ctx, url)
}

func (s *fileStorage) GetAllUserUrls(ctx context.Context, by string) (shortenedURLs, error) {
	return s.memoryStorage.GetAllUserUrls(ctx, by)
}

func (s *fileStorage) StoreBatch(ctx context.Context, storeBatchRequest *StoreBatchRequest, by string) (*StoreBatchResponse, error) {
	return s.memoryStorage.StoreBatch(ctx, storeBatchRequest, by)
}

// createBackupFile создаёт файл с дампом текущего состояния хранилища
// сохранённых в памяти урлов. Подмену старого дампа новым сделаем, только если
// всё закончится успешно.
func (s *fileStorage) createBackupFile(path string) (*os.File, error) {
	// Сочетание флагов os.O_CREATE|os.O_EXCL требует, чтобы файла не было - а
	// быть он может, только если прошлое сохранение на диск завершилось ошибкой.
	// В таком случае лучше не будем ничего делать, пока оператор не отреагирует.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows := s.memoryStorage.memoryToRows()
	bytes, err := json.Marshal(rows)
	if err != nil {
		return nil, err
	}

	_, err = file.Write(bytes)
	if err != nil {
		return nil, err
	}

	return file, file.Sync()
}

func (s *fileStorage) Close() error {
	newFile, err := s.createBackupFile(s.file.Name() + ".new")
	if err != nil {
		panic(fmt.Errorf(`
Some error happened during dumping state to disk.
WARNING - if it fails because .new file already exists,
this should have not happened at all - we checked at
the start that this file is not here. Do investigate.

If it fails for some other reason - I have no idea, see an error:
%w`, err))
	}

	// заменяем старый бэкап на свежий
	return os.Rename(newFile.Name(), s.file.Name())
}
