package storage

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/util"
	"io"
	"os"
)

type (
	StoreHandler interface {
		Initialize() error
		Get(ctx context.Context, key string) (URLStore, error)
		Save(ctx context.Context, originalURL string) (string, error)
		SaveBatch(ctx context.Context, store *[]URLStore) ([]URLStore, error)
		Index(ctx context.Context) ([]URLStore, error)
		BatchDelete(ctx context.Context, urls []string) error
	}
	URLStore struct {
		UUID          int    `json:"uuid,omitempty" db:"-"`
		CorrelationID string `json:"correlation_id,omitempty" db:"-"`
		ShortURL      string `json:"short_url" db:"short_url"`
		OriginalURL   string `json:"original_url" db:"original_url"`
		DeletedFlag   bool   `json:"-" db:"is_deleted"`
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

func (store *MemoryStore) Save(_ context.Context, originalURL string) (string, error) {
	file, err := os.OpenFile(config.Current.FileStoragePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return "", err
	}
	defer file.Close()

	urlStore := URLStore{UUID: len(store.urls), ShortURL: util.RandomString(8), OriginalURL: originalURL}
	store.urls = append(store.urls, urlStore)

	err = json.NewEncoder(file).Encode(urlStore)
	if err != nil {
		return "", err
	}
	return config.Current.BaseURL + "/" + urlStore.ShortURL, nil
}

func (store *MemoryStore) SaveBatch(_ context.Context, urlStore *[]URLStore) ([]URLStore, error) {
	file, err := os.OpenFile(config.Current.FileStoragePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	var resultURLs []URLStore
	for _, item := range *urlStore {
		item.ShortURL = util.RandomString(8)
		store.urls = append(store.urls, item)
		resultItem := item
		resultItem.ShortURL = config.Current.BaseURL + "/" + item.ShortURL
		resultURLs = append(resultURLs, resultItem)

		if err = encoder.Encode(item); err != nil {
			return resultURLs, err
		}
	}

	return resultURLs, nil
}

func (store *MemoryStore) Get(_ context.Context, key string) (URLStore, error) {
	for i := range store.urls {
		if store.urls[i].ShortURL == key {
			return store.urls[i], nil
		}
	}
	return URLStore{}, errors.New("key not found")
}

func (store *MemoryStore) Index(_ context.Context) ([]URLStore, error) {
	result := store.urls
	for _, item := range result {
		item.ShortURL = config.Current.BaseURL + "/" + item.ShortURL
	}
	return result, nil
}

func (store *MemoryStore) BatchDelete(_ context.Context, _ []string) error {
	return nil
}
