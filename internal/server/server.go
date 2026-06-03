package server

import (
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/paradas"
	"github.com/fredsaggio/bondrota-api/internal/pontos"
	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
	"github.com/go-chi/chi/v5"
)

const reqBodyLimitBytes = 250 * 1024

type Handlers struct {
	AdminHandler   *admin.AdminHandler
	VeiculoHandler *veiculos.VeiculoHandler
	PontoHandler   *pontos.PontoHandler
	ParadaHandler   *paradas.ParadaHandler
	RotaInternaHandler *rotasinternas.RotaInternaHandler
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

	r.Route("/veiculos", func(r chi.Router) {
		r.Post("/", srv.handlers.VeiculoHandler.Create)
		r.Get("/", srv.handlers.VeiculoHandler.List)
		r.Get("/{veiculoID}", srv.handlers.VeiculoHandler.GetByID)
		r.Put("/{veiculoID}", srv.handlers.VeiculoHandler.Update)
		r.Delete("/{veiculoID}", srv.handlers.VeiculoHandler.Delete)
	})

	r.Route("/pontos", func(r chi.Router) {
		r.Post("/", srv.handlers.PontoHandler.Create)
		r.Get("/", srv.handlers.PontoHandler.List)
		r.Get("/cidade/{cidade}", srv.handlers.PontoHandler.ListByCity)
		r.Get("/{id}", srv.handlers.PontoHandler.GetByID)
		r.Put("/{id}", srv.handlers.PontoHandler.Update)
		r.Delete("/{id}", srv.handlers.PontoHandler.Delete)
	})

	r.Route("/paradas", func(r chi.Router) {
		r.Post("/", srv.handlers.ParadaHandler.Create)
		r.Get("/", srv.handlers.ParadaHandler.List)
		r.Get("/cidade/{cidade}", srv.handlers.ParadaHandler.ListByCity)
		r.Get("/{id}", srv.handlers.ParadaHandler.GetByID)
		r.Put("/{id}", srv.handlers.ParadaHandler.Update)
		r.Delete("/{id}", srv.handlers.ParadaHandler.Delete)
	})

	r.Route("/rotas-internas", func(r chi.Router) {
		r.Post("/", srv.handlers.RotaInternaHandler.Create)
		r.Get("/", srv.handlers.RotaInternaHandler.List)
		r.Get("/cidade/{cidade}", srv.handlers.RotaInternaHandler.ListByCity)
		r.Get("/{id}", srv.handlers.RotaInternaHandler.GetByID)
		r.Put("/{id}/paradas", srv.handlers.RotaInternaHandler.UpdateParadas)
		r.Delete("/{id}", srv.handlers.RotaInternaHandler.Delete)
	})
}
