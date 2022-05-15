package storage

type Storage interface {
	Shorten(string) int
	Unshorten(int) (string, bool)
}

type storageType int

const (
	Memory storageType = iota + 1
)

type memoryStorage struct {
	// это странное решение нужно, чтобы при передаче по ссылке инстанса memoryStorage
	// counter в этом инстансе не копировался, а оставался общим (нужно в тестах,
	// чтобы общий storage у двух разных router работал корректно)
	counter *int
	store   map[int]string
}

func Ptr[T any](v T) *T {
	return &v
}

func New(typ storageType) Storage {
	switch typ {
	case Memory:
		return &memoryStorage{Ptr[int](0), make(map[int]string)}
	default:
		panic("unacceptable!")
	}
}

func (s memoryStorage) Shorten(u string) int {
	//log.Print(fmt.Sprintf("Store: %v", s))
	*s.counter++
	s.store[*s.counter] = u
	return *s.counter
}

func (s memoryStorage) Unshorten(id int) (string, bool) {
	url, ok := s.store[id]
	return url, ok
}
