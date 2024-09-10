package main

import (
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type urlStorage struct {
	urls map[string]string
}

var storage = urlStorage{
    urls: map[string]string{},
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(n int) string {
    sb := strings.Builder{}
    sb.Grow(n)
    for i := 0; i < n; i++ {
        sb.WriteByte(charset[rand.Intn(len(charset))])
    }
    return sb.String()
}

func storeURLHandle(w http.ResponseWriter, req *http.Request) {
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

    urlID := randomString(8)
    storage.urls[urlID] = urlStr
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte("http://localhost:8080/" + urlID))
}

func restoreURLHandle(w http.ResponseWriter, req *http.Request) {
    urlID := chi.URLParam(req, "id")
    url := storage.urls[urlID]
    if url == "" {
        http.Error(w, "The specified ID is not found", http.StatusNotFound)
        return
    }

    middleware.SetHeader("Location", url)
    w.WriteHeader(http.StatusTemporaryRedirect)
}

func router() chi.Router {
    r := chi.NewRouter()
    r.Use(
        middleware.Logger,
        middleware.Recoverer,
    )
    
    r.Route("/", func(r chi.Router) {
        r.Post("/", storeURLHandle)
        r.Route("/{id}", func(r chi.Router) {
            r.Get("/", restoreURLHandle)
        })
    })
    return r
}

func main() {
    err := http.ListenAndServe(`:8080`, router())
    if err != nil {
        panic(err)
    }
}