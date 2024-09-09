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

func TestSaveURLHandler(t *testing.T) {
	type want struct {
		code     int
		response string
	}
	tests := []struct {
		name   string
		reqURL string
		want   want
	}{
		{
			name:   "with created status",
			reqURL: "practicum.yandex.ru",
			want: want{
				code:     201,
				response: "http://localhost:8080/.{8}$",
			},
		},
		{
			name:   "without request body",
			reqURL: "",
			want: want{
				code:     200,
				response: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.reqURL))
			w := httptest.NewRecorder()
			SaveURLHandler(w, request)

			res := w.Result()

			defer res.Body.Close()
			assert.Equal(t, tt.want.code, res.StatusCode)

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Regexp(t, tt.want.response, string(resBody))
		})
	}
}

func TestGetURLHandler(t *testing.T) {
	urlID := randomString(8)
	storage.urls[urlID] = "practicum.yandex.ru"

	type want struct {
		code     int
	}
	tests := []struct {
		name string
		reqID string
		want want
	}{
		{
			name:   "when ID is in storage",
			reqID: urlID,
			want: want{
				code: 307,
			},
		},
		{
			name:   "when ID is not in storage",
			reqID: randomString(8),
			want: want{
				code: 404,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/" + tt.reqID, nil)
			w := httptest.NewRecorder()
			GetURLHandler(w, request)

			res := w.Result()

			defer res.Body.Close()
			assert.Equal(t, tt.want.code, res.StatusCode)

			_, err := io.ReadAll(res.Body)
			require.NoError(t, err)
		})
	}
}
