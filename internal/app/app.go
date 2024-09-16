package app

import (
	"flag"
	"github.com/alexch365/go-url-shortener/internal/config"
	"github.com/alexch365/go-url-shortener/internal/handlers"
	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

func router() chi.Router {
	r := chi.NewRouter()
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
	)

	r.Route("/", func(r chi.Router) {
		r.Post("/", handlers.Shorten)
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

	err = http.ListenAndServe(config.Current.ServerAddress, router())
	if err != nil {
		panic(err)
	}
}
