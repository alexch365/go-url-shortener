package storage

import (
	"context"
	"fmt"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/jackc/pgx/v5"
)

type DatabaseStore struct {
	DBConn *pgx.Conn
}

func (store *DatabaseStore) Initialize() error {
	var err error
	store.DBConn, err = pgx.Connect(context.Background(), config.Current.DatabaseDSN)
	if err != nil {
		return err
	}

	createTable := `
		CREATE TABLE IF NOT EXISTS urls (
    		id serial PRIMARY KEY,
    		short_url TEXT NOT NULL,
    		original_url TEXT NOT NULL
		)
	`
	_, err = store.DBConn.Exec(context.Background(), createTable)
	if err != nil {
		return err
	}

	return nil
}

func (store *DatabaseStore) Save(ctx context.Context, key string, value string) error {
	item := &URLStore{0, key, value}
	query := `INSERT INTO urls (short_url, original_url) VALUES (@short, @original)`
	args := pgx.NamedArgs{
		"short":    item.ShortURL,
		"original": item.OriginalURL,
	}
	fmt.Println(item)
	_, err := store.DBConn.Exec(ctx, query, args)
	if err != nil {
		return err
	}
	return nil
}

func (store *DatabaseStore) Get(ctx context.Context, key string) (string, error) {
	item := &URLStore{0, key, ""}
	query := `
        SELECT original_url FROM urls WHERE short_url = @short
    `
	args := pgx.NamedArgs{
		"short": item.ShortURL,
	}
	row := store.DBConn.QueryRow(ctx, query, args)
	err := row.Scan(&item.OriginalURL)
	if err != nil {
		return "", err
	}
	return item.OriginalURL, nil
}

func (store *DatabaseStore) Close(ctx context.Context) error {
	return store.DBConn.Close(ctx)
}
