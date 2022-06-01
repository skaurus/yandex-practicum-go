package storage

import (
	"context"
	"errors"
	"time"

	"github.com/skaurus/yandex-practicum-go/internal/env"

	"github.com/jackc/pgx/v4"
)

type dbStorage struct {
	handle *pgx.Conn
}

func NewDBStorage(env *env.Environment) (*dbStorage, error) {
	db := &dbStorage{env.DBConn}

	// создадим основную таблицу данных
	// (вообще лучше было бы использовать какую-нибудь систему миграций)
	err := db.ExecWithTimeout(
		context.Background(), 1*time.Second, `
CREATE TABLE IF NOT EXISTS urls (
	id			 serial PRIMARY KEY,
	original_url text NOT NULL,
	added_by	 text NOT NULL
)`,
	)
	if err != nil {
		return nil, err
	}

	err = db.ExecWithTimeout(
		context.Background(), 1*time.Second,
		"CREATE INDEX IF NOT EXISTS \"urls_original_url_idx\" ON urls (original_url)",
	)
	if err != nil {
		return nil, err
	}

	err = db.ExecWithTimeout(
		context.Background(), 1*time.Second,
		"CREATE INDEX IF NOT EXISTS \"urls_added_by_idx\" ON urls (added_by)",
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *dbStorage) Store(u string, by string) (int, error) {
	var id int
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	row := db.handle.QueryRow(
		ctx,
		"INSERT INTO urls (original_url, added_by) VALUES ($1, $2) RETURNING id",
		u, by,
	)
	err := row.Scan(&id)
	return id, err
}

func (db *dbStorage) GetByID(id int) (string, error) {
	var originalURL string
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	row := db.handle.QueryRow(
		ctx,
		"SELECT original_url FROM urls WHERE id = $1",
		id,
	)
	err := row.Scan(&originalURL)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return originalURL, err
}

func (db *dbStorage) GetAllUserUrls(by string) (shortenedURLs, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	rows, err := db.handle.Query(
		ctx,
		"SELECT id, original_url FROM urls WHERE added_by = $1",
		by,
	)
	cancel()

	var answer shortenedURLs
	var id int
	var originalURL string
	for rows.Next() {
		err := rows.Scan(&id, &originalURL)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				err = ErrNotFound
			}
			return nil, err
		}
		answer = append(answer, shortenedURL{id, originalURL, by})
	}

	return answer, err
}

func (db *dbStorage) Close() error {
	return db.handle.Close(context.Background())
}

func (db *dbStorage) ExecWithTimeout(ctx context.Context, timeout time.Duration, sql string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := db.handle.Exec(ctx, sql)
	return err
}
