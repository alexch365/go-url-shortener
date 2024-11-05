package storage

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/models"
	"github.com/alexch365/go-url-shortener/internal/util"
	"io"
	"net/http"
	"os"
)

type (
	StoreHandler interface {
		Initialize() error
		Get(ctx context.Context, key string) (models.URLStore, error)
		Save(ctx context.Context, originalURL string) (string, error)
		Index(ctx context.Context) ([]models.URLStore, error)
		BatchDelete(ctx context.Context, urls []string) error
	}
	MemoryStore struct {
		urls []models.URLStore
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
		var item models.URLStore
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

	urlStore := models.URLStore{UUID: len(store.urls), ShortURL: util.RandomString(8), OriginalURL: originalURL}
	store.urls = append(store.urls, urlStore)

	err = json.NewEncoder(file).Encode(urlStore)
	if err != nil {
		return "", err
	}
	return config.URLFor(urlStore.ShortURL), nil
}

func (store *MemoryStore) Get(_ context.Context, key string) (models.URLStore, error) {
	for i := range store.urls {
		if store.urls[i].ShortURL == key {
			return store.urls[i], nil
		}
	}
	return models.URLStore{}, errors.New("key not found")
}

func (store *MemoryStore) Index(_ context.Context) ([]models.URLStore, error) {
	result := store.urls
	for _, item := range result {
		item.ShortURL = config.Current.BaseURL + "/" + item.ShortURL
	}
	return result, nil
}

func (store *MemoryStore) BatchDelete(_ context.Context, _ []string) error {
	return nil
}

func CheckConflict(err error) (string, int) {
	if errors.As(err, &ConflictError{}) {
		return err.(ConflictError).ShortURL, http.StatusConflict
	} else {
		return err.Error(), http.StatusInternalServerError
	}
}
