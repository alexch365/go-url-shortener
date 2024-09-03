package main

import (
    "io"
    "net/http"
	"math/rand"
	"strings"
)

type URLShortener struct {
	urls map[string]string
}

var storage = URLShortener{}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomString(n int) string {
    sb := strings.Builder{}
    sb.Grow(n)
    for i := 0; i < n; i++ {
        sb.WriteByte(charset[rand.Intn(len(charset))])
    }
    return sb.String()
}

func mainPage(w http.ResponseWriter, req *http.Request) {
    if req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return
		}

		urlID := randomString(8)
		storage.urls[urlID] = string(body)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("http://localhost:8080/" + urlID))
    } else {
		urlID := req.PathValue("id")
		url := storage.urls[urlID]
		if url == "" {
			return
		}

        w.Header().Set("Location", url)
		w.WriteHeader(http.StatusTemporaryRedirect)
    }
}

func main() {
    mux := http.NewServeMux()
	storage.urls = make(map[string]string)

    mux.HandleFunc(`/{id}`, mainPage)
    mux.HandleFunc("/", mainPage)

    err := http.ListenAndServe(`:8080`, mux)
    if err != nil {
        panic(err)
    }
}