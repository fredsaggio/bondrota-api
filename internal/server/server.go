package server

import (
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/fredsaggio/bondrota-api/internal/destinos"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/municipios"
	"github.com/fredsaggio/bondrota-api/internal/paradas"
	"github.com/fredsaggio/bondrota-api/internal/reservas"
	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
	"github.com/fredsaggio/bondrota-api/internal/storage"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/go-chi/chi/v5"
)

const reqBodyLimitBytes = 250 * 1024

type Handlers struct {
	AdminHandler        *admin.AdminHandler
	VeiculoHandler      *veiculos.VeiculoHandler
	DestinoHandler      *destinos.DestinoHandler
	MunicipioHandler    *municipios.Handler
	ParadaHandler       *paradas.ParadaHandler
	RotaInternaHandler  *rotasinternas.RotaInternaHandler
	MotoristaHandler    *motoristas.MotoristaHandler
	ClienteHandler      *clientes.ClienteHandler
	VinculoHandler      *clientes.VinculoHandler
	ReservaHandler      *reservas.ReservaHandler
	ViagemHandler       *viagens.ViagemHandler
	PlanejamentoHandler *viagens.PlanejamentoHandler
	HorarioTurnoHandler *viagens.HorarioTurnoViagemHandler
	RotaDinamicaHandler *rotasdinamicas.RotaDinamicaHandler
	StorageHandler      *storage.Handler
}

type Server struct {
	handlers Handlers
	authSvc  *auth.AuthService
	config   Config
}

type Config struct {
	BaseCity string `json:"cidade_base"`
}

func NewServer(handlers Handlers, authSvc *auth.AuthService, config Config) *Server {
	return &Server{
		handlers: handlers,
		authSvc:  authSvc,
		config:   config,
	}
}

