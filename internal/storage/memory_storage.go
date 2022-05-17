package storage

type memoryStorage struct {
	// это странное решение - что counter это ссылка - нужно, чтобы при передаче
	// по ссылке инстанса memoryStorage поле counter в этом инстансе не копировалось,
	// а оставалось общим (нужно в тестах, чтобы общий storage у двух разных router
	// работал корректно)
	counter *int
	store   map[int]string
}

// IntPtr - хелпер, чтобы легко было создавать ссылки на literal инты.
// Пример использования - memoryStorage{ counter: IntPtr(0) }
func IntPtr(x int) *int {
	return &x
}

func (s memoryStorage) Shorten(u string) int {
	*s.counter++
	s.store[*s.counter] = u
	return *s.counter
}

func (s memoryStorage) Unshorten(id int) (string, bool) {
	url, ok := s.store[id]
	return url, ok
}

func (s memoryStorage) Close() error {
	return nil
}
