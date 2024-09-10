package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreURLHandle(t *testing.T) {
	setDefaults()
	tests := []struct {
		name string
		body   string
        want   string
        status int
    }{
		{ "with valid URL", "https://practicum.yandex.ru", "http://localhost:8080/.{8}$", http.StatusCreated },
		{ "with invalid URL", "https//practicum.yandex.ru", "", http.StatusBadRequest },
		{ "with empty URL", "", "", http.StatusBadRequest },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			storeURLHandle(rec, request)
			resp := rec.Result()
			defer resp.Body.Close()

			resBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Regexp(t, tt.want, string(resBody))
		})
    }
}

func TestRestoreURLHandle(t *testing.T) {
	urlID := randomString(8)
	storage.urls[urlID] = "https://practicum.yandex.ru"

	tests := []struct {
		name string
		id string
		status int
	}{
		{ "with stored ID", urlID, http.StatusTemporaryRedirect },
		{ "with random ID", randomString(8), http.StatusNotFound },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)
			rec := httptest.NewRecorder()
			restoreURLHandle(rec, request)
			resp := rec.Result()
			defer resp.Body.Close()

			_, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}
