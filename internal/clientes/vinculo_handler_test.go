package clientes_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/fredsaggio/bondrota-api/internal/clientes"
)

type fakeVinculoService struct {
	createFn        func(ctx context.Context, input clientes.VinculoInput) (*clientes.Vinculo, error)
	getFn           func(ctx context.Context, vinculoID int64) (*clientes.Vinculo, error)
	listFn          func(ctx context.Context) ([]clientes.VinculoComCliente, error)
	listByClienteFn func(ctx context.Context, clienteID int64) ([]clientes.Vinculo, error)
	updateFn        func(ctx context.Context, vinculoID int64, input clientes.VinculoUpdateInput) (*clientes.Vinculo, error)
	deleteFn        func(ctx context.Context, vinculoID int64) error
}

func (s fakeVinculoService) List(ctx context.Context) ([]clientes.VinculoComCliente, error) {
	return s.listFn(ctx)
}

func (s fakeVinculoService) Create(ctx context.Context, input clientes.VinculoInput) (*clientes.Vinculo, error) {
	return s.createFn(ctx, input)
}

func (s fakeVinculoService) GetByID(ctx context.Context, vinculoID int64) (*clientes.Vinculo, error) {
	return s.getFn(ctx, vinculoID)
}

func (s fakeVinculoService) ListByCliente(ctx context.Context, clienteID int64) ([]clientes.Vinculo, error) {
	return s.listByClienteFn(ctx, clienteID)
}

func (s fakeVinculoService) Update(ctx context.Context, vinculoID int64, input clientes.VinculoUpdateInput) (*clientes.Vinculo, error) {
	return s.updateFn(ctx, vinculoID, input)
}

func (s fakeVinculoService) Delete(ctx context.Context, vinculoID int64) error {
	return s.deleteFn(ctx, vinculoID)
}

func newVinculoRouter(h *clientes.VinculoHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/vinculos/", h.List)
	r.Post("/clientes/{clienteID}/vinculos", h.Create)
	r.Get("/clientes/{clienteID}/vinculos", h.ListByCliente)
	r.Get("/clientes/{clienteID}/vinculos/{vinculoID}", h.GetByID)
	r.Put("/clientes/{clienteID}/vinculos/{vinculoID}", h.Update)
	r.Delete("/clientes/{clienteID}/vinculos/{vinculoID}", h.Delete)
	return r
}

func validVinculoBody() map[string]any {
	return map[string]any{
		"tipo":            "estudante",
		"turno":           "NT",
		"destino_id":      2,
		"rota_interna_id": 3,
		"curso":           "Computacao",
		"validade":        "2026-07-01",
		"horarios_fixos":  []int{1, 3},
	}
}

func TestVinculoHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       map[string]any
		svc        fakeVinculoService
		wantStatus int
	}{
		{
			name: "success",
			path: "/clientes/1/vinculos",
			body: validVinculoBody(),
			svc: fakeVinculoService{
				createFn: func(_ context.Context, input clientes.VinculoInput) (*clientes.Vinculo, error) {
					if input.ClienteID != 1 || input.DestinoID != 2 || input.RotaInternaID != 3 {
						t.Fatalf("unexpected input: %+v", input)
					}
					return sampleVinculo(), nil
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid cliente id",
			path:       "/clientes/abc/vinculos",
			body:       validVinculoBody(),
			svc:        fakeVinculoService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing destino id",
			path:       "/clientes/1/vinculos",
			body:       map[string]any{"rota_interna_id": 3, "validade": "2026-07-01"},
			svc:        fakeVinculoService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "domain validation error",
			path: "/clientes/1/vinculos",
			body: validVinculoBody(),
			svc: fakeVinculoService{
				createFn: func(_ context.Context, _ clientes.VinculoInput) (*clientes.Vinculo, error) {
					return nil, clientes.ErrTurnoInvalido
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal error",
			path: "/clientes/1/vinculos",
			body: validVinculoBody(),
			svc: fakeVinculoService{
				createFn: func(_ context.Context, _ clientes.VinculoInput) (*clientes.Vinculo, error) {
					return nil, errors.New("db")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := clientes.NewVinculoHandler(tc.svc)
			req := httptest.NewRequest(http.MethodPost, tc.path, body(tc.body))
			rr := httptest.NewRecorder()

			newVinculoRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestVinculoHandler_List(t *testing.T) {
	t.Run("returns vinculos with cliente_nome flattened", func(t *testing.T) {
		svc := fakeVinculoService{
			listFn: func(_ context.Context) ([]clientes.VinculoComCliente, error) {
				return []clientes.VinculoComCliente{
					{Vinculo: *sampleVinculo(), ClienteNome: "Maria Souza"},
				}, nil
			},
		}

		rr := httptest.NewRecorder()
		newVinculoRouter(clientes.NewVinculoHandler(svc)).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/vinculos/", nil))

		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var got []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 vinculo, got %d", len(got))
		}
		if got[0]["cliente_nome"] != "Maria Souza" {
			t.Fatalf("want cliente_nome Maria Souza, got %v", got[0]["cliente_nome"])
		}
		// O painel espera os campos do vinculo no mesmo nivel de cliente_nome.
		if got[0]["id"] != float64(10) || got[0]["cliente_id"] != float64(1) {
			t.Fatalf("vinculo fields not flattened: %v", got[0])
		}
		if got[0]["validade"] != "2026-07-01" {
			t.Fatalf("want validade 2026-07-01, got %v", got[0]["validade"])
		}
	})

	t.Run("returns empty array when there is no vinculo", func(t *testing.T) {
		svc := fakeVinculoService{
			listFn: func(_ context.Context) ([]clientes.VinculoComCliente, error) {
				return nil, nil
			},
		}

		rr := httptest.NewRecorder()
		newVinculoRouter(clientes.NewVinculoHandler(svc)).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/vinculos/", nil))

		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rr.Code)
		}
		if body := strings.TrimSpace(rr.Body.String()); body != "[]" {
			t.Fatalf("want [], got %s", body)
		}
	})

	t.Run("translates store failure to 500", func(t *testing.T) {
		svc := fakeVinculoService{
			listFn: func(_ context.Context) ([]clientes.VinculoComCliente, error) {
				return nil, errors.New("db")
			},
		}

		rr := httptest.NewRecorder()
		newVinculoRouter(clientes.NewVinculoHandler(svc)).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/vinculos/", nil))

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("want 500, got %d", rr.Code)
		}
	})
}

func TestVinculoHandler_ListByCliente(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		svc        fakeVinculoService
		wantStatus int
	}{
		{
			name: "success",
			path: "/clientes/1/vinculos",
			svc: fakeVinculoService{
				listByClienteFn: func(_ context.Context, clienteID int64) ([]clientes.Vinculo, error) {
					if clienteID != 1 {
						t.Fatalf("unexpected clienteID: %d", clienteID)
					}
					return []clientes.Vinculo{*sampleVinculo()}, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid cliente id",
			path:       "/clientes/abc/vinculos",
			svc:        fakeVinculoService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			path: "/clientes/1/vinculos",
			svc: fakeVinculoService{
				listByClienteFn: func(_ context.Context, _ int64) ([]clientes.Vinculo, error) {
					return nil, errors.New("db")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := clientes.NewVinculoHandler(tc.svc)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			newVinculoRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestVinculoHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		svc        fakeVinculoService
		wantStatus int
	}{
		{
			name: "success",
			path: "/clientes/1/vinculos/10",
			svc: fakeVinculoService{
				getFn: func(_ context.Context, vinculoID int64) (*clientes.Vinculo, error) {
					if vinculoID != 10 {
						t.Fatalf("unexpected vinculoID: %d", vinculoID)
					}
					return sampleVinculo(), nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			path:       "/clientes/1/vinculos/abc",
			svc:        fakeVinculoService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			path: "/clientes/1/vinculos/99",
			svc: fakeVinculoService{
				getFn: func(_ context.Context, _ int64) (*clientes.Vinculo, error) {
					return nil, clientes.ErrVinculoNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "wrong cliente returns not found",
			path: "/clientes/1/vinculos/10",
			svc: fakeVinculoService{
				getFn: func(_ context.Context, _ int64) (*clientes.Vinculo, error) {
					v := sampleVinculo()
					v.ClienteID = 2
					return v, nil
				},
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := clientes.NewVinculoHandler(tc.svc)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			newVinculoRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestVinculoHandler_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := clientes.NewVinculoHandler(fakeVinculoService{
			getFn: func(_ context.Context, _ int64) (*clientes.Vinculo, error) {
				return sampleVinculo(), nil
			},
			updateFn: func(_ context.Context, vinculoID int64, input clientes.VinculoUpdateInput) (*clientes.Vinculo, error) {
				if vinculoID != 10 || input.DestinoID != 2 {
					t.Fatalf("unexpected update: %d %+v", vinculoID, input)
				}
				return sampleVinculo(), nil
			},
		})

		req := httptest.NewRequest(http.MethodPut, "/clientes/1/vinculos/10", body(validVinculoBody()))
		rr := httptest.NewRecorder()
		newVinculoRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong cliente returns not found before update", func(t *testing.T) {
		h := clientes.NewVinculoHandler(fakeVinculoService{
			getFn: func(_ context.Context, _ int64) (*clientes.Vinculo, error) {
				v := sampleVinculo()
				v.ClienteID = 2
				return v, nil
			},
		})

		req := httptest.NewRequest(http.MethodPut, "/clientes/1/vinculos/10", body(validVinculoBody()))
		rr := httptest.NewRecorder()
		newVinculoRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("want %d, got %d", http.StatusNotFound, rr.Code)
		}
	})
}

func TestVinculoHandler_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := clientes.NewVinculoHandler(fakeVinculoService{
			getFn: func(_ context.Context, _ int64) (*clientes.Vinculo, error) {
				return sampleVinculo(), nil
			},
			deleteFn: func(_ context.Context, vinculoID int64) error {
				if vinculoID != 10 {
					t.Fatalf("unexpected vinculoID: %d", vinculoID)
				}
				return nil
			},
		})

		req := httptest.NewRequest(http.MethodDelete, "/clientes/1/vinculos/10", nil)
		rr := httptest.NewRecorder()
		newVinculoRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("want %d, got %d", http.StatusNoContent, rr.Code)
		}
	})

	t.Run("vinculo com reservas registradas vira 409", func(t *testing.T) {
		h := clientes.NewVinculoHandler(fakeVinculoService{
			getFn: func(_ context.Context, _ int64) (*clientes.Vinculo, error) {
				return sampleVinculo(), nil
			},
			deleteFn: func(_ context.Context, _ int64) error {
				// reservas.vinculo_id usa ON DELETE RESTRICT
				return fmt.Errorf("db/vinculoStore.Delete: %w", &pgconn.PgError{Code: "23503", ConstraintName: "reservas_vinculo_id_fkey"})
			},
		})

		req := httptest.NewRequest(http.MethodDelete, "/clientes/1/vinculos/10", nil)
		rr := httptest.NewRecorder()
		newVinculoRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Fatalf("want %d, got %d", http.StatusConflict, rr.Code)
		}
	})

	t.Run("delete error not found", func(t *testing.T) {
		h := clientes.NewVinculoHandler(fakeVinculoService{
			getFn: func(_ context.Context, _ int64) (*clientes.Vinculo, error) {
				return sampleVinculo(), nil
			},
			deleteFn: func(_ context.Context, _ int64) error {
				return clientes.ErrVinculoNotFound
			},
		})

		req := httptest.NewRequest(http.MethodDelete, "/clientes/1/vinculos/10", nil)
		rr := httptest.NewRecorder()
		newVinculoRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("want %d, got %d", http.StatusNotFound, rr.Code)
		}
	})
}
