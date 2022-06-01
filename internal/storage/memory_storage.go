package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
)

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

func (s *memoryStorage) GetByID(id int) (string, error) {
	url, ok := s.idToURLs[id]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

// опишем тип "строки" с данными; будем энкодить его в файле как JSON,
// но сэкономим на повторении ключей: https://eagain.net/articles/go-json-array-to-struct/
// будущее описание строки таблицы в базе данных, скорее всего
type shortenedURL struct {
	ID          int
	OriginalURL string
	AddedBy     string
}
type shortenedURLs []shortenedURL

func (s shortenedURL) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`[%d,"%s","%s"]`, s.ID, s.OriginalURL, s.AddedBy)), nil
}

func (s shortenedURLs) MarshalJSON() ([]byte, error) {
	var rowJSON []byte
	var err error

	// https://stackoverflow.com/a/1766304/320345
	// интересно, что вариант с strings.Builder в 1.5 раза быстрее на конкатенировании
	// 10 слайсов []byte, с 50 уже чуть медленнее, и дальше отставание нарастает
	var bytesBuffer bytes.Buffer

	bytesBuffer.WriteString("[")
	for _, v := range s {
		rowJSON, err = v.MarshalJSON()
		if err != nil {
			return nil, err
		}
		bytesBuffer.Write(rowJSON)
		bytesBuffer.WriteString(",")
	}
	if len(s) > 0 { // отрезаем лишнюю запятую
		bytesBuffer.Truncate(bytesBuffer.Len() - 1)
	}
	bytesBuffer.WriteString("]")

	return bytesBuffer.Bytes(), nil
}

func (s *shortenedURL) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&s.ID, &s.OriginalURL, &s.AddedBy}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf("wrong number of fields in Notification: %d != %d", g, e)
	}
	return nil
}

// 2. также, самонадеянно выглядит отсутствие error в ответе. возможна ли она тут и
// что вообще ей считать - хороший вопрос, оставшийся за скобками
func (s *memoryStorage) memoryToRows() *shortenedURLs {
	// понятно, что скорее всего такой длины не хватит, но как стартовая точка...
	rows := make(shortenedURLs, 0, len(s.userToIDs))
	for user, ids := range s.userToIDs {
		for _, id := range ids {
			// жалко создавать переменную для OriginalURL - хотя тогда создание
			// строки было бы более читаемо. может, зря жалко?
			row := shortenedURL{id, s.idToURLs[id], user}
			rows = append(rows, row)
		}
	}
	return &rows
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
