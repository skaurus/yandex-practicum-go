package storage

type Storage interface {
	Shorten(string) int
	Unshorten(int) (string, bool)
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
		return &memoryStorage{IntPtr(0), make(map[int]string)}
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
