package storage

import "context"

type memoryStorage struct {
	// Это странное решение - что counter это ссылка - нужно, чтобы при передаче
	// по ссылке инстанса memoryStorage поле counter в этом инстансе не копировалось,
	// а оставалось общим (нужно в тестах, чтобы общий storage у двух разных router
	// работал корректно)
	counter   *int
	idToURLs  map[int]shortenedURL
	userToIDs map[string][]int
}

// IntPtr - хелпер, чтобы легко было создавать ссылки на literal инты.
// Пример использования - memoryStorage{ counter: IntPtr(0) }
// Был вариант красивого решения с дженериками, но в тестах старый Go.
func IntPtr(x int) *int {
	return &x
}

func NewMemoryStorage() memoryStorage {
	return memoryStorage{IntPtr(0), make(map[int]shortenedURL), make(map[string][]int)}
}

func (s memoryStorage) Store(ctx context.Context, u string, by string) (int, error) {
	*s.counter++
	s.idToURLs[*s.counter] = shortenedURL{*s.counter, u, by}
	s.userToIDs[by] = append(s.userToIDs[by], *s.counter)
	return *s.counter, nil
}

func (s memoryStorage) StoreBatch(ctx context.Context, storeBatchRequest *StoreBatchRequest, by string) (*StoreBatchResponse, error) {
	answer := make(StoreBatchResponse, 0, len(*storeBatchRequest))
	for _, record := range *storeBatchRequest {
		newID, err := s.Store(ctx, record.OriginalURL, by)
		if err != nil {
			return nil, err
		}
		answer = append(answer, storeBatchResponseRecord{record.CorrelationID, newID})
	}
	return &answer, nil
}

func (s memoryStorage) GetByID(ctx context.Context, id int) (shortenedURL, error) {
	answer, ok := s.idToURLs[id]
	if !ok {
		return shortenedURL{}, newError(errNotFound, nil)
	}

	return answer, nil
}

func (s memoryStorage) GetByURL(ctx context.Context, url string) (shortenedURL, error) {
	// текущая структура хранения не очень удобна для этого метода;
	// ну что делать, применим брутфорс
	for _, storedURL := range s.idToURLs {
		if storedURL.OriginalURL == url {
			return storedURL, nil
		}
	}

	return shortenedURL{}, newError(errNotFound, nil)
}

func (s memoryStorage) GetAllUserUrls(ctx context.Context, by string) (shortenedURLs, error) {
	ids, ok := s.userToIDs[by]
	if !ok {
		return nil, ErrNotFound
	}

	answer := make(shortenedURLs, 0, len(ids))
	for _, id := range ids {
		shortenedURL, err := s.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		answer = append(answer, shortenedURL)
	}

	return answer, nil
}

func (s memoryStorage) Close() error {
	return nil
}
