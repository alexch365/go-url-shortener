package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/storage"
	"github.com/alexch365/go-url-shortener/internal/util"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type (
	apiRequest struct {
		URL string `json:"url"`
	}
	apiResponse struct {
		Result string `json:"result,omitempty"`
		Error  string `json:"error,omitempty"`
	}
	batchAPIResponse struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}
)

var StoreHandler storage.StoreHandler

func PingDatabase(w http.ResponseWriter, r *http.Request) {
	handler, ok := StoreHandler.(*storage.DatabaseStore)
	if !ok {
		http.Error(w, "Database connection failed.", http.StatusInternalServerError)
		return
	}

	if err := handler.DB.PingContext(r.Context()); err != nil {
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

	store := storage.URLStore{ShortURL: util.RandomString(8), OriginalURL: string(body)}
	if _, err = url.ParseRequestURI(store.OriginalURL); err != nil {
		http.Error(w, fmt.Sprintf("Invalid URL: %s", store.OriginalURL), http.StatusBadRequest)
		return
	}

	err = StoreHandler.Save(req.Context(), &store)
	if errors.As(err, &storage.ConflictError{}) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(config.Current.BaseURL + "/" + err.(storage.ConflictError).ShortURL))
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if _, err = w.Write([]byte(config.Current.BaseURL + "/" + store.ShortURL)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func ShortenAPI(w http.ResponseWriter, req *http.Request) {
	var requestJSON apiRequest

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewDecoder(req.Body).Decode(&requestJSON); err != nil {
		util.JSONError(w, apiResponse{Error: "Invalid request format."}, http.StatusBadRequest)
		return
	}

	if _, err := url.ParseRequestURI(requestJSON.URL); err != nil {
		response := apiResponse{Error: fmt.Sprintf("Invalid URL: %s", requestJSON.URL)}
		util.JSONError(w, response, http.StatusBadRequest)
		return
	}

	store := storage.URLStore{ShortURL: util.RandomString(8), OriginalURL: requestJSON.URL}
	err := StoreHandler.Save(req.Context(), &store)
	if errors.As(err, &storage.ConflictError{}) {
		w.WriteHeader(http.StatusConflict)
		response := apiResponse{Result: config.Current.BaseURL + "/" + err.(storage.ConflictError).ShortURL}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err != nil {
		util.JSONError(w, apiResponse{Error: err.Error()}, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(apiResponse{Result: config.Current.BaseURL + "/" + store.ShortURL})
	if err != nil {
		util.JSONError(w, apiResponse{Error: err.Error()}, http.StatusBadRequest)
	}
}

func ShortenAPIBatch(w http.ResponseWriter, req *http.Request) {
	var store []storage.URLStore
	if err := json.NewDecoder(req.Body).Decode(&store); err != nil {
		util.JSONError(w, apiResponse{Error: "Invalid request format."}, http.StatusBadRequest)
		return
	}

	var responseStore []batchAPIResponse
	for i, item := range store {
		if _, err := url.ParseRequestURI(item.OriginalURL); err != nil {
			response := apiResponse{Error: fmt.Sprintf("Invalid URL: %s", item.OriginalURL)}
			util.JSONError(w, response, http.StatusBadRequest)
			return
		}

		store[i].ShortURL = util.RandomString(8)
		responseItem := batchAPIResponse{
			item.CorrelationID,
			config.Current.BaseURL + "/" + store[i].ShortURL,
		}
		responseStore = append(responseStore, responseItem)
	}

	if err := StoreHandler.SaveBatch(req.Context(), &store); err != nil {
		util.JSONError(w, apiResponse{Error: err.Error()}, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err := json.NewEncoder(w).Encode(responseStore)
	if err != nil {
		util.JSONError(w, apiResponse{Error: err.Error()}, http.StatusBadRequest)
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
