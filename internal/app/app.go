package app

import (
	"flag"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/handlers"
	"github.com/alexch365/go-url-shortener/internal/logger"
	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func router() chi.Router {
	r := chi.NewRouter()
	r.Use(logger.Middleware)
	r.Use(gzipMiddleware)

	r.Route("/", func(r chi.Router) {
		r.Post("/", handlers.Shorten)
		r.Post("/api/shorten", handlers.ShortenAPI)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", handlers.Expand)
		})
	})
	return r
}

func Run() {
	// flag.Var(&params.ServerAddress, "a", "Server address host:port")
	flag.StringVar(&config.Current.ServerAddress, "a", "", "Server address host:port")
	flag.StringVar(&config.Current.BaseURL, "b", "", "Base for short URL")
	flag.Parse()

	err := env.Parse(&config.Current)
	if err != nil {
		panic(err)
	}

	config.SetDefaults()

	if err := logger.Initialize(); err != nil {
		panic(err)
	}

	err = http.ListenAndServe(config.Current.ServerAddress, router())
	if err != nil {
		panic(err)
	}
}
