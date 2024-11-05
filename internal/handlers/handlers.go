package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/alexch365/go-url-shortener/internal/models"
	"github.com/alexch365/go-url-shortener/internal/services"
	"github.com/alexch365/go-url-shortener/internal/storage"
	"github.com/alexch365/go-url-shortener/internal/util"
	"io"
	"net/http"
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
	bodyData, err := io.ReadAll(req.Body)
	if err != nil || len(bodyData) == 0 {
		http.Error(w, "empty or invalid body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	bodyURL := string(bodyData)
	_, err = util.ParseURL(bodyURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status := http.StatusCreated
	result, err := StoreHandler.Save(req.Context(), bodyURL)
	if err != nil {
		result, status = storage.CheckConflict(err)
	}

	w.WriteHeader(status)
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

	_, err := util.ParseURL(requestJSON.URL)
	if err != nil {
		util.JSONResponse(w, apiResponse{Error: err.Error()}, http.StatusBadRequest)
		return
	}

	status := http.StatusCreated
	result, err := StoreHandler.Save(req.Context(), requestJSON.URL)
	if err != nil {
		result, status = storage.CheckConflict(err)
		if status != http.StatusConflict {
			util.JSONResponse(w, apiResponse{Error: result}, status)
			return
		}
	}

	util.JSONResponse(w, apiResponse{Result: result}, status)
}

func ShortenAPIBatch(w http.ResponseWriter, req *http.Request) {
	var store []models.URLStore
	if err := json.NewDecoder(req.Body).Decode(&store); err != nil {
		util.JSONResponse(w, apiResponse{Error: "Invalid request format."}, http.StatusBadRequest)
		return
	}

	responseStore, err := services.BatchCreate(StoreHandler, req.Context(), &store)
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
		status = http.StatusUnauthorized
	}
	util.JSONResponse(w, urls, status)
}

func APIDeleteUserURLs(w http.ResponseWriter, req *http.Request) {
	var ids []string
	if err := json.NewDecoder(req.Body).Decode(&ids); err != nil {
		util.JSONResponse(w, apiResponse{Error: "Invalid request format."}, http.StatusBadRequest)
		return
	}
	err := services.BatchDelete(StoreHandler, req.Context(), ids)
	if err != nil {
		util.JSONResponse(w, apiResponse{Error: err.Error()}, http.StatusBadRequest)
	}
	util.JSONResponse(w, nil, http.StatusAccepted)
}

func Expand(w http.ResponseWriter, req *http.Request) {
	urlID := strings.TrimPrefix(req.URL.Path, "/")
	urlStore, err := StoreHandler.Get(req.Context(), urlID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid ID: %s", urlID), http.StatusNotFound)
		return
	}
	if urlStore.DeletedFlag {
		w.WriteHeader(http.StatusGone)
	} else {
		w.Header().Set("Location", urlStore.OriginalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}
