package storage

import (
	"context"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/util"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
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
	DB *sqlx.DB
}

type ConflictError struct {
	ShortURL string `db:"short_url"`
}

func (store *DatabaseStore) Initialize() error {
	var err error
	store.DB, err = sqlx.Connect("pgx", config.Current.DatabaseDSN)
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
	urlStore := URLStore{ShortURL: util.RandomString(8), OriginalURL: originalURL}

	stmt, err := store.DB.PrepareNamedContext(ctx, `
		INSERT INTO urls (short_url, original_url) VALUES (:short_url, :original_url)
		ON CONFLICT (original_url) DO UPDATE
		SET original_url = EXCLUDED.original_url
		RETURNING short_url;
	`)
	if err != nil {
		return "", err
	}

	var conflictErr ConflictError
	err = stmt.QueryRowx(&urlStore).Scan(&conflictErr.ShortURL)
	if err != nil {
		return "", err
	}

	if conflictErr.ShortURL != urlStore.ShortURL {
		return config.Current.BaseURL + "/" + conflictErr.ShortURL, conflictErr
	}
	return config.Current.BaseURL + "/" + urlStore.ShortURL, nil
}

func (store *DatabaseStore) SaveBatch(ctx context.Context, urlStore *[]URLStore) ([]URLStore, error) {
	tx, err := store.DB.BeginTxx(ctx, nil)
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
	}

	_, err = tx.NamedExec(
		`INSERT INTO urls (short_url, original_url) VALUES (:short_url, :original_url)`,
		*urlStore,
	)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return resultURLs, nil
}

func (store *DatabaseStore) Get(ctx context.Context, key string) (string, error) {
	item := URLStore{ShortURL: key, OriginalURL: ""}
	nstmt, _ := store.DB.PrepareNamedContext(ctx, `
        SELECT original_url FROM urls WHERE short_url = :short_url
    `)
	err := nstmt.Get(&item.OriginalURL, item)
	if err != nil {
		return "", err
	}
	return item.OriginalURL, nil
}

func (err ConflictError) Error() string {
	return "Original URL already exists"
}
