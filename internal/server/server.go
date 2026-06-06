package server

import (
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/fredsaggio/bondrota-api/internal/destinos"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/paradas"
	"github.com/fredsaggio/bondrota-api/internal/reservas"
	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/go-chi/chi/v5"
)

const reqBodyLimitBytes = 250 * 1024

type Handlers struct {
	AdminHandler       *admin.AdminHandler
	VeiculoHandler     *veiculos.VeiculoHandler
	DestinoHandler     *destinos.DestinoHandler
	ParadaHandler      *paradas.ParadaHandler
	RotaInternaHandler *rotasinternas.RotaInternaHandler
	MotoristaHandler   *motoristas.MotoristaHandler
	ClienteHandler     *clientes.ClienteHandler
	VinculoHandler     *clientes.VinculoHandler
	ReservaHandler     *reservas.ReservaHandler
	ViagemHandler      *viagens.ViagemHandler
}

type Server struct {
	handlers Handlers
	authSvc  *auth.AuthService
}

func NewServer(handlers Handlers, authSvc *auth.AuthService) *Server {
	return &Server{
		handlers: handlers,
		authSvc:  authSvc,
	}
}

func (srv *Server) RegisterRoutes(r chi.Router) {
	r.Use(limitRequestBody(reqBodyLimitBytes))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/admin/login", srv.handlers.AdminHandler.Login)
	r.Post("/motoristas/login", srv.handlers.MotoristaHandler.Login)
	r.Post("/clientes/login", srv.handlers.ClienteHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(srv.authSvc.Authenticate)

		r.Route("/admin", func(r chi.Router) {
			r.Post("/", srv.handlers.AdminHandler.Create)
			r.Get("/", srv.handlers.AdminHandler.List)
			r.Get("/{adminID}", srv.handlers.AdminHandler.GetByID)
			r.Put("/{adminID}", srv.handlers.AdminHandler.Update)
			r.Delete("/{adminID}", srv.handlers.AdminHandler.Delete)
		})

		r.Route("/veiculos", func(r chi.Router) {
			r.Post("/", srv.handlers.VeiculoHandler.Create)
			r.Get("/", srv.handlers.VeiculoHandler.List)
			r.Get("/{veiculoID}", srv.handlers.VeiculoHandler.GetByID)
			r.Put("/{veiculoID}", srv.handlers.VeiculoHandler.Update)
			r.Delete("/{veiculoID}", srv.handlers.VeiculoHandler.Delete)
		})

		r.Route("/destinos", func(r chi.Router) {
			r.Post("/", srv.handlers.DestinoHandler.Create)
			r.Get("/", srv.handlers.DestinoHandler.List)
			r.Get("/cidade/{cidade}", srv.handlers.DestinoHandler.ListByCity)
			r.Get("/{id}", srv.handlers.DestinoHandler.GetByID)
			r.Put("/{id}", srv.handlers.DestinoHandler.Update)
			r.Delete("/{id}", srv.handlers.DestinoHandler.Delete)
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

		r.Route("/motoristas", func(r chi.Router) {
			r.Post("/", srv.handlers.MotoristaHandler.Create)
			r.Get("/", srv.handlers.MotoristaHandler.List)
			r.Get("/{id}", srv.handlers.MotoristaHandler.GetByID)
			r.Put("/{id}", srv.handlers.MotoristaHandler.Update)
			r.Delete("/{id}", srv.handlers.MotoristaHandler.Delete)
		})

		r.Route("/clientes", func(r chi.Router) {
			r.Post("/", srv.handlers.ClienteHandler.Create)
			r.Get("/", srv.handlers.ClienteHandler.List)
			r.Get("/{clienteID}", srv.handlers.ClienteHandler.GetByID)
			r.Put("/{clienteID}", srv.handlers.ClienteHandler.Update)
			r.Delete("/{clienteID}", srv.handlers.ClienteHandler.Delete)
			r.Get("/{clienteID}/reservas/", srv.handlers.ReservaHandler.ListByCliente)

			r.Route("/{clienteID}/vinculos", func(r chi.Router) {
				r.Post("/", srv.handlers.VinculoHandler.Create)
				r.Get("/", srv.handlers.VinculoHandler.ListByCliente)
				r.Get("/{vinculoID}", srv.handlers.VinculoHandler.GetByID)
				r.Put("/{vinculoID}", srv.handlers.VinculoHandler.Update)
				r.Delete("/{vinculoID}", srv.handlers.VinculoHandler.Delete)
				r.Post("/{vinculoID}/reservas/", srv.handlers.ReservaHandler.CreateByVinculo)
				r.Get("/{vinculoID}/reservas/", srv.handlers.ReservaHandler.ListByVinculo)
			})
		})

		r.Route("/reservas", func(r chi.Router) {
			r.Get("/", srv.handlers.ReservaHandler.List)
			r.Get("/{reservaID}", srv.handlers.ReservaHandler.GetByID)
			r.Put("/{reservaID}", srv.handlers.ReservaHandler.Update)
			r.Post("/{reservaID}/cancelar", srv.handlers.ReservaHandler.Cancel)
			r.Delete("/{reservaID}", srv.handlers.ReservaHandler.Delete)
		})

		r.Route("/viagens", func(r chi.Router) {
			r.Get("/", srv.handlers.ViagemHandler.List)
			r.Get("/{viagemID}", srv.handlers.ViagemHandler.GetByID)
			r.Post("/{viagemID}/iniciar", srv.handlers.ViagemHandler.Iniciar)
			r.Post("/{viagemID}/concluir", srv.handlers.ViagemHandler.Concluir)
			r.Post("/{viagemID}/cancelar", srv.handlers.ViagemHandler.Cancelar)
			r.Get("/{viagemID}/reservas/", srv.handlers.ViagemHandler.ListReservas)
			r.Put("/{viagemID}/reservas/{reservaID}/presenca", srv.handlers.ViagemHandler.AtualizarPresenca)
		})
	})
}

func limitRequestBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
