package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/util"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/sync/errgroup"
)

const (
	bufferSize  = 100
	workerCount = 5
)

var schema = `
	CREATE TABLE IF NOT EXISTS urls (
		id serial PRIMARY KEY,
		short_url TEXT NOT NULL,
		original_url TEXT NOT NULL,
		user_id uuid,
		is_deleted boolean NOT NULL
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
	err := store.DB.QueryRowContext(ctx, query, shortURL, originalURL, config.CurrentUserID).Scan(&existingShortURL)
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

		_, err := tx.ExecContext(ctx, `INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3)`,
			item.ShortURL, item.OriginalURL, config.CurrentUserID)
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

func (store *DatabaseStore) Get(ctx context.Context, key string) (URLStore, error) {
	query := `SELECT original_url, is_deleted FROM urls WHERE short_url = $1`
	var urlStore URLStore
	err := store.DB.QueryRowContext(ctx, query, key).Scan(&urlStore.OriginalURL, &urlStore.DeletedFlag)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return urlStore, fmt.Errorf("short URL not found: %s", key)
		}
		return urlStore, err
	}
	return urlStore, nil
}

func (store *DatabaseStore) Index(ctx context.Context) ([]URLStore, error) {
	query := `SELECT short_url, original_url FROM urls WHERE user_id = $1`
	rows, err := store.DB.QueryContext(ctx, query, config.CurrentUserID)
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

func (store *DatabaseStore) BatchDelete(ctx context.Context, urls []string) error {
	urlsToDeleteCh := urlsToDeleteGen(urls)
	g := new(errgroup.Group)

	for i := 0; i < workerCount; i++ {
		g.Go(func() error {
			var batch []string
			var err error

			for url := range urlsToDeleteCh {
				batch = append(batch, url)

				if len(batch) == bufferSize {
					err = store.processBatchDelete(ctx, batch)
					batch = batch[:0]
				}
			}

			if len(batch) > 0 {
				return store.processBatchDelete(ctx, batch)
			}
			return err
		})
	}

	return g.Wait()
}

func (store *DatabaseStore) processBatchDelete(ctx context.Context, ids []string) error {
	query := `UPDATE urls SET is_deleted = true WHERE short_url = ANY($1) AND user_id = $2`
	_, err := store.DB.ExecContext(ctx, query, ids, config.CurrentUserID)
	return err
}

func urlsToDeleteGen(urls []string) chan string {
	urlsCh := make(chan string, bufferSize)
	go func() {
		defer close(urlsCh)
		for _, url := range urls {
			urlsCh <- url
		}
	}()
	return urlsCh
}

func (err ConflictError) Error() string {
	return fmt.Sprintf("Original URL already exists with short URL: %s", err.ShortURL)
}
