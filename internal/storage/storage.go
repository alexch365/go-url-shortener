package storage

import (
	"encoding/json"
	"github.com/alexch365/go-url-shortener/internal/config"
	"io"
	"os"
)

type URLStore struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var urlStore = map[string]string{}

func Initialize() error {
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
		Set(item.ShortURL, item.OriginalURL)
	}
	return nil
}

func Save(key string, value string) error {
	file, err := os.OpenFile(config.Current.FileStoragePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	Set(key, value)
	encoder := json.NewEncoder(file)
	item := &URLStore{len(urlStore), key, value}
	err = encoder.Encode(item)
	if err != nil {
		return err
	}
	return nil
}

func Get(key string) string {
	return urlStore[key]
}

func Set(key string, value string) {
	urlStore[key] = value
}
