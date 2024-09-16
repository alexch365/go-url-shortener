package handlers

import (
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/storage"
	"github.com/alexch365/go-url-shortener/internal/util"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func Shorten(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "You must provide a valid URL.", http.StatusBadRequest)
		return
	}

	urlStr := string(body)
	_, err = url.ParseRequestURI(urlStr)
	if err != nil {
		http.Error(w, "The specified URL is not valid", http.StatusBadRequest)
		return
	}

	urlID := util.RandomString(8)
	storage.Save(urlID, urlStr)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(config.Current.BaseURL + "/" + urlID))
	if err != nil {
		return
	}
}

func Expand(w http.ResponseWriter, req *http.Request) {
	urlID := strings.TrimPrefix(req.URL.Path, "/")
	storedURL := storage.Get(urlID)
	if storedURL == "" {
		http.Error(w, "The specified ID is not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Location", storedURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
