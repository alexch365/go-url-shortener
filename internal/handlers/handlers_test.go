package handlers

import (
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
	type want struct {
		Result string `json:"result"`
	}
	tests := []struct {
		name   string
		body   string
		want   want
		status int
	}{
		{
			"with valid URL",
			`{"url": "https://practicum.yandex.ru"}`,
			want{"http://localhost:8080/.{8}$"},
			http.StatusCreated,
		},
		{
			"with invalid URL",
			`{"url": "https//practicum.yandex.ru"}`,
			want{"parse .*: invalid URI for request"},
			http.StatusBadRequest,
		},
		{
			"with incorrect JSON key",
			`{"uri": "https//practicum.yandex.ru"}`,
			want{"parse .*: empty url"},
			http.StatusBadRequest,
		},
		{
			"with string request",
			"https://practicum.yandex.ru",
			want{"invalid character.*"},
			http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			ShortenAPI(rec, request)
			resp := rec.Result()
			defer resp.Body.Close()

			var resBody want
			err := json.NewDecoder(resp.Body).Decode(&resBody)
			require.NoError(t, err)

			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Regexp(t, tt.want.Result, resBody.Result)
		})
	}
}

func TestExpand(t *testing.T) {
	urlID := util.RandomString(8)
	StoreHandler = &storage.MemoryStore{}
	_ = StoreHandler.Save(nil, urlID, "https://practicum.yandex.ru")

	tests := []struct {
		name   string
		id     string
		status int
	}{
		{"with stored ID", urlID, http.StatusTemporaryRedirect},
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
