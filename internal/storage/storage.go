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

func New(typ storageType, ci ConnectInfo) Storage {
	switch typ {
	case Memory:
		return &memoryStorage{IntPtr(0), make(map[int]string), make(map[string][]int)}
	case File:
		storage, err := NewFileStorage(ci.Filename)
		if err != nil {
			panic(err)
		}
		defer storage.Close()
		return storage
	default:
		panic("unacceptable!")
	}
}
