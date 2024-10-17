package storage

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/alexch365/go-url-shortener/internal/config"
	"io"
	"os"
)

type (
	StoreHandler interface {
		Initialize() error
		Get(ctx context.Context, key string) (string, error)
		Save(ctx context.Context, store *URLStore) error
		SaveBatch(ctx context.Context, store *[]URLStore) error
	}
	URLStore struct {
		UUID          int    `json:"uuid,omitempty" db:"-"`
		CorrelationID string `json:"correlation_id,omitempty" db:"-"`
		ShortURL      string `json:"short_url" db:"short_url"`
		OriginalURL   string `json:"original_url" db:"original_url"`
	}
	MemoryStore struct {
		urls []URLStore
	}
)

func (store *MemoryStore) Initialize() error {
	file, err := os.OpenFile(config.Current.FileStoragePath, os.O_RDONLY, 0666)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		var item URLStore
		if err := decoder.Decode(&item); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		store.urls = append(store.urls, item)
	}
	return nil
}

func (store *MemoryStore) Save(_ context.Context, urlStore *URLStore) error {
	file, err := os.OpenFile(config.Current.FileStoragePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	urlStore.UUID = len(store.urls)
	store.urls = append(store.urls, *urlStore)

	err = json.NewEncoder(file).Encode(urlStore)
	if err != nil {
		return err
	}
	return nil
}

func (store *MemoryStore) SaveBatch(_ context.Context, urlStore *[]URLStore) error {
	file, err := os.OpenFile(config.Current.FileStoragePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	store.urls = append(store.urls, *urlStore...)
	encoder := json.NewEncoder(file)
	for _, item := range *urlStore {
		err = encoder.Encode(item)
		if err != nil {
			return err
		}
	}

	return nil
}

func (store *MemoryStore) Get(_ context.Context, key string) (string, error) {
	for i := range store.urls {
		if store.urls[i].ShortURL == key {
			return store.urls[i].OriginalURL, nil
		}
	}
	return "", errors.New("key not found")
}
