package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/middleware"
	"github.com/alexch365/go-url-shortener/internal/models"
	"github.com/alexch365/go-url-shortener/internal/util"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var schema = `
	CREATE TABLE IF NOT EXISTS urls (
		id serial PRIMARY KEY,
		short_url TEXT NOT NULL,
		original_url TEXT NOT NULL,
		user_id uuid,
		is_deleted boolean NOT NULL default false
	);
	CREATE UNIQUE INDEX IF NOT EXISTS urls_original_url ON urls(original_url, user_id);
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
		INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3)
		ON CONFLICT (original_url, user_id) DO UPDATE
		SET original_url = EXCLUDED.original_url
		RETURNING short_url;
	`
	var existingShortURL string
	err := store.DB.QueryRowContext(ctx, query, shortURL, originalURL, middleware.GetUserID(ctx)).Scan(&existingShortURL)
	if err != nil {
		return "", err
	}

	if existingShortURL != shortURL {
		return "", ConflictError{ShortURL: config.URLFor(existingShortURL)}
	}
	return config.URLFor(shortURL), nil
}

func (store *DatabaseStore) Get(ctx context.Context, key string) (models.URLStore, error) {
	query := `SELECT original_url, is_deleted FROM urls WHERE short_url = $1`
	var urlStore models.URLStore
	err := store.DB.QueryRowContext(ctx, query, key).Scan(&urlStore.OriginalURL, &urlStore.DeletedFlag)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return urlStore, fmt.Errorf("short URL not found: %s", key)
		}
		return urlStore, err
	}
	return urlStore, nil
}

func (store *DatabaseStore) Index(ctx context.Context) ([]models.URLStore, error) {
	query := `SELECT short_url, original_url FROM urls WHERE user_id = $1`
	userID := middleware.GetUserID(ctx)
	rows, err := store.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultURLs []models.URLStore
	for rows.Next() {
		var storeItem models.URLStore
		err = rows.Scan(&storeItem.ShortURL, &storeItem.OriginalURL)
		if err != nil {
			return resultURLs, err
		}
		resultURLs = append(resultURLs, models.URLStore{ShortURL: config.URLFor(storeItem.ShortURL)})
	}
	if err = rows.Err(); err != nil {
		return resultURLs, err
	}
	return resultURLs, nil
}

func (store *DatabaseStore) BatchDelete(ctx context.Context, urls []string) error {
	query := `UPDATE urls SET is_deleted = true WHERE short_url = ANY($1) AND user_id = $2`
	userID := middleware.GetUserID(ctx)
	_, err := store.DB.ExecContext(ctx, query, urls, userID)
	return err
}

func (err ConflictError) Error() string {
	return fmt.Sprintf("Original URL already exists with short URL: %s", err.ShortURL)
}
