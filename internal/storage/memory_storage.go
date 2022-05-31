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

func (s memoryStorage) Store(u string, by string) int {
	*s.counter++
	s.idToURLs[*s.counter] = u
	s.userToIDs[by] = append(s.userToIDs[by], *s.counter)
	return *s.counter
}

func (s memoryStorage) GetByID(id int) (string, bool) {
	url, ok := s.idToURLs[id]
	return url, ok
}

func (s memoryStorage) GetAllIDsFromUser(by string) []int {
	return s.userToIDs[by]
}

func (s memoryStorage) Close() error {
	return nil
}
