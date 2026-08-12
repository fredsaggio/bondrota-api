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
	listFn          func(ctx context.Context, params clientes.VinculoListParams) (clientes.VinculoListResult, error)
	listByClienteFn func(ctx context.Context, clienteID int64) ([]clientes.Vinculo, error)
	updateFn        func(ctx context.Context, vinculoID int64, input clientes.VinculoUpdateInput) (*clientes.Vinculo, error)
	deleteFn        func(ctx context.Context, vinculoID int64) error
}

func (s fakeVinculoService) List(ctx context.Context, params clientes.VinculoListParams) (clientes.VinculoListResult, error) {
	return s.listFn(ctx, params)
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

func TestVinculoHandler_Create_OrganizaComprovante(t *testing.T) {
	body := body(map[string]any{
		"tipo":            "estagio",
		"turno":           "NT",
		"destino_id":      2,
		"rota_interna_id": 3,
		"curso":           "Computacao",
		"validade":        "2026-07-01",
		"horarios_fixos":  []int{1, 3},
		"comprovante":     "clientes/1/vinculos/_novo/xyz/comprovante-estagio.pdf",
	})

	svc := fakeVinculoService{
		createFn: func(_ context.Context, input clientes.VinculoInput) (*clientes.Vinculo, error) {
			if input.Comprovante != "clientes/1/vinculos/_novo/xyz/comprovante-estagio.pdf" {
				t.Fatalf("unexpected comprovante no create: %q", input.Comprovante)
			}
			v := sampleVinculo()
			v.ID = 9
			v.ClienteID = 1
			v.Comprovante = input.Comprovante
			return v, nil
		},
		updateFn: func(_ context.Context, vinculoID int64, input clientes.VinculoUpdateInput) (*clientes.Vinculo, error) {
			if vinculoID != 9 {
				t.Fatalf("unexpected vinculoID no update: %d", vinculoID)
			}
			// O caminho final carrega o tipo do vinculo, para dar pra saber pelo
			// nome do arquivo no Supabase se e comprovante de estagio ou de
			// faculdade sem precisar abrir o banco.
			if input.Comprovante != "clientes/1/vinculos/9/comprovante-estagio.pdf" {
				t.Fatalf("update nao organizou o comprovante corretamente: %q", input.Comprovante)
			}
			v := sampleVinculo()
			v.ID = 9
			v.ClienteID = 1
			v.Comprovante = input.Comprovante
			return v, nil
		},
	}

	mover := &fakeArquivoMovedor{}
	h := clientes.NewVinculoHandler(svc, mover)
	req := httptest.NewRequest(http.MethodPost, "/clientes/1/vinculos", body)
	rr := httptest.NewRecorder()
	newVinculoRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d — %s", rr.Code, rr.Body.String())
	}
	wantTo := "clientes/1/vinculos/9/comprovante-estagio.pdf"
	if mover.bucket != "documentos" || mover.from != "clientes/1/vinculos/_novo/xyz/comprovante-estagio.pdf" || mover.to != wantTo {
		t.Fatalf("unexpected move: bucket=%q from=%q to=%q", mover.bucket, mover.from, mover.to)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["comprovante"] != wantTo {
		t.Fatalf("want comprovante organizado na resposta, got %v", resp["comprovante"])
	}
}

func TestVinculoHandler_Create_FalhaAoOrganizarComprovanteNaoDerrubaCriacao(t *testing.T) {
	body := body(map[string]any{
		"tipo":            "estudante",
		"turno":           "NT",
		"destino_id":      2,
		"rota_interna_id": 3,
		"curso":           "Computacao",
		"validade":        "2026-07-01",
		"horarios_fixos":  []int{1, 3},
		"comprovante":     "clientes/1/vinculos/_novo/xyz/comprovante-estudante.pdf",
	})

	svc := fakeVinculoService{
		createFn: func(_ context.Context, input clientes.VinculoInput) (*clientes.Vinculo, error) {
			v := sampleVinculo()
			v.ID = 9
			v.ClienteID = 1
			v.Comprovante = input.Comprovante
			return v, nil
		},
		updateFn: func(context.Context, int64, clientes.VinculoUpdateInput) (*clientes.Vinculo, error) {
			t.Fatal("update nao deveria ser chamado quando mover falha")
			return nil, nil
		},
	}

	mover := &fakeArquivoMovedor{err: errors.New("supabase indisponivel")}
	h := clientes.NewVinculoHandler(svc, mover)
	req := httptest.NewRequest(http.MethodPost, "/clientes/1/vinculos", body)
	rr := httptest.NewRecorder()
	newVinculoRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("falha ao mover nao deveria derrubar a criacao: want 201, got %d — %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["comprovante"] != "clientes/1/vinculos/_novo/xyz/comprovante-estudante.pdf" {
		t.Fatalf("comprovante deveria continuar no caminho de espera apos falha, got %v", resp["comprovante"])
	}
}

func TestVinculoHandler_List(t *testing.T) {
	t.Run("returns vinculos with cliente_nome flattened", func(t *testing.T) {
		svc := fakeVinculoService{
			listFn: func(_ context.Context, _ clientes.VinculoListParams) (clientes.VinculoListResult, error) {
				return clientes.VinculoListResult{
					Items: []clientes.VinculoComCliente{
						{Vinculo: *sampleVinculo(), ClienteNome: "Maria Souza", DestinoNome: "Campus A"},
					},
					NextCursor: &clientes.VinculoCursor{ClienteNome: "Maria Souza", ID: 10},
					HasMore:    true,
				}, nil
			},
		}

		rr := httptest.NewRecorder()
		newVinculoRouter(clientes.NewVinculoHandler(svc)).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/vinculos/", nil))

		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp struct {
			Items      []map[string]any `json:"items"`
			NextCursor string           `json:"next_cursor"`
			HasMore    bool             `json:"has_more"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if len(resp.Items) != 1 {
			t.Fatalf("want 1 vinculo, got %d", len(resp.Items))
		}
		got := resp.Items
		if got[0]["cliente_nome"] != "Maria Souza" || got[0]["destino_nome"] != "Campus A" {
			t.Fatalf("nomes resolvidos nao vieram: %v", got[0])
		}
		// O painel espera os campos do vinculo no mesmo nivel de cliente_nome.
		if got[0]["id"] != float64(10) || got[0]["cliente_id"] != float64(1) {
			t.Fatalf("vinculo fields not flattened: %v", got[0])
		}
		if got[0]["validade"] != "2026-07-01" {
			t.Fatalf("want validade 2026-07-01, got %v", got[0]["validade"])
		}
		if resp.NextCursor == "" || !resp.HasMore {
			t.Fatalf("esperava next_cursor e has_more: %+v", resp)
		}
	})

	t.Run("returns empty array when there is no vinculo", func(t *testing.T) {
		svc := fakeVinculoService{
			listFn: func(_ context.Context, _ clientes.VinculoListParams) (clientes.VinculoListResult, error) {
				return clientes.VinculoListResult{}, nil
			},
		}

		rr := httptest.NewRecorder()
		newVinculoRouter(clientes.NewVinculoHandler(svc)).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/vinculos/", nil))

		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rr.Code)
		}
		if body := strings.TrimSpace(rr.Body.String()); !strings.Contains(body, `"items":[]`) {
			t.Fatalf("want items vazio, got %s", body)
		}
	})

	// O nome do cliente entra no cursor e pode conter o separador; a decodificacao
	// corta no ultimo "|" porque o id nunca tem um.
	t.Run("cursor sobrevive a nome com o separador", func(t *testing.T) {
		var received clientes.VinculoListParams
		nome := "Ana | Maria"
		primeiro := fakeVinculoService{
			listFn: func(_ context.Context, _ clientes.VinculoListParams) (clientes.VinculoListResult, error) {
				return clientes.VinculoListResult{
					NextCursor: &clientes.VinculoCursor{ClienteNome: nome, ID: 42},
					HasMore:    true,
				}, nil
			},
		}
		rr := httptest.NewRecorder()
		newVinculoRouter(clientes.NewVinculoHandler(primeiro)).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/vinculos/", nil))

		var resp struct {
			NextCursor string `json:"next_cursor"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("invalid json: %v", err)
		}

		segundo := fakeVinculoService{
			listFn: func(_ context.Context, params clientes.VinculoListParams) (clientes.VinculoListResult, error) {
				received = params
				return clientes.VinculoListResult{}, nil
			},
		}
		rr2 := httptest.NewRecorder()
		newVinculoRouter(clientes.NewVinculoHandler(segundo)).ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/vinculos/?cursor="+resp.NextCursor, nil))

		if rr2.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", rr2.Code, rr2.Body.String())
		}
		if received.Cursor == nil || received.Cursor.ClienteNome != nome || received.Cursor.ID != 42 {
			t.Fatalf("cursor nao sobreviveu ao roundtrip: %+v", received.Cursor)
		}
	})

	t.Run("parametro invalido vira 400 sem chamar o service", func(t *testing.T) {
		for _, query := range []string{"?limit=abc", "?cursor=***"} {
			svc := fakeVinculoService{
				listFn: func(_ context.Context, _ clientes.VinculoListParams) (clientes.VinculoListResult, error) {
					t.Fatalf("service nao pode ser chamado para %q", query)
					return clientes.VinculoListResult{}, nil
				},
			}
			rr := httptest.NewRecorder()
			newVinculoRouter(clientes.NewVinculoHandler(svc)).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/vinculos/"+query, nil))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("%s: want 400, got %d", query, rr.Code)
			}
		}
	})

	t.Run("translates store failure to 500", func(t *testing.T) {
		svc := fakeVinculoService{
			listFn: func(_ context.Context, _ clientes.VinculoListParams) (clientes.VinculoListResult, error) {
				return clientes.VinculoListResult{}, errors.New("db")
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
