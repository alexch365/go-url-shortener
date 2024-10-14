package app

import (
	"context"
	"flag"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/handlers"
	"github.com/alexch365/go-url-shortener/internal/logger"
	"github.com/alexch365/go-url-shortener/internal/storage"
	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"net/http"
)

func router() chi.Router {
	r := chi.NewRouter()
	r.Use(logger.Middleware)
	r.Use(gzipMiddleware)

	r.Route("/", func(r chi.Router) {
		r.Get("/ping", handlers.PingDatabase)
		r.Post("/", handlers.Shorten)
		r.Post("/api/shorten", handlers.ShortenAPI)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", handlers.Expand)
		})
	})
	return r
}

func Run() {
	flag.StringVar(&config.Current.ServerAddress, "a", "", "Server address host:port")
	flag.StringVar(&config.Current.BaseURL, "b", "", "Base for short URL")
	flag.StringVar(&config.Current.FileStoragePath, "r", "", "Base for short URL")
	flag.StringVar(&config.Current.DatabaseDSN, "d", "", "Database source string")
	flag.Parse()

	if err := env.Parse(&config.Current); err != nil {
		panic(err)
	}

	config.SetDefaults()

	if err := logger.Initialize(); err != nil {
		panic(err)
	}
	if err := storage.Initialize(); err != nil {
		panic(err)
	}

	storage.DBConn, _ = pgx.Connect(context.Background(), config.Current.DatabaseDSN)
	defer storage.DBConn.Close(context.Background())

	err := http.ListenAndServe(config.Current.ServerAddress, router())
	if err != nil {
		panic(err)
	}
}
