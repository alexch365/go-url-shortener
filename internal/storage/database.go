package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/util"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var schema = `
	CREATE TABLE IF NOT EXISTS urls (
		id serial PRIMARY KEY,
		short_url TEXT NOT NULL,
		original_url TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS urls_original_url ON urls(original_url);
`

type DatabaseStore struct {
	DB *sql.DB
}

type ConflictError struct {
	ShortURL string
}

func (store *DatabaseStore) Initialize() error {
	var err error
	store.DB, err = sql.Open("pgx", config.Current.DatabaseDSN)
	if err != nil {
		return err
	}

	_, err = store.DB.Exec(schema)
	if err != nil {
		return err
	}

	return nil
}

func (store *DatabaseStore) Save(ctx context.Context, originalURL string) (string, error) {
	shortURL := util.RandomString(8)
	query := `
		INSERT INTO urls (short_url, original_url) VALUES ($1, $2)
		ON CONFLICT (original_url) DO UPDATE
		SET original_url = EXCLUDED.original_url
		RETURNING short_url;
	`
	var existingShortURL string
	err := store.DB.QueryRowContext(ctx, query, shortURL, originalURL).Scan(&existingShortURL)
	if err != nil {
		return "", err
	}

	if existingShortURL != shortURL {
		return "", ConflictError{ShortURL: config.Current.BaseURL + "/" + existingShortURL}
	}
	return config.Current.BaseURL + "/" + shortURL, nil
}

func (store *DatabaseStore) SaveBatch(ctx context.Context, urlStore *[]URLStore) ([]URLStore, error) {
	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var resultURLs []URLStore
	for _, item := range *urlStore {
		item.ShortURL = util.RandomString(8)
		resultItem := item
		resultItem.ShortURL = config.Current.BaseURL + "/" + item.ShortURL
		resultURLs = append(resultURLs, resultItem)

		_, err := tx.ExecContext(ctx, `INSERT INTO urls (short_url, original_url) VALUES ($1, $2)`,
			item.ShortURL, item.OriginalURL)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return resultURLs, nil
}

func (store *DatabaseStore) Get(ctx context.Context, key string) (string, error) {
	query := `SELECT original_url FROM urls WHERE short_url = $1`
	var originalURL string
	err := store.DB.QueryRowContext(ctx, query, key).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("short URL not found: %s", key)
		}
		return "", err
	}
	return originalURL, nil
}

func (store *DatabaseStore) Index(ctx context.Context) ([]URLStore, error) {
	query := `SELECT short_url, original_url FROM urls`
	rows, err := store.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultURLs []URLStore
	for rows.Next() {
		var storeItem URLStore
		err = rows.Scan(&storeItem.ShortURL, &storeItem.OriginalURL)
		storeItem.ShortURL = config.Current.BaseURL + "/" + storeItem.ShortURL
		if err != nil {
			return resultURLs, err
		}
		resultURLs = append(resultURLs, storeItem)
	}
	if err = rows.Err(); err != nil {
		return resultURLs, err
	}
	return resultURLs, nil
}

func (err ConflictError) Error() string {
	return fmt.Sprintf("Original URL already exists with short URL: %s", err.ShortURL)
}