func (srv *Server) RegisterRoutes(r chi.Router) {
	r.Use(limitRequestBody(reqBodyLimitBytes))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
		httputils.Respond(w, http.StatusOK, srv.config)
	})

	r.Post("/admin/login", srv.handlers.AdminHandler.Login)
	r.Post("/motoristas/login", srv.handlers.MotoristaHandler.Login)
	r.Post("/clientes/login", srv.handlers.ClienteHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(srv.authSvc.Authenticate)

		r.Route("/admin", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin))
			r.Post("/", srv.handlers.AdminHandler.Create)
			r.Get("/", srv.handlers.AdminHandler.List)
			r.Get("/{adminID}", srv.handlers.AdminHandler.GetByID)
			r.Put("/{adminID}", srv.handlers.AdminHandler.Update)
			r.Delete("/{adminID}", srv.handlers.AdminHandler.Delete)
		})

		r.Route("/veiculos", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin))
			r.Post("/", srv.handlers.VeiculoHandler.Create)
			r.Get("/", srv.handlers.VeiculoHandler.List)
			r.Get("/{veiculoID}", srv.handlers.VeiculoHandler.GetByID)
			r.Put("/{veiculoID}", srv.handlers.VeiculoHandler.Update)
			r.Delete("/{veiculoID}", srv.handlers.VeiculoHandler.Delete)
		})

		r.Route("/destinos", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin))
			r.Post("/", srv.handlers.DestinoHandler.Create)
			r.Get("/", srv.handlers.DestinoHandler.List)
			r.Get("/municipio/{municipioID}", srv.handlers.DestinoHandler.ListByMunicipio)
			r.Get("/{id}", srv.handlers.DestinoHandler.GetByID)
			r.Put("/{id}", srv.handlers.DestinoHandler.Update)
			r.Delete("/{id}", srv.handlers.DestinoHandler.Delete)
		})

		r.Route("/municipios", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin))
			r.Get("/", srv.handlers.MunicipioHandler.ListByUF)
		})

		r.Route("/paradas", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin))
			r.Post("/", srv.handlers.ParadaHandler.Create)
			r.Get("/", srv.handlers.ParadaHandler.List)
			r.Get("/{id}", srv.handlers.ParadaHandler.GetByID)
			r.Put("/{id}", srv.handlers.ParadaHandler.Update)
			r.Delete("/{id}", srv.handlers.ParadaHandler.Delete)
		})

		r.Route("/rotas-internas", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin))
			r.Post("/", srv.handlers.RotaInternaHandler.Create)
			r.Get("/", srv.handlers.RotaInternaHandler.List)
			r.Get("/{id}", srv.handlers.RotaInternaHandler.GetByID)
			r.Put("/{id}/paradas", srv.handlers.RotaInternaHandler.UpdateParadas)
			r.Delete("/{id}", srv.handlers.RotaInternaHandler.Delete)
		})

		r.Route("/motoristas", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin))
			r.Post("/", srv.handlers.MotoristaHandler.Create)
			r.Get("/", srv.handlers.MotoristaHandler.List)
			r.Get("/{id}", srv.handlers.MotoristaHandler.GetByID)
			r.Put("/{id}", srv.handlers.MotoristaHandler.Update)
			r.Delete("/{id}", srv.handlers.MotoristaHandler.Delete)
		})

		r.Route("/clientes", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin, auth.RoleCliente))
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
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin, auth.RoleCliente))
			r.Get("/", srv.handlers.ReservaHandler.List)
			r.Get("/{reservaID}", srv.handlers.ReservaHandler.GetByID)
			r.Put("/{reservaID}", srv.handlers.ReservaHandler.Update)
			r.Post("/{reservaID}/cancelar", srv.handlers.ReservaHandler.Cancel)
			r.Delete("/{reservaID}", srv.handlers.ReservaHandler.Delete)
		})

		r.Group(func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin, auth.RoleMotorista, auth.RoleCliente))
			r.Get("/viagens/{viagemID}/localizacao", srv.handlers.ViagemHandler.GetLocalizacao)
		})

		r.Group(func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin, auth.RoleMotorista))
			r.Put("/viagens/{viagemID}/localizacao", srv.handlers.ViagemHandler.AtualizarLocalizacao)
		})

		r.Route("/viagens", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin, auth.RoleMotorista))
			r.Get("/", srv.handlers.ViagemHandler.List)
			r.Get("/{viagemID}", srv.handlers.ViagemHandler.GetByID)
			r.Post("/{viagemID}/iniciar", srv.handlers.ViagemHandler.Iniciar)
			r.Post("/{viagemID}/concluir", srv.handlers.ViagemHandler.Concluir)
			r.Post("/{viagemID}/cancelar", srv.handlers.ViagemHandler.Cancelar)
			r.Get("/{viagemID}/horarios", srv.handlers.ViagemHandler.ListHorarios)
			r.Get("/{viagemID}/reservas/", srv.handlers.ViagemHandler.ListReservas)
			r.Put("/{viagemID}/reservas/{reservaID}/presenca", srv.handlers.ViagemHandler.AtualizarPresenca)
			r.Post("/{viagemID}/rota-dinamica/calcular", srv.handlers.RotaDinamicaHandler.Calcular)
			r.Post("/{viagemID}/rota-dinamica", srv.handlers.RotaDinamicaHandler.Create)
			r.Get("/{viagemID}/rota-dinamica", srv.handlers.RotaDinamicaHandler.GetByViagem)
			r.Delete("/{viagemID}/rota-dinamica", srv.handlers.RotaDinamicaHandler.DeleteByViagem)
		})

		r.Route("/planejamentos", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin))
			r.Post("/viagens", srv.handlers.PlanejamentoHandler.PlanejarViagens)
		})

		r.Route("/horarios-turno-viagem", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin))
			r.Post("/", srv.handlers.HorarioTurnoHandler.Create)
			r.Get("/", srv.handlers.HorarioTurnoHandler.List)
			r.Get("/{horarioTurnoID}", srv.handlers.HorarioTurnoHandler.GetByID)
			r.Put("/{horarioTurnoID}", srv.handlers.HorarioTurnoHandler.Update)
			r.Delete("/{horarioTurnoID}", srv.handlers.HorarioTurnoHandler.Delete)
		})

		r.Route("/storage", func(r chi.Router) {
			r.Use(srv.authSvc.RequireRole(auth.RoleAdmin, auth.RoleCliente, auth.RoleMotorista))
			r.Post("/signed-upload-url", srv.handlers.StorageHandler.CreateSignedUploadURL)
			r.Post("/signed-download-url", srv.handlers.StorageHandler.CreateSignedDownloadURL)
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
