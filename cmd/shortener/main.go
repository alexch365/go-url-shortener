package main

import (
	"io"
	"math/rand"
	"net/http"
	"strings"
)

type URLShortener struct {
	urls map[string]string
}

var storage = URLShortener{ urls: make(map[string]string) }

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomString(n int) string {
    sb := strings.Builder{}
    sb.Grow(n)
    for i := 0; i < n; i++ {
        sb.WriteByte(charset[rand.Intn(len(charset))])
    }
    return sb.String()
}

func SaveURLHandler(w http.ResponseWriter, req *http.Request) {
    body, err := io.ReadAll(req.Body)
    if err != nil || len(body) == 0 {
        return
    }

    urlID := randomString(8)
    storage.urls[urlID] = string(body)
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte("http://localhost:8080/" + urlID))
}

func GetURLHandler(w http.ResponseWriter, req *http.Request) {
    urlID := strings.TrimPrefix(req.URL.Path, "/")
    url := storage.urls[urlID]
    if url == "" {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    w.Header().Set("Location", url)
    w.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
    mux := http.NewServeMux()

    mux.HandleFunc(`/{id}`, GetURLHandler)
    mux.HandleFunc("/", SaveURLHandler)

    err := http.ListenAndServe(`:8080`, mux)
    if err != nil {
        panic(err)
    }
}