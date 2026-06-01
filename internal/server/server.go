package server

import (
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/go-chi/chi/v5"
)

// 250 KB limit
const reqBodyLimitBytes = 250 * 1024

type Stores struct {
	AdminStore admin.AdminStore
}

type Server struct {
	stores Stores
}

func NewServer(stores Stores) *Server {
	return &Server{
		stores: stores,
	}
}

func (srv *Server) RegisterRoutes(r chi.Router) {

	adminHandler := admin.NewAdminHandler(srv.stores.AdminStore)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/admin", func(r chi.Router) {
		r.Post("/", adminHandler.Create)
		r.Get("/", adminHandler.List)
		r.Get("/{adminID}", adminHandler.GetByID)
		r.Put("/{adminID}", adminHandler.Update)
		r.Delete("/{adminID}", adminHandler.Delete)
	})

}
