package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 250 KB limit
const reqBodyLimitBytes = 250 * 1024

type Server struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Server {
	return &Server{
		pool: pool,
	}
}

func (srv *Server) RegisterRoutes(r chi.Router) {

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	
}
