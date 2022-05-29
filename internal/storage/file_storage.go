package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// опишем тип "строки" с данными; будем энкодить его в файле как JSON,
// но сэкономим на повторении ключей: https://eagain.net/articles/go-json-array-to-struct/
// будущее описание строки таблицы в базе данных, скорее всего
type shortenedURL struct {
	id          int
	originalUrl string
	addedBy     string
}
type shortenedURLs []shortenedURL

func (s shortenedURL) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`[%d,"%s","%s"]`, s.id, s.originalUrl, s.addedBy)), nil
}

func (s shortenedURLs) MarshalJSON() ([]byte, error) {
	var rowJson []byte
	var err error

	// https://stackoverflow.com/a/1766304/320345
	// интересно, что вариант с strings.Builder в 1.5 раза быстрее на конкатенировании
	// 10 слайсов []byte, с 50 уже чуть медленнее, и дальше отставание нарастает
	var bytesBuffer bytes.Buffer

	for _, v := range s {
		rowJson, err = v.MarshalJSON()
		if err != nil {
			return nil, err
		}
		bytesBuffer.Write(rowJson)
		bytesBuffer.WriteString(",")
	}

	return bytesBuffer.Bytes(), nil
}

func (s *shortenedURL) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&s.id, &s.originalUrl, &s.addedBy}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf("wrong number of fields in Notification: %d != %d", g, e)
	}
	return nil
}

type fileStorage struct {
	file          *os.File
	encoder       *json.Encoder
	decoder       *json.Decoder
	memoryStorage // распаршенное содержимое файла
}

func NewFileStorage(filename string) (*fileStorage, error) {
	// проверка консистентности перед стартом
	file, err := os.OpenFile(filename+".new", os.O_RDONLY, 0644)
	if err == nil {
		return nil, fmt.Errorf(`
WARNING - probably, temporary backup file still exists.
It means that last shutdown was unsuccessful
and some shortened urls are probably lost.
To stop seeing this message and start - move that file somewhere
(and maybe look at it later for clues why that happened).
`)
	}

	file, err = os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	// file.Close() структуры делается в методе Close() самой структуры,
	// см. ниже; даже если file.Close() забыть, эта структура выходит
	// из зоны видимости только при шатдауне сервера, вроде не страшно.
	//defer file.Close()

	s := &fileStorage{
		file:    file,
		encoder: json.NewEncoder(file),
		decoder: json.NewDecoder(file),
		// переиспользуем код хранилища данных в памяти, чтобы хранить
		// распаршенный файл, что без всяких накладных расходов даёт
		// удобный, совместимый интерфейс
		memoryStorage: New(Memory, ConnectInfo{}).(memoryStorage),
	}
	s.encoder.SetEscapeHTML(false)

	var rows shortenedURLs
	err = s.decoder.Decode(&rows)
	if err != nil && err != io.EOF {
		return nil, err
	}

	var maxID int
	for _, row := range rows {
		if row.id > maxID {
			maxID = row.id
		}
		s.memoryStorage.idToURLs[row.id] = row.originalUrl
		s.userToIDs[row.addedBy] = append(s.userToIDs[row.addedBy], row.id)
	}
	s.memoryStorage.counter = IntPtr(maxID)

	return s, nil
}

func (s fileStorage) Store(u string, by string) int {
	// переиспользуем апи, имеем всегда актуальное состояние базы в памяти
	s.memoryStorage.Store(u, by)

	// TODO: попробовать разобраться, точно ли дело в этом, и можно ли это
	// TODO: поправить. А то как-то не внушает надёжности. И без этого метод
	// TODO: мог бы быть вообще однострочным
	// так как при остановке проекта в GoLand не вызывается defer s.Close() в
	// main.go - приходится писать в файл при каждом сокращении ссылки
	/*bytes, err := json.Marshal(&s.memoryStorage.store)
	if err != nil {
		panic(err)
	}
	_, err = s.file.WriteAt(bytes, 0)
	if err != nil {
		panic(err)
	}*/

	return *s.counter
}

// TODO: поискать, можно ли как-то без бойлерплейта сказать коду все
// TODO: "неопределённые" методы пробовать искать в своём поле memoryStorage.
// TODO: То есть что-то вроде объявления наследования
func (s fileStorage) GetByID(id int) (string, bool) {
	return s.memoryStorage.GetByID(id)
}

func (s fileStorage) GetAllIDsFromUser(by string) []int {
	return s.memoryStorage.GetAllIDsFromUser(by)
}

// 1. несколько беспардонно выглядит лезть тут во внутренние поля класса-основы, но
// если делать это в нём - то в него надо нести знание о строках, а это ещё страннее
// 2. также, самонадеянно выглядит отсутствие error в ответе. возможна ли она тут и
// что вообще ей считать - хороший вопрос, оставшийся за скобками
func (s fileStorage) memoryToRows() *shortenedURLs {
	// понятно, что скорее всего такой длины не хватит, но как стартовая точка...
	rows := make(shortenedURLs, len(s.memoryStorage.userToIDs))
	for user, ids := range s.memoryStorage.userToIDs {
		for _, id := range ids {
			// жалко создавать переменную для originalUrl - хотя тогда создание
			// строки было бы более читаемо. может, зря жалко?
			row := shortenedURL{id, s.memoryStorage.idToURLs[id], user}
			rows = append(rows, row)
		}
	}
	return &rows
}

func (s fileStorage) createBackupFile(path string) (*os.File, error) {
	// пишем всё в файл рядом и только если завершили успешно - подменяем
	// файлы. сочетание флагов os.O_CREATE|os.O_EXCL требует, чтобы файла
	// не было - а быть он может только если прошлое сохранение на диск
	// завершилось ошибкой. поэтому пусть оператор сначала отреагирует,
	// прежде чем мы например перезапишем этот файл
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows := s.memoryToRows()
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

func (s fileStorage) Close() error {
	newFile, err := s.createBackupFile(s.file.Name() + ".new")
	if err == nil {
		err = s.file.Close()
	}
	if err != nil {
		// TODO: не уверен, что будет если тут просто вернуть ошибку; хочется
		// TODO: быть уверенным в том, что это точно заметят. так - шансы выше
		panic(fmt.Errorf(`
Some error happened during dumping state to disk.
WARNING - if it fails because file already exists,
this should have not happened at all - we checked at
the start that this file is not here. Do investigate.

If it fails for some other reason - I have no idea, see an error:
%w`, err))
	}

	// заменяем старую базу на свежий бэкап (и молимся)
	return os.Rename(newFile.Name(), s.file.Name())
}
