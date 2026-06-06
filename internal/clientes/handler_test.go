package clientes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/clientes"
)

type fakeClienteService struct {
	loginFn  func(ctx context.Context, cpf, senha string) (string, error)
	createFn func(ctx context.Context, input clientes.ClienteInput) (*clientes.Cliente, error)
	getFn    func(ctx context.Context, clienteID int64) (*clientes.ClienteComVinculos, error)
	listFn   func(ctx context.Context) ([]clientes.Cliente, error)
	updateFn func(ctx context.Context, clienteID int64, updateFunc func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error)
	deleteFn func(ctx context.Context, clienteID int64) error
}

func (s fakeClienteService) Login(ctx context.Context, cpf, senha string) (string, error) {
	return s.loginFn(ctx, cpf, senha)
}

func (s fakeClienteService) Create(ctx context.Context, input clientes.ClienteInput) (*clientes.Cliente, error) {
	return s.createFn(ctx, input)
}

func (s fakeClienteService) GetByID(ctx context.Context, clienteID int64) (*clientes.ClienteComVinculos, error) {
	return s.getFn(ctx, clienteID)
}

func (s fakeClienteService) List(ctx context.Context) ([]clientes.Cliente, error) {
	return s.listFn(ctx)
}

func (s fakeClienteService) Update(ctx context.Context, clienteID int64, updateFunc func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error) {
	return s.updateFn(ctx, clienteID, updateFunc)
}

func (s fakeClienteService) Delete(ctx context.Context, clienteID int64) error {
	return s.deleteFn(ctx, clienteID)
}

func newClienteRouter(h *clientes.ClienteHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/clientes/login", h.Login)
	r.Post("/clientes", h.Create)
	r.Get("/clientes", h.List)
	r.Get("/clientes/{clienteID}", h.GetByID)
	r.Put("/clientes/{clienteID}", h.Update)
	r.Delete("/clientes/{clienteID}", h.Delete)
	return r
}

func body(v any) *bytes.Buffer {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(v)
	return &buf
}

func TestClienteHandler_Login(t *testing.T) {
	tests := []struct {
		name       string
		body       *bytes.Buffer
		svc        fakeClienteService
		wantStatus int
	}{
		{
			name: "success",
			body: body(map[string]any{"cpf": " 123 ", "senha": "secret"}),
			svc: fakeClienteService{
				loginFn: func(_ context.Context, cpf, senha string) (string, error) {
					if cpf != "123" || senha != "secret" {
						t.Fatalf("unexpected login args: %q %q", cpf, senha)
					}
					return "token", nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing cpf",
			body:       body(map[string]any{"senha": "secret"}),
			svc:        fakeClienteService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid credentials",
			body: body(map[string]any{"cpf": "123", "senha": "wrong"}),
			svc: fakeClienteService{
				loginFn: func(_ context.Context, _, _ string) (string, error) {
					return "", auth.ErrInvalidCredentials
				},
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := clientes.NewClienteHandler(tc.svc)
			req := httptest.NewRequest(http.MethodPost, "/clientes/login", tc.body)
			rr := httptest.NewRecorder()

			newClienteRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestClienteHandler_Create(t *testing.T) {
	validBody := map[string]any{
		"nome":      " Maria ",
		"cpf":       "123",
		"senha":     "secret",
		"data_nasc": "2000-01-02",
	}

	tests := []struct {
		name       string
		body       *bytes.Buffer
		svc        fakeClienteService
		wantStatus int
	}{
		{
			name: "success",
			body: body(validBody),
			svc: fakeClienteService{
				createFn: func(_ context.Context, input clientes.ClienteInput) (*clientes.Cliente, error) {
					if input.Nome != "Maria" || input.CPF != "123" {
						t.Fatalf("unexpected input: %+v", input)
					}
					return sampleCliente(), nil
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid date",
			body:       body(map[string]any{"nome": "Maria", "cpf": "123", "senha": "secret", "data_nasc": "02-01-2000"}),
			svc:        fakeClienteService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			body: body(validBody),
			svc: fakeClienteService{
				createFn: func(_ context.Context, _ clientes.ClienteInput) (*clientes.Cliente, error) {
					return nil, errors.New("db")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := clientes.NewClienteHandler(tc.svc)
			req := httptest.NewRequest(http.MethodPost, "/clientes", tc.body)
			rr := httptest.NewRecorder()

			newClienteRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestClienteHandler_GetListUpdateDelete(t *testing.T) {
	t.Run("get not found", func(t *testing.T) {
		h := clientes.NewClienteHandler(fakeClienteService{
			getFn: func(_ context.Context, _ int64) (*clientes.ClienteComVinculos, error) {
				return nil, clientes.ErrNotFound
			},
		})

		req := httptest.NewRequest(http.MethodGet, "/clientes/99", nil)
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("want %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("list success", func(t *testing.T) {
		h := clientes.NewClienteHandler(fakeClienteService{
			listFn: func(_ context.Context) ([]clientes.Cliente, error) {
				return []clientes.Cliente{*sampleCliente()}, nil
			},
		})

		req := httptest.NewRequest(http.MethodGet, "/clientes", nil)
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("update applies request", func(t *testing.T) {
		h := clientes.NewClienteHandler(fakeClienteService{
			updateFn: func(_ context.Context, clienteID int64, updateFunc func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error) {
				if clienteID != 1 {
					t.Fatalf("unexpected clienteID: %d", clienteID)
				}
				c := sampleCliente()
				changed, err := updateFunc(c)
				if err != nil || !changed {
					t.Fatalf("expected changed without error, changed=%v err=%v", changed, err)
				}
				return c, nil
			},
		})

		req := httptest.NewRequest(http.MethodPut, "/clientes/1", body(map[string]any{"nome": "Ana"}))
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}
	})

	t.Run("delete success", func(t *testing.T) {
		h := clientes.NewClienteHandler(fakeClienteService{
			deleteFn: func(_ context.Context, clienteID int64) error {
				if clienteID != 1 {
					t.Fatalf("unexpected clienteID: %d", clienteID)
				}
				return nil
			},
		})

		req := httptest.NewRequest(http.MethodDelete, "/clientes/1", nil)
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("want %d, got %d", http.StatusNoContent, rr.Code)
		}
	})
}
