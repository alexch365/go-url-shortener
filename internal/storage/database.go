package storage

import (
	"context"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var schema = `
	CREATE TABLE IF NOT EXISTS urls (
		id serial PRIMARY KEY,
		short_url TEXT NOT NULL,
		original_url TEXT NOT NULL
	)
`

type DatabaseStore struct {
	DB *sqlx.DB
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

func (store *DatabaseStore) Save(ctx context.Context, urlStore *URLStore) error {
	tx, err := store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareNamed(`INSERT INTO urls (short_url, original_url) VALUES (:short_url, :original_url)`)
	if err != nil {
		return err
	}

	if _, err = stmt.Exec(urlStore); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (store *DatabaseStore) SaveBatch(ctx context.Context, urlStore *[]URLStore) error {
	tx, err := store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.NamedExec(
		`INSERT INTO urls (short_url, original_url) VALUES (:short_url, :original_url)`,
		*urlStore,
	)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
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
