package publicid

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("identificador público não encontrado")

type Resolver interface {
	Resolve(ctx context.Context, prefix Prefix, value string) (int64, error)
}

type resolver struct {
	db db.DB
}

func NewResolver(database db.DB) Resolver {
	return &resolver{db: database}
}

func (r *resolver) Resolve(ctx context.Context, prefix Prefix, value string) (int64, error) {
	if !Valid(value, prefix) {
		return 0, ErrNotFound
	}
	table, err := tableFor(prefix)
	if err != nil {
		return 0, err
	}

	var id int64
	if err := r.db.QueryRow(ctx, "SELECT id FROM "+table+" WHERE public_id = $1", value).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("resolve %s public id: %w", prefix, err)
	}
	return id, nil
}

func tableFor(prefix Prefix) (string, error) {
	switch prefix {
	case Admin:
		return "administrador", nil
	case Cliente:
		return "clientes", nil
	case Motorista:
		return "motoristas", nil
	case Vinculo:
		return "cliente_vinculos", nil
	case Reserva:
		return "reservas", nil
	case Viagem:
		return "viagens", nil
	default:
		return "", fmt.Errorf("unsupported public id prefix %q", prefix)
	}
}

type contextKey string

func resolvedKey(param string) contextKey {
	return contextKey("bondrota.public-id." + param)
}

// ResolveParam translates a validated public identifier at the HTTP boundary.
// Domain services and repositories only receive the internal BIGINT from here.
func ResolveParam(resolver Resolver, prefix Prefix, param string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if resolver == nil {
				http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
				return
			}

			id, err := resolver.Resolve(r.Context(), prefix, chi.URLParam(r, param))
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					http.Error(w, "Registro não encontrado.", http.StatusNotFound)
					return
				}
				http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), resolvedKey(param), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ResolvedParam(ctx context.Context, param string) (int64, bool) {
	id, ok := ctx.Value(resolvedKey(param)).(int64)
	return id, ok && id > 0
}
