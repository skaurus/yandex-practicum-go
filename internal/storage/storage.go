package storage

type Storage interface {
	Store(string, string) int
	GetByID(int) (string, bool)
	GetAllIDsFromUser(string) []int
	Close() error
}

type storageType int

const (
	Memory storageType = iota + 1
	File
)

type ConnectInfo struct {
	Filename string
}

// New - в декларации метода указано, что он возвращает тип Storage;
// при этом значения, которые возвращает return - это на самом деле ссылки;
// но всё же в вызывающем коде (main.go) мы получаем value, а не ссылки.
// Что не оптимально по скорости. Но если попробовать изменить тип в
// декларации, компилятор будет ругаться, что мы возвращаем не тот тип:
// cannot use ... (type *memoryStorage) as the type *Storage
// TODO: Как быть?
func New(typ storageType, ci ConnectInfo) Storage {
	switch typ {
	case Memory:
		return &memoryStorage{IntPtr(0), make(map[int]string), make(map[string][]int)}
	case File:
		storage, err := NewFileStorage(ci.Filename)
		if err != nil {
			panic(err)
		}
		return storage
	default:
		panic("unacceptable!")
	}
}
