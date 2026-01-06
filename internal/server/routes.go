package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"go-backend-example/pkg/httpx/reply"
)

func (s Server) RegisterRoutes(r chi.Router) { //nolint:funlen
	r.Route("/", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			// unauthorized zone
			r.Route("/example", func(r chi.Router) {
				r.Post("/", handler(s.postV1Example))
				r.Get("/{id}", handler(s.getV1Example))
			})
		})
	})
}

func handler(f func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			reply.Error(r.Context(), w, err)
		}
	}
}
