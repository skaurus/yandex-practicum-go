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
	counter int
	store   map[int]string
}

func New(typ storageType) Storage {
	switch typ {
	case Memory:
		return &memoryStorage{0, make(map[int]string)}
	default:
		panic("unacceptable!")
	}
}

func (s *memoryStorage) Shorten(u string) int {
	//log.Print(fmt.Sprintf("Store: %v", s))
	s.counter++
	s.store[s.counter] = u
	return s.counter
}

func (s *memoryStorage) Unshorten(id int) (string, bool) {
	url, ok := s.store[id]
	return url, ok
}
