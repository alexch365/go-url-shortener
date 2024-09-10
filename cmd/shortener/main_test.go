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

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestStoreURLHandle(t *testing.T) {
	ts := httptest.NewServer(router())
    defer ts.Close()

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
			resp, body := testRequest(t, ts, "POST", "/", tt.body)
			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Regexp(t, tt.want, body)
		})
    }
}

func TestRestoreURLHandle(t *testing.T) {
	ts := httptest.NewServer(router())
    defer ts.Close()

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
			resp, _ := testRequest(t, ts, "GET", "/"+tt.id, "")
			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}
