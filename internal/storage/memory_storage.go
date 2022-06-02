package storage

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

func (s *memoryStorage) Store(u string, by string) (int, error) {
	*s.counter++
	s.idToURLs[*s.counter] = u
	s.userToIDs[by] = append(s.userToIDs[by], *s.counter)
	return *s.counter, nil
}

func (s *memoryStorage) StoreBatch(storeBatchRequest *StoreBatchRequest, by string) (*StoreBatchResponse, error) {
	answer := make(StoreBatchResponse, 0, len(*storeBatchRequest))
	for _, record := range *storeBatchRequest {
		newID, err := s.Store(record.OriginalURL, by)
		if err != nil {
			return nil, err
		}
		answer = append(answer, storeBatchResponseRecord{record.CorrelationID, newID})
	}
	return &answer, nil
}

func (s *memoryStorage) GetByID(id int) (string, error) {
	url, ok := s.idToURLs[id]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

func (s *memoryStorage) GetAllUserUrls(by string) (shortenedURLs, error) {
	ids, ok := s.userToIDs[by]
	if !ok {
		return nil, ErrNotFound
	}

	var err error
	answer := make(shortenedURLs, 0, len(ids))
	for _, id := range ids {
		originalURL, err := s.GetByID(id)
		if err != nil {
			return nil, err
		}
		answer = append(answer, shortenedURL{id, originalURL, by})
	}

	return answer, err
}

func (s *memoryStorage) Close() error {
	return nil
}
