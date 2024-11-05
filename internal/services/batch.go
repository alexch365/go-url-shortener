package services

import (
	"context"
	"github.com/alexch365/go-url-shortener/internal/models"
	"github.com/alexch365/go-url-shortener/internal/storage"
	"github.com/alexch365/go-url-shortener/internal/util"
	"golang.org/x/sync/errgroup"
)

const (
	batchSize   = 100
	workerCount = 5
)

func BatchCreate(store storage.StoreHandler, ctx context.Context, data *[]models.URLStore) ([]models.URLStore, error) {
	var resultURLs []models.URLStore
	for _, item := range *data {
		_, err := util.ParseURL(item.OriginalURL)
		if err != nil {
			return resultURLs, err
		}
		resultItem := item
		resultItem.ShortURL, err = store.Save(ctx, item.OriginalURL)
		if err != nil {
			return resultURLs, err
		}
		resultURLs = append(resultURLs, resultItem)
	}
	return resultURLs, nil
}

func BatchDelete(store storage.StoreHandler, ctx context.Context, data []string) error {
	dataCh := batchesChannel(data)
	g := new(errgroup.Group)

	for i := 0; i < workerCount; i++ {
		g.Go(func() error {
			var batch []string
			var err error

			for url := range dataCh {
				batch = append(batch, url)

				if len(batch) == batchSize {
					err = store.BatchDelete(ctx, batch)
					batch = batch[:0]
				}
			}

			if len(batch) > 0 {
				return store.BatchDelete(ctx, batch)
			}
			return err
		})
	}

	return g.Wait()
}

func batchesChannel(data []string) chan string {
	channel := make(chan string, batchSize)
	go func() {
		defer close(channel)
		for _, item := range data {
			channel <- item
		}
	}()
	return channel
}
