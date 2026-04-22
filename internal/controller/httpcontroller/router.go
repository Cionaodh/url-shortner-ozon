package httpcontroller

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func NewRouter(s Service, log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer) // ловит панику

	handler := NewHandler(s, log)

	r.Post("/", handler.CreateLink)
	r.Get("/{short}", handler.GetLink)

	return r
}
