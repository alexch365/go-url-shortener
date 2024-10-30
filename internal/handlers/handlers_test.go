package handlers

import (
	"context"
	"encoding/json"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/storage"
	"github.com/alexch365/go-url-shortener/internal/util"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShorten(t *testing.T) {
	config.SetDefaults()
	StoreHandler = &storage.MemoryStore{}
	tests := []struct {
		name   string
		body   string
		want   string
		status int
	}{
		{"with valid URL", "https://practicum.yandex.ru", "http://localhost:8080/.{8}$", http.StatusCreated},
		{"with invalid URL", "https//practicum.yandex.ru", "", http.StatusBadRequest},
		{"with empty URL", "", "", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			Shorten(rec, request)
			resp := rec.Result()
			defer resp.Body.Close()

			resBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Regexp(t, tt.want, string(resBody))
		})
	}
}

func TestShortenAPI(t *testing.T) {
	config.SetDefaults()
	StoreHandler = &storage.MemoryStore{}

	tests := []struct {
		name   string
		body   string
		want   apiResponse
		status int
	}{
		{
			"with valid URL",
			`{"url": "https://practicum.yandex.ru"}`,
			apiResponse{Result: "http://localhost:8080/.{8}$"},
			http.StatusCreated,
		},
		{
			"with invalid URL",
			`{"url": "https//practicum.yandex.ru"}`,
			apiResponse{Error: "Invalid URL: .*"},
			http.StatusBadRequest,
		},
		{
			"with incorrect JSON key",
			`{"uri": "https://practicum.yandex.ru"}`,
			apiResponse{Error: "Invalid URL: .*"},
			http.StatusBadRequest,
		},
		{
			"with string request",
			"https://practicum.yandex.ru",
			apiResponse{Error: "Invalid request format."},
			http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			ShortenAPI(rec, request)
			resp := rec.Result()
			defer resp.Body.Close()

			var resBody apiResponse
			err := json.NewDecoder(resp.Body).Decode(&resBody)
			require.NoError(t, err)

			assert.Equal(t, tt.status, resp.StatusCode)
			if tt.want.Error != "" {
				assert.Regexp(t, tt.want.Error, resBody.Error)
			}
			if tt.want.Result != "" {
				assert.Regexp(t, tt.want.Result, resBody.Result)
			}
		})
	}
}

func TestShortenAPIBatch(t *testing.T) {
	config.SetDefaults()
	StoreHandler = &storage.MemoryStore{}

	tests := []struct {
		name     string
		body     string
		response []storage.URLStore
		error    apiResponse
		status   int
	}{
		{
			"with valid URL",
			`[
					{
					  "correlation_id": "30d53d47-6d08-41ce-992f-097b0f01479b",
					  "original_url": "https://practicum.yandex.ru"
					},
					{
					  "correlation_id": "c65f7a7b-770d-4a59-97d2-946bcdfa2589",
					  "original_url": "https://ya.ru"
					}
				]`,
			[]storage.URLStore{
				{CorrelationID: "30d53d47-6d08-41ce-992f-097b0f01479b", ShortURL: "http://localhost:8080/.{8}$"},
				{CorrelationID: "c65f7a7b-770d-4a59-97d2-946bcdfa2589", ShortURL: "http://localhost:8080/.{8}$"},
			},
			apiResponse{},
			http.StatusCreated,
		},
		{
			"with invalid URL",
			`[
					{
					  "correlation_id": "30d53d47-6d08-41ce-992f-097b0f01479b",
					  "original_url": "https//practicum.yandex.ru"
					}
				]`,
			[]storage.URLStore{},
			apiResponse{Error: "Invalid URL: .*"},
			http.StatusBadRequest,
		},
		{
			"with incorrect JSON key",
			`[{"uri": "https://practicum.yandex.ru"}]`,
			[]storage.URLStore{},
			apiResponse{Error: "Invalid URL: .*"},
			http.StatusBadRequest,
		},
		{
			"with string request",
			"https://practicum.yandex.ru",
			[]storage.URLStore{},
			apiResponse{Error: "Invalid request format."},
			http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			ShortenAPIBatch(rec, request)
			resp := rec.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.status, resp.StatusCode)
			switch resp.StatusCode {
			case http.StatusCreated:
				var resBody []storage.URLStore
				err := json.NewDecoder(resp.Body).Decode(&resBody)
				require.NoError(t, err)
				for i, res := range tt.response {
					assert.Equal(t, res.CorrelationID, resBody[i].CorrelationID)
					assert.Regexp(t, res.ShortURL, resBody[i].ShortURL)
				}
			case http.StatusBadRequest:
				var resBody apiResponse
				err := json.NewDecoder(resp.Body).Decode(&resBody)
				require.NoError(t, err)
				assert.Regexp(t, tt.error.Error, resBody.Error)
			}
		})
	}
}

func TestAPIUserURLs(t *testing.T) {
	config.SetDefaults()
	StoreHandler = &storage.MemoryStore{}
	store := []storage.URLStore{
		{OriginalURL: "https://practicum.yandex.ru", ShortURL: util.RandomString(8)},
		{OriginalURL: "https://ya.ru", ShortURL: util.RandomString(8)},
	}
	_, _ = StoreHandler.SaveBatch(context.TODO(), &store)
	tests := []struct {
		name     string
		response []storage.URLStore
		error    apiResponse
		status   int
	}{
		{
			"",
			store,
			apiResponse{},
			http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/user/urls", strings.NewReader(""))
			rec := httptest.NewRecorder()
			APIUserURLs(rec, request)
			resp := rec.Result()
			defer resp.Body.Close()

			err := json.NewDecoder(resp.Body).Decode(&tt.response)
			require.NoError(t, err)
			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

func TestExpand(t *testing.T) {
	config.SetDefaults()
	StoreHandler = &storage.MemoryStore{}
	result, _ := StoreHandler.Save(context.TODO(), "https://practicum.yandex.ru")
	urlParts := strings.Split(result, "/")

	tests := []struct {
		name   string
		id     string
		status int
	}{
		{"with stored ID", urlParts[len(urlParts)-1], http.StatusTemporaryRedirect},
		{"with random ID", util.RandomString(8), http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)
			rec := httptest.NewRecorder()
			Expand(rec, request)
			resp := rec.Result()
			defer resp.Body.Close()

			_, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}
