package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
)

type dbStorage struct {
	handle *pgx.Conn
}

func NewDBStorage(connString string) (*dbStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}
	cancel()
	defer conn.Close(context.Background())

	db := &dbStorage{conn}

	// создадим основную таблицу данных
	// (вообще лучше было бы использовать какую-нибудь систему миграций)
	err = db.ExecWithTimeout(
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
		"CREATE INDEX IF NOT EXISTS ON urls (original_url)",
	)
	if err != nil {
		return nil, err
	}

	err = db.ExecWithTimeout(
		context.Background(), 1*time.Second,
		"CREATE INDEX IF NOT EXISTS ON urls (added_by)",
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
		"INSERT INTO urls (original_url, added_by) VALUES (?, ?) RETURNING id",
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
		"SELECT original_url FROM urls WHERE id = ?",
		id,
	)
	err := row.Scan(&originalURL)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return originalURL, err
}

func (db *dbStorage) GetAllIDsFromUser(by string) ([]int, error) {
	var ids []int
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	row := db.handle.QueryRow(
		ctx,
		"SELECT array_agg(id) FROM urls WHERE added_by = ?",
		by,
	)
	err := row.Scan(&ids)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		err = errors.New(utils.StorageErrNotFound)
	}
	return ids, err
}

func (db *dbStorage) Close() error {
	return db.Close()
}

func (db *dbStorage) ExecWithTimeout(ctx context.Context, timeout time.Duration, sql string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := db.handle.Exec(ctx, sql)
	return err
}
