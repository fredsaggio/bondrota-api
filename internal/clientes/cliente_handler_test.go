package clientes_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/clientes"
)

type fakeClienteService struct {
	loginFn  func(ctx context.Context, cpf, senha string) (string, error)
	createFn func(ctx context.Context, input clientes.ClienteInput) (*clientes.Cliente, error)
	getFn    func(ctx context.Context, clienteID int64) (*clientes.ClienteComVinculos, error)
	listFn   func(ctx context.Context, params clientes.ClienteListParams) (clientes.ClienteListResult, error)
	resumoFn func(ctx context.Context) (clientes.ClienteResumo, error)
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

func (s fakeClienteService) List(ctx context.Context, params clientes.ClienteListParams) (clientes.ClienteListResult, error) {
	return s.listFn(ctx, params)
}

func (s fakeClienteService) Resumo(ctx context.Context) (clientes.ClienteResumo, error) {
	return s.resumoFn(ctx)
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
	r.Get("/clientes/resumo", h.Resumo)
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
		"nome":                    " Maria Souza ",
		"cpf":                     "123.456.789-09",
		"senha":                   "secret",
		"data_nasc":               "2000-01-02",
		"documento_identificacao": "clientes/_novo/teste/documento-identificacao.pdf",
		"comprovante_residencia":  "clientes/_novo/teste/comprovante-residencia.pdf",
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
					if input.Nome != "MARIA SOUZA" || input.CPF != "12345678909" {
						t.Fatalf("unexpected input: %+v", input)
					}
					return sampleCliente(), nil
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid date",
			body:       body(map[string]any{"nome": "Maria Souza", "cpf": "123.456.789-01", "senha": "secret", "data_nasc": "02-01-2000"}),
			svc:        fakeClienteService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "nome com digito → 400",
			body:       body(map[string]any{"nome": "Maria 2", "cpf": "123.456.789-01", "senha": "secret", "data_nasc": "2000-01-02"}),
			svc:        fakeClienteService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "cpf curto → 400",
			body:       body(map[string]any{"nome": "Maria Souza", "cpf": "123", "senha": "secret", "data_nasc": "2000-01-02"}),
			svc:        fakeClienteService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "documento de identificacao obrigatorio",
			body:       body(map[string]any{"nome": "Maria Souza", "cpf": "123.456.789-09", "senha": "secret", "data_nasc": "2000-01-02", "comprovante_residencia": "residencia.pdf"}),
			svc:        fakeClienteService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "comprovante de residencia obrigatorio",
			body:       body(map[string]any{"nome": "Maria Souza", "cpf": "123.456.789-09", "senha": "secret", "data_nasc": "2000-01-02", "documento_identificacao": "identidade.pdf"}),
			svc:        fakeClienteService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "documento com extensao invalida",
			body:       body(map[string]any{"nome": "Maria Souza", "cpf": "123.456.789-09", "senha": "secret", "data_nasc": "2000-01-02", "documento_identificacao": "documento.exe", "comprovante_residencia": "residencia.pdf"}),
			svc:        fakeClienteService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "comprovante com path inseguro",
			body:       body(map[string]any{"nome": "Maria Souza", "cpf": "123.456.789-09", "senha": "secret", "data_nasc": "2000-01-02", "documento_identificacao": "identidade.pdf", "comprovante_residencia": "clientes/1/../residencia.pdf"}),
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

func TestClienteHandler_TelefoneDuplicado(t *testing.T) {
	duplicateError := fmt.Errorf("db/clienteStore: %w", &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "telefones_cadastrados_pkey",
	})
	const wantMessage = "Já existe outro cadastro com este telefone.\n"

	t.Run("create retorna conflito claro", func(t *testing.T) {
		h := clientes.NewClienteHandler(fakeClienteService{
			createFn: func(_ context.Context, _ clientes.ClienteInput) (*clientes.Cliente, error) {
				return nil, duplicateError
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/clientes", body(map[string]any{
			"nome": "Maria Souza", "cpf": "12345678909", "senha": "secret",
			"telefone": "82999990000", "data_nasc": "2000-01-02",
			"documento_identificacao": "clientes/_novo/teste/documento-identificacao.pdf",
			"comprovante_residencia":  "clientes/_novo/teste/comprovante-residencia.pdf",
		}))
		rr := httptest.NewRecorder()

		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict || rr.Body.String() != wantMessage {
			t.Fatalf("want 409 with clear message, got %d: %q", rr.Code, rr.Body.String())
		}
	})

	t.Run("update retorna conflito claro", func(t *testing.T) {
		h := clientes.NewClienteHandler(fakeClienteService{
			updateFn: func(_ context.Context, _ int64, _ func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error) {
				return nil, duplicateError
			},
		})
		req := httptest.NewRequest(http.MethodPut, "/clientes/1", body(map[string]any{
			"telefone": "82999990000",
		}))
		rr := httptest.NewRecorder()

		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict || rr.Body.String() != wantMessage {
			t.Fatalf("want 409 with clear message, got %d: %q", rr.Code, rr.Body.String())
		}
	})
}

// fakeArquivoMovedor grava a ultima chamada de MoveObject, para os testes
// afirmarem de onde para onde o arquivo foi movido.
type fakeArquivoMovedor struct {
	bucket, from, to string
	err              error
	calls            []movimentoArquivo
}

type movimentoArquivo struct {
	bucket string
	from   string
	to     string
}

func (f *fakeArquivoMovedor) MoveObject(_ context.Context, bucket, from, to string) error {
	f.bucket, f.from, f.to = bucket, from, to
	f.calls = append(f.calls, movimentoArquivo{bucket: bucket, from: from, to: to})
	return f.err
}

func TestClienteHandler_Create_OrganizaDocumentos(t *testing.T) {
	body := body(map[string]any{
		"nome":                    " Maria Souza ",
		"cpf":                     "123.456.789-09",
		"senha":                   "secret",
		"data_nasc":               "2000-01-02",
		"documento_identificacao": "clientes/_novo/xyz789/documento-identificacao.png",
		"comprovante_residencia":  "clientes/_novo/xyz789/comprovante-residencia.pdf",
	})

	svc := fakeClienteService{
		createFn: func(_ context.Context, input clientes.ClienteInput) (*clientes.Cliente, error) {
			if input.DocumentoIdentificacao != "clientes/_novo/xyz789/documento-identificacao.png" || input.ComprovanteResidencia != "clientes/_novo/xyz789/comprovante-residencia.pdf" {
				t.Fatalf("unexpected documents on create: %+v", input)
			}
			return &clientes.Cliente{ID: 7, PublicID: "cli_012345678901234567890", DocumentoIdentificacao: input.DocumentoIdentificacao, ComprovanteResidencia: input.ComprovanteResidencia}, nil
		},
		updateFn: func(_ context.Context, clienteID int64, updateFunc func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error) {
			if clienteID != 7 {
				t.Fatalf("unexpected clienteID no update: %d", clienteID)
			}
			c := &clientes.Cliente{ID: 7, PublicID: "cli_012345678901234567890", DocumentoIdentificacao: "clientes/_novo/xyz789/documento-identificacao.png", ComprovanteResidencia: "clientes/_novo/xyz789/comprovante-residencia.pdf"}
			changed, err := updateFunc(c)
			if err != nil || !changed || c.DocumentoIdentificacao != "clientes/cli_012345678901234567890/documento-identificacao.png" || c.ComprovanteResidencia != "clientes/cli_012345678901234567890/comprovante-residencia.pdf" {
				t.Fatalf("update nao organizou os documentos corretamente: changed=%v err=%v cliente=%+v", changed, err, c)
			}
			return c, nil
		},
	}

	mover := &fakeArquivoMovedor{}
	h := clientes.NewClienteHandler(svc, mover)
	req := httptest.NewRequest(http.MethodPost, "/clientes", body)
	rr := httptest.NewRecorder()
	newClienteRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d — %s", rr.Code, rr.Body.String())
	}
	if len(mover.calls) != 2 {
		t.Fatalf("want two moves, got %+v", mover.calls)
	}
	wantMoves := []movimentoArquivo{
		{bucket: "documentos", from: "clientes/_novo/xyz789/documento-identificacao.png", to: "clientes/cli_012345678901234567890/documento-identificacao.png"},
		{bucket: "documentos", from: "clientes/_novo/xyz789/comprovante-residencia.pdf", to: "clientes/cli_012345678901234567890/comprovante-residencia.pdf"},
	}
	for i, want := range wantMoves {
		if mover.calls[i] != want {
			t.Fatalf("move %d: want %+v, got %+v", i, want, mover.calls[i])
		}
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["documento_identificacao"] != "clientes/cli_012345678901234567890/documento-identificacao.png" || resp["comprovante_residencia"] != "clientes/cli_012345678901234567890/comprovante-residencia.pdf" {
		t.Fatalf("want organized documents in response, got %+v", resp)
	}
}

func TestClienteHandler_Create_FalhaAoOrganizarDocumentosNaoDerrubaCriacao(t *testing.T) {
	body := body(map[string]any{
		"nome":                    " Maria Souza ",
		"cpf":                     "123.456.789-09",
		"senha":                   "secret",
		"data_nasc":               "2000-01-02",
		"documento_identificacao": "clientes/_novo/xyz789/documento-identificacao.png",
		"comprovante_residencia":  "clientes/_novo/xyz789/comprovante-residencia.pdf",
	})

	svc := fakeClienteService{
		createFn: func(_ context.Context, input clientes.ClienteInput) (*clientes.Cliente, error) {
			return &clientes.Cliente{ID: 7, PublicID: "cli_012345678901234567890", DocumentoIdentificacao: input.DocumentoIdentificacao, ComprovanteResidencia: input.ComprovanteResidencia}, nil
		},
		updateFn: func(context.Context, int64, func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error) {
			t.Fatal("update nao deveria ser chamado quando mover falha")
			return nil, nil
		},
	}

	mover := &fakeArquivoMovedor{err: errors.New("supabase indisponivel")}
	h := clientes.NewClienteHandler(svc, mover)
	req := httptest.NewRequest(http.MethodPost, "/clientes", body)
	rr := httptest.NewRecorder()
	newClienteRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("falha ao mover nao deveria derrubar a criacao: want 201, got %d — %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["documento_identificacao"] != "clientes/_novo/xyz789/documento-identificacao.png" || resp["comprovante_residencia"] != "clientes/_novo/xyz789/comprovante-residencia.pdf" {
		t.Fatalf("documents should remain in staging paths after failure, got %+v", resp)
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
			listFn: func(_ context.Context, _ clientes.ClienteListParams) (clientes.ClienteListResult, error) {
				return clientes.ClienteListResult{
					Items:        []clientes.Cliente{*sampleCliente()},
					NextCursorID: 7,
					HasMore:      true,
				}, nil
			},
		})

		req := httptest.NewRequest(http.MethodGet, "/clientes", nil)
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want %d, got %d", http.StatusOK, rr.Code)
		}
		var resp struct {
			Items      []map[string]any `json:"items"`
			NextCursor string           `json:"next_cursor"`
			HasMore    bool             `json:"has_more"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(resp.Items) != 1 || resp.NextCursor == "" || !resp.HasMore {
			t.Fatalf("envelope paginado inesperado: %+v", resp)
		}
	})

	t.Run("list repassa busca, limit e cursor", func(t *testing.T) {
		var received clientes.ClienteListParams
		h := clientes.NewClienteHandler(fakeClienteService{
			listFn: func(_ context.Context, params clientes.ClienteListParams) (clientes.ClienteListResult, error) {
				received = params
				return clientes.ClienteListResult{}, nil
			},
		})

		// Cursor precisa ser o mesmo formato opaco que a resposta devolve.
		cursor := base64.RawURLEncoding.EncodeToString([]byte("42"))
		req := httptest.NewRequest(http.MethodGet, "/clientes?q=maria&limit=10&cursor="+cursor, nil)
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}
		if received.Busca != "maria" || received.Limit != 10 || received.CursorID != 42 {
			t.Fatalf("params nao chegaram no service: %+v", received)
		}
	})

	t.Run("list rejeita parametros invalidos sem chamar o service", func(t *testing.T) {
		for _, query := range []string{"?limit=abc", "?limit=0", "?cursor=***"} {
			h := clientes.NewClienteHandler(fakeClienteService{
				listFn: func(_ context.Context, _ clientes.ClienteListParams) (clientes.ClienteListResult, error) {
					t.Fatalf("service nao pode ser chamado para %q", query)
					return clientes.ClienteListResult{}, nil
				},
			})

			rr := httptest.NewRecorder()
			newClienteRouter(h).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/clientes"+query, nil))

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("%s: want 400, got %d", query, rr.Code)
			}
		}
	})

	t.Run("resumo devolve o total", func(t *testing.T) {
		h := clientes.NewClienteHandler(fakeClienteService{
			resumoFn: func(_ context.Context) (clientes.ClienteResumo, error) {
				return clientes.ClienteResumo{Total: 137}, nil
			},
		})

		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/clientes/resumo", nil))

		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp struct {
			Total int64 `json:"total"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Total != 137 {
			t.Fatalf("want 137, got %d", resp.Total)
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

	t.Run("telefone vazio explicito limpa o campo", func(t *testing.T) {
		var captured *clientes.Cliente
		h := clientes.NewClienteHandler(fakeClienteService{
			updateFn: func(_ context.Context, _ int64, updateFunc func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error) {
				c := sampleCliente()
				changed, err := updateFunc(c)
				if err != nil || !changed {
					t.Fatalf("expected changed without error, changed=%v err=%v", changed, err)
				}
				captured = c
				return c, nil
			},
		})

		req := httptest.NewRequest(http.MethodPut, "/clientes/1", body(map[string]any{"telefone": ""}))
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}
		if captured.Telefone != "" {
			t.Fatalf("want telefone cleared, got %q", captured.Telefone)
		}
	})

	t.Run("telefone ausente preserva o valor atual", func(t *testing.T) {
		var captured *clientes.Cliente
		h := clientes.NewClienteHandler(fakeClienteService{
			updateFn: func(_ context.Context, _ int64, updateFunc func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error) {
				c := sampleCliente()
				changed, err := updateFunc(c)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if changed {
					t.Fatal("want changed=false when only nome is sent unchanged and no other field is present")
				}
				captured = c
				return c, nil
			},
		})

		req := httptest.NewRequest(http.MethodPut, "/clientes/1", body(map[string]any{}))
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}
		if captured.Telefone != "82999999999" {
			t.Fatalf("want telefone untouched, got %q", captured.Telefone)
		}
	})

	t.Run("documento vazio explicito e rejeitado", func(t *testing.T) {
		var captured *clientes.Cliente
		h := clientes.NewClienteHandler(fakeClienteService{
			updateFn: func(_ context.Context, _ int64, updateFunc func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error) {
				c := sampleCliente()
				_, err := updateFunc(c)
				if err != nil {
					return nil, err
				}
				captured = c
				return c, nil
			},
		})

		req := httptest.NewRequest(http.MethodPut, "/clientes/1", body(map[string]any{"documento_identificacao": ""}))
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("want %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
		}
		if captured != nil {
			t.Fatalf("invalid document must not be persisted: %+v", captured)
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

	t.Run("cliente com reserva alocada a viagem vira 409", func(t *testing.T) {
		h := clientes.NewClienteHandler(fakeClienteService{
			deleteFn: func(_ context.Context, _ int64) error {
				// A cascata chega em reservas, mas viagem_reservas usa ON DELETE RESTRICT.
				return fmt.Errorf("db/clienteStore.Delete: %w", &pgconn.PgError{Code: "23503", ConstraintName: "viagem_reservas_reserva_id_fkey"})
			},
		})

		req := httptest.NewRequest(http.MethodDelete, "/clientes/1", nil)
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Fatalf("want %d, got %d", http.StatusConflict, rr.Code)
		}
	})

	t.Run("falha generica do banco continua 500", func(t *testing.T) {
		h := clientes.NewClienteHandler(fakeClienteService{
			deleteFn: func(_ context.Context, _ int64) error { return errors.New("connection refused") },
		})

		req := httptest.NewRequest(http.MethodDelete, "/clientes/1", nil)
		rr := httptest.NewRecorder()
		newClienteRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("want %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})
}
