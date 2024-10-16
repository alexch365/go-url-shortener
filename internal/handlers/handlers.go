package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/storage"
	"github.com/alexch365/go-url-shortener/internal/util"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var StoreHandler storage.StoreHandler

func PingDatabase(w http.ResponseWriter, r *http.Request) {
	handler, ok := StoreHandler.(*storage.DatabaseStore)
	if !ok {
		http.Error(w, "Database connection failed.", http.StatusInternalServerError)
		return
	}

	if err := handler.DBConn.Ping(r.Context()); err != nil {
		http.Error(w, "Database connection failed.", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func Shorten(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "You must provide a valid URL.", http.StatusBadRequest)
		return
	}

	urlStr := string(body)
	if _, err = url.ParseRequestURI(urlStr); err != nil {
		http.Error(w, fmt.Sprintf("Invalid URL: %s", urlStr), http.StatusBadRequest)
		return
	}

	urlID := util.RandomString(8)
	_ = StoreHandler.Save(req.Context(), urlID, urlStr)
	w.WriteHeader(http.StatusCreated)

	if _, err = w.Write([]byte(config.Current.BaseURL + "/" + urlID)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func ShortenAPI(w http.ResponseWriter, req *http.Request) {
	var urls struct {
		URL string `json:"url"`
	}
	type response struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(req.Body).Decode(&urls); err != nil {
		util.JSONError(w, response{err.Error()}, http.StatusBadRequest)
		return
	}

	if _, err := url.ParseRequestURI(urls.URL); err != nil {
		util.JSONError(w, response{err.Error()}, http.StatusBadRequest)
		return
	}

	urlID := util.RandomString(8)
	_ = StoreHandler.Save(req.Context(), urlID, urls.URL)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err := json.NewEncoder(w).Encode(response{config.Current.BaseURL + "/" + urlID})
	if err != nil {
		util.JSONError(w, response{err.Error()}, http.StatusBadRequest)
		return
	}
}

func Expand(w http.ResponseWriter, req *http.Request) {
	urlID := strings.TrimPrefix(req.URL.Path, "/")
	storedURL, err := StoreHandler.Get(req.Context(), urlID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid ID: %s", urlID), http.StatusNotFound)
		return
	}

	w.Header().Set("Location", storedURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
