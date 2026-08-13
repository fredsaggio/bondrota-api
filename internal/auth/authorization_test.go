package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/publicid"
	"github.com/go-chi/chi/v5"
)

const authorizationClienteID = "cli_012345678901234567890"

type authorizationResolver struct{}

func (authorizationResolver) Resolve(_ context.Context, prefix publicid.Prefix, value string) (int64, error) {
	if prefix == publicid.Cliente && value == authorizationClienteID {
		return 10, nil
	}
	return 0, publicid.ErrNotFound
}

func TestRequireUserIDOrRole(t *testing.T) {
	tests := []struct {
		name       string
		claims     *Claims
		path       string
		wantStatus int
	}{
		{name: "owner", claims: &Claims{UserID: 10, Role: RoleCliente}, path: "/clientes/" + authorizationClienteID, wantStatus: http.StatusNoContent},
		{name: "other cliente", claims: &Claims{UserID: 11, Role: RoleCliente}, path: "/clientes/" + authorizationClienteID, wantStatus: http.StatusForbidden},
		{name: "admin bypass", claims: &Claims{UserID: 1, Role: RoleAdmin}, path: "/clientes/" + authorizationClienteID, wantStatus: http.StatusNoContent},
		{name: "wrong role", claims: &Claims{UserID: 10, Role: RoleMotorista}, path: "/clientes/" + authorizationClienteID, wantStatus: http.StatusForbidden},
		{name: "missing claims", path: "/clientes/" + authorizationClienteID, wantStatus: http.StatusUnauthorized},
		{name: "invalid id", claims: &Claims{UserID: 10, Role: RoleCliente}, path: "/clientes/not-a-public-id", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.With(publicid.ResolveParam(authorizationResolver{}, publicid.Cliente, "clienteID"), func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tc.claims != nil {
						r = r.WithContext(context.WithValue(r.Context(), ClaimsKey, tc.claims))
					}
					next.ServeHTTP(w, r)
				})
			}, RequireUserIDOrRole("clienteID", RoleCliente, RoleAdmin)).Get("/clientes/{clienteID}", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

var _ publicid.Resolver = authorizationResolver{}
