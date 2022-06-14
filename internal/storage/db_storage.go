package storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/skaurus/yandex-practicum-go/internal/env"

	"github.com/jackc/pgx/v4"
)

type dbStorage struct {
	handle *pgx.Conn
}

func NewDBStorage(env env.Environment) (dbStorage, error) {
	db := dbStorage{env.DBConn}

	// создадим основную таблицу данных
	// (вообще лучше было бы использовать какую-нибудь систему миграций)
	err := db.ExecWithTimeout(
		context.Background(), 1*time.Second, `
CREATE TABLE IF NOT EXISTS urls (
	id			 serial PRIMARY KEY,
	original_url text NOT NULL,
	added_by	 text NOT NULL,
	is_deleted   bool NOT NULL DEFAULT 'false'
)`,
	)
	if err != nil {
		return dbStorage{}, err
	}

	err = db.ExecWithTimeout(
		context.Background(), 1*time.Second,
		"CREATE UNIQUE INDEX IF NOT EXISTS \"urls_original_url_idx\" ON urls (original_url)",
	)
	if err != nil {
		return dbStorage{}, err
	}

	err = db.ExecWithTimeout(
		context.Background(), 1*time.Second,
		"CREATE INDEX IF NOT EXISTS \"urls_added_by_idx\" ON urls (added_by)",
	)
	if err != nil {
		return dbStorage{}, err
	}

	return db, nil
}

func (db dbStorage) Store(ctx context.Context, u string, by string) (int, error) {
	var id int
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	row := db.handle.QueryRow(
		ctx,
		`
INSERT INTO urls (original_url, added_by)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
RETURNING id`,
		u, by,
	)
	err := row.Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = newError(errNotFound, err)
		}
		return 0, err
	}
	return id, err
}

func generateValuesClause(argsNum int, rowsNum int) string {
	numbers := make([]string, argsNum)
	values := make([]string, rowsNum)
	for r := 0; r < rowsNum; r++ {
		for a := 0; a < argsNum; a++ {
			numbers[a] = "$" + strconv.Itoa(1+r*argsNum+a)
		}
		values[r] = "(" + strings.Join(numbers, ", ") + ")"
	}
	return strings.Join(values, ", ")
}

func (db dbStorage) StoreBatch(ctx context.Context, storeBatchRequest *StoreBatchRequest, by string) (*StoreBatchResponse, error) {
	argsNum := 2
	rowsNum := len(*storeBatchRequest)

	// по-хорошему, задания на вставку надо разбивать на пачки какого-то принятого
	// в проекте максимального размера (1000 работала для меня хорошо), но пока лень)
	sql := fmt.Sprintf(
		"INSERT INTO urls (original_url, added_by) VALUES %s RETURNING id",
		generateValuesClause(argsNum, rowsNum),
	)
	values := make([]interface{}, 0, argsNum*rowsNum)
	for _, r := range *storeBatchRequest {
		values = append(values, r.OriginalURL, by)
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	rows, err := db.handle.Query(ctx, sql, values...)
	cancel()
	if err != nil {
		return nil, err
	}

	answer := make(StoreBatchResponse, 0, rowsNum)
	var id int
	for i := 0; rows.Next(); i++ {
		err := rows.Scan(&id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				err = newError(errNotFound, err)
			}
			return nil, err
		}
		answer = append(answer, storeBatchResponseRecord{(*storeBatchRequest)[i].CorrelationID, id})
	}

	return &answer, nil
}

func (db dbStorage) GetByID(ctx context.Context, id int) (*shortenedURL, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	row := db.handle.QueryRow(
		ctx,
		"SELECT original_url, added_by, is_deleted FROM urls WHERE id = $1",
		id,
	)
	cancel()

	var originalURL, addedBy string
	var isDeleted bool
	err := row.Scan(&originalURL, &addedBy, &isDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = newError(errNotFound, err)
		}
		return nil, err
	}

	return &shortenedURL{id, originalURL, addedBy, isDeleted}, nil
}

func (db dbStorage) GetByIDMulti(ctx context.Context, ids []int) (shortenedURLs, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	rows, err := db.handle.Query(
		ctx,
		"SELECT id, original_url, added_by, is_deleted FROM urls WHERE id = ANY($1)",
		ids,
	)
	cancel()
	if err != nil {
		return nil, err
	}

	answer := make(shortenedURLs, 0, len(ids))
	var id int
	var originalURL, addedBy string
	var isDeleted bool
	for rows.Next() {
		err := rows.Scan(&id, &originalURL, &addedBy, &isDeleted)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				err = newError(errNotFound, err)
			}
			return answer, err
		}
		answer = append(answer, &shortenedURL{id, originalURL, addedBy, isDeleted})
	}

	return answer, nil
}

func (db dbStorage) GetByURL(ctx context.Context, url string) (*shortenedURL, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	row := db.handle.QueryRow(
		ctx,
		"SELECT id, added_by FROM urls WHERE original_url = $1 AND NOT is_deleted",
		url,
	)
	cancel()

	var id int
	var addedBy string
	err := row.Scan(&id, &addedBy)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		err = newError(errNotFound, err)
	}
	if err != nil {
		return nil, err
	}

	return &shortenedURL{id, url, addedBy, false}, nil
}

func (db dbStorage) GetAllUserUrls(ctx context.Context, by string) (shortenedURLs, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	rows, err := db.handle.Query(
		ctx,
		"SELECT id, original_url FROM urls WHERE added_by = $1 AND NOT is_deleted",
		by,
	)
	cancel()
	if err != nil {
		return nil, err
	}

	var answer shortenedURLs
	var id int
	var originalURL string
	for rows.Next() {
		err := rows.Scan(&id, &originalURL)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				err = newError(errNotFound, err)
			}
			return nil, err
		}
		answer = append(answer, &shortenedURL{id, originalURL, by, false})
	}

	return answer, nil
}

func (db dbStorage) DeleteByID(ctx context.Context, id int) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	rows, _ := db.handle.Query(
		ctx,
		"UPDATE urls SET is_deleted = true WHERE id = $1",
		id,
	)
	cancel()

	rows.Close()
	err := rows.Err()
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = newError(errNotFound, err)
		}
	}

	return err
}

func (db dbStorage) Close() error {
	return db.handle.Close(context.Background())
}

func (db dbStorage) ExecWithTimeout(ctx context.Context, timeout time.Duration, sql string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := db.handle.Exec(ctx, sql)
	return err
}
