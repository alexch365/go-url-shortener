package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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
	bodyURL, err := parseURLFromBody(req.Body)
	if err != nil {
		http.Error(w, "You must provide a valid URL.", http.StatusBadRequest)
		return
	}

	result, err := StoreHandler.Save(req.Context(), bodyURL)
	if err != nil {
		if errors.As(err, &storage.ConflictError{}) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(err.(storage.ConflictError).ShortURL))
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(result))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func ShortenAPI(w http.ResponseWriter, req *http.Request) {
	var requestJSON apiRequest
	if err := json.NewDecoder(req.Body).Decode(&requestJSON); err != nil {
		util.JSONResponse(w, apiResponse{Error: "Invalid request format."}, http.StatusBadRequest)
		return
	}

	if _, err := url.ParseRequestURI(requestJSON.URL); err != nil {
		response := apiResponse{Error: fmt.Sprintf("Invalid URL: %s", requestJSON.URL)}
		util.JSONResponse(w, response, http.StatusBadRequest)
		return
	}

	shortURL, err := StoreHandler.Save(req.Context(), requestJSON.URL)
	if err != nil {
		if errors.As(err, &storage.ConflictError{}) {
			util.JSONResponse(w, apiResponse{Result: err.(storage.ConflictError).ShortURL}, http.StatusConflict)
		} else {
			util.JSONResponse(w, apiResponse{Error: err.Error()}, http.StatusInternalServerError)
		}
		return
	}

	util.JSONResponse(w, apiResponse{Result: shortURL}, http.StatusCreated)
}

func ShortenAPIBatch(w http.ResponseWriter, req *http.Request) {
	var store []storage.URLStore
	if err := json.NewDecoder(req.Body).Decode(&store); err != nil {
		util.JSONResponse(w, apiResponse{Error: "Invalid request format."}, http.StatusBadRequest)
		return
	}

	for _, item := range store {
		if _, err := url.ParseRequestURI(item.OriginalURL); err != nil {
			response := apiResponse{Error: fmt.Sprintf("Invalid URL: %s", item.OriginalURL)}
			util.JSONResponse(w, response, http.StatusBadRequest)
			return
		}
	}

	responseStore, err := StoreHandler.SaveBatch(req.Context(), &store)
	if err != nil {
		util.JSONResponse(w, apiResponse{Error: err.Error()}, http.StatusBadRequest)
		return
	}

	util.JSONResponse(w, responseStore, http.StatusCreated)
}

func APIUserURLs(w http.ResponseWriter, req *http.Request) {
	urls, err := StoreHandler.Index(req.Context())
	if err != nil {
		util.JSONResponse(w, apiResponse{Error: err.Error()}, http.StatusBadRequest)
		return
	}
	status := http.StatusOK
	if len(urls) == 0 {
		status = http.StatusNoContent
	}
	util.JSONResponse(w, urls, status)
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

func parseURLFromBody(body io.ReadCloser) (string, error) {
	defer body.Close()
	bodyData, err := io.ReadAll(body)
	if err != nil || len(bodyData) == 0 {
		return "", errors.New("empty or invalid body")
	}
	urlStr := string(bodyData)
	if _, err := url.ParseRequestURI(urlStr); err != nil {
		return "", fmt.Errorf("invalid URL: %s", urlStr)
	}
	return urlStr, nil
}
