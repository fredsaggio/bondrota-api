package server

import (
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/go-chi/chi/v5"
)

const reqBodyLimitBytes = 250 * 1024

type Handlers struct {
	AdminHandler *admin.AdminHandler
}

type Server struct {
	handlers Handlers
}

func NewServer(handlers Handlers) *Server {
	return &Server{
		handlers: handlers,
	}
}

func (srv *Server) RegisterRoutes(r chi.Router) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/admin", func(r chi.Router) {
		r.Post("/", srv.handlers.AdminHandler.Create)
		r.Get("/", srv.handlers.AdminHandler.List)
		r.Get("/{adminID}", srv.handlers.AdminHandler.GetByID)
		r.Put("/{adminID}", srv.handlers.AdminHandler.Update)
		r.Delete("/{adminID}", srv.handlers.AdminHandler.Delete)
		r.Post("/login", srv.handlers.AdminHandler.Login)
	})
}