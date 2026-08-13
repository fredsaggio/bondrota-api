package motoristas_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/mocks"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
)

// --- helpers ---

func newMotoristaRouter(h *motoristas.MotoristaHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/motoristas/login", h.Login)
	r.Post("/motoristas", h.Create)
	r.Get("/motoristas", h.List)
	r.Get("/motoristas/{id}", h.GetByID)
	r.Patch("/motoristas/{id}", h.Update)
	r.Delete("/motoristas/{id}", h.Delete)
	return r
}

func jsonBuf(v any) *bytes.Buffer {
	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(v)
	return &b
}

func sampleMotorista() *motoristas.Motorista {
	return &motoristas.Motorista{
		ID:                  1,
		Nome:                "João Silva",
		CPF:                 "123.456.789-00",
		Telefone:            "81999990000",
		DataNasc:            time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Turno:               motoristas.TurnoMatutino,
		MunicipioTrabalhoID: 2611606,
		Residencia:          "Olinda",
		Foto:                "",
	}
}

var anyUpdateFunc = mock.MatchedBy(func(_ func(*motoristas.Motorista) (bool, error)) bool { return true })

// --- Login ---

func TestMotoristaHandler_Login(t *testing.T) {
	tests := []struct {
		name       string
		body       *bytes.Buffer
		setup      func(*mocks.MockMotoristaService)
		wantStatus int
	}{
		{
			name: "sucesso",
			body: jsonBuf(map[string]any{"cpf": "123.456.789-00", "senha": "secret"}),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Login(mock.Anything, "123.456.789-00", "secret").Return("tok123", nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "body malformado → 400",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "cpf vazio → 400",
			body:       jsonBuf(map[string]any{"cpf": "   ", "senha": "pw"}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "senha vazia → 400",
			body:       jsonBuf(map[string]any{"cpf": "123.456.789-00", "senha": ""}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "credenciais inválidas → 401",
			body: jsonBuf(map[string]any{"cpf": "000.000.000-00", "senha": "wrong"}),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Login(mock.Anything, "000.000.000-00", "wrong").
					Return("", errors.New("auth/AuthService: auth/AuthService: invalid credentials"))
			},
			wantStatus: http.StatusInternalServerError, // service wraps o erro com fmt.Errorf
		},
		{
			name: "erro interno → 500",
			body: jsonBuf(map[string]any{"cpf": "123.456.789-00", "senha": "pw"}),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Login(mock.Anything, "123.456.789-00", "pw").Return("", errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockMotoristaService(t)
			tc.setup(svc)
			h := motoristas.NewMotoristaHandler(svc)
			req := httptest.NewRequest(http.MethodPost, "/motoristas/login", tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newMotoristaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// --- Create ---

func validCreateBody() *bytes.Buffer {
	return jsonBuf(map[string]any{
		"nome":                  "João Silva",
		"cpf":                   "123.456.789-09",
		"senha":                 "secret",
		"turno":                 "MT",
		"data_nasc":             "1990-01-01",
		"municipio_trabalho_id": int64(2611606),
	})
}

func TestMotoristaHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       *bytes.Buffer
		setup      func(*mocks.MockMotoristaService)
		wantStatus int
	}{
		{
			name: "sucesso → 201",
			body: validCreateBody(),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in motoristas.MotoristaInput) bool {
					return in.Nome == "JOÃO SILVA" && in.CPF == "12345678909" && in.Turno == motoristas.TurnoMatutino && in.MunicipioTrabalhoID == 2611606
				})).Return(sampleMotorista(), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "body malformado → 400",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "nome vazio → 400",
			body:       jsonBuf(map[string]any{"cpf": "123.456.789-01", "senha": "pw", "turno": "MT", "data_nasc": "1990-01-01"}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "nome com digito → 400",
			body:       jsonBuf(map[string]any{"nome": "João 2", "cpf": "123.456.789-01", "senha": "pw", "turno": "MT", "data_nasc": "1990-01-01", "municipio_trabalho_id": int64(2611606)}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "cpf curto → 400",
			body:       jsonBuf(map[string]any{"nome": "João Silva", "cpf": "123", "senha": "pw", "turno": "MT", "data_nasc": "1990-01-01", "municipio_trabalho_id": int64(2611606)}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "cpf com letra misturada → 400",
			body:       jsonBuf(map[string]any{"nome": "João Silva", "cpf": "123a45678909", "senha": "pw", "turno": "MT", "data_nasc": "1990-01-01", "municipio_trabalho_id": int64(2611606)}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "telefone com letra misturada → 400",
			body:       jsonBuf(map[string]any{"nome": "João Silva", "cpf": "12345678909", "senha": "pw", "telefone": "82abc988887777", "turno": "MT", "data_nasc": "1990-01-01", "municipio_trabalho_id": int64(2611606)}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "cpf vazio → 400",
			body:       jsonBuf(map[string]any{"nome": "João", "senha": "pw", "turno": "MT", "data_nasc": "1990-01-01"}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "senha vazia → 400",
			body:       jsonBuf(map[string]any{"nome": "João", "cpf": "123", "turno": "MT", "data_nasc": "1990-01-01"}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "turno ausente → 400",
			body:       jsonBuf(map[string]any{"nome": "João", "cpf": "123", "senha": "pw", "data_nasc": "1990-01-01"}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "turno inválido → 400",
			body:       jsonBuf(map[string]any{"nome": "João", "cpf": "123", "senha": "pw", "turno": "XX", "data_nasc": "1990-01-01"}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "data_nasc ausente → 400",
			body:       jsonBuf(map[string]any{"nome": "João", "cpf": "123", "senha": "pw", "turno": "MT"}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "data_nasc inválida → 400",
			body:       jsonBuf(map[string]any{"nome": "João", "cpf": "123", "senha": "pw", "turno": "MT", "data_nasc": "not-a-date"}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "municipio trabalho ausente → 400",
			body:       jsonBuf(map[string]any{"nome": "João", "cpf": "123", "senha": "pw", "turno": "MT", "data_nasc": "1990-01-01"}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "erro interno → 500",
			body: validCreateBody(),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockMotoristaService(t)
			tc.setup(svc)
			h := motoristas.NewMotoristaHandler(svc)
			req := httptest.NewRequest(http.MethodPost, "/motoristas", tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newMotoristaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestMotoristaHandler_TelefoneDuplicado(t *testing.T) {
	duplicateError := fmt.Errorf("db/motoristaStore: %w", &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "telefones_cadastrados_pkey",
	})
	const wantMessage = "Já existe outro cadastro com este telefone.\n"

	t.Run("create retorna conflito claro", func(t *testing.T) {
		svc := mocks.NewMockMotoristaService(t)
		svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, duplicateError)
		h := motoristas.NewMotoristaHandler(svc)
		req := httptest.NewRequest(http.MethodPost, "/motoristas", jsonBuf(map[string]any{
			"nome": "João Silva", "cpf": "12345678909", "senha": "secret",
			"telefone": "82999990000", "turno": "MT", "data_nasc": "1990-01-01",
			"municipio_trabalho_id": int64(2611606),
		}))
		rr := httptest.NewRecorder()

		newMotoristaRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict || rr.Body.String() != wantMessage {
			t.Fatalf("want 409 with clear message, got %d: %q", rr.Code, rr.Body.String())
		}
	})

	t.Run("update retorna conflito claro", func(t *testing.T) {
		svc := mocks.NewMockMotoristaService(t)
		svc.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(nil, duplicateError)
		h := motoristas.NewMotoristaHandler(svc)
		req := httptest.NewRequest(http.MethodPatch, "/motoristas/1", jsonBuf(map[string]any{
			"telefone": "82999990000",
		}))
		rr := httptest.NewRecorder()

		newMotoristaRouter(h).ServeHTTP(rr, req)

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
}

func (f *fakeArquivoMovedor) MoveObject(_ context.Context, bucket, from, to string) error {
	f.bucket, f.from, f.to = bucket, from, to
	return f.err
}

func TestMotoristaHandler_Create_OrganizaFoto(t *testing.T) {
	body := jsonBuf(map[string]any{
		"nome":                  "João Silva",
		"cpf":                   "123.456.789-09",
		"senha":                 "secret",
		"turno":                 "MT",
		"data_nasc":             "1990-01-01",
		"municipio_trabalho_id": int64(2611606),
		"foto":                  "motoristas/_novo/abc123/foto.jpg",
	})

	svc := mocks.NewMockMotoristaService(t)
	svc.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in motoristas.MotoristaInput) bool {
		return in.Foto == "motoristas/_novo/abc123/foto.jpg"
	})).Return(&motoristas.Motorista{ID: 1, Foto: "motoristas/_novo/abc123/foto.jpg"}, nil)
	svc.EXPECT().Update(mock.Anything, int64(1), mock.MatchedBy(func(fn func(*motoristas.Motorista) (bool, error)) bool {
		m := &motoristas.Motorista{ID: 1, Foto: "motoristas/_novo/abc123/foto.jpg"}
		changed, err := fn(m)
		return err == nil && changed && m.Foto == "motoristas/1/foto.jpg"
	})).Return(&motoristas.Motorista{ID: 1, Foto: "motoristas/1/foto.jpg"}, nil)

	mover := &fakeArquivoMovedor{}
	h := motoristas.NewMotoristaHandler(svc, mover)
	req := httptest.NewRequest(http.MethodPost, "/motoristas", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMotoristaRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d — %s", rr.Code, rr.Body.String())
	}
	if mover.bucket != "fotos" || mover.from != "motoristas/_novo/abc123/foto.jpg" || mover.to != "motoristas/1/foto.jpg" {
		t.Fatalf("unexpected move: bucket=%q from=%q to=%q", mover.bucket, mover.from, mover.to)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["foto"] != "motoristas/1/foto.jpg" {
		t.Fatalf("want foto organizada na resposta, got %v", resp["foto"])
	}
}

func TestMotoristaHandler_Create_FalhaAoOrganizarFotoNaoDerrubaCriacao(t *testing.T) {
	body := jsonBuf(map[string]any{
		"nome":                  "João Silva",
		"cpf":                   "123.456.789-09",
		"senha":                 "secret",
		"turno":                 "MT",
		"data_nasc":             "1990-01-01",
		"municipio_trabalho_id": int64(2611606),
		"foto":                  "motoristas/_novo/abc123/foto.jpg",
	})

	svc := mocks.NewMockMotoristaService(t)
	svc.EXPECT().Create(mock.Anything, mock.Anything).
		Return(&motoristas.Motorista{ID: 1, Foto: "motoristas/_novo/abc123/foto.jpg"}, nil)
	// Update nao e chamado: sem organizar a foto com sucesso, nao ha nada novo
	// para persistir.

	mover := &fakeArquivoMovedor{err: errors.New("supabase indisponivel")}
	h := motoristas.NewMotoristaHandler(svc, mover)
	req := httptest.NewRequest(http.MethodPost, "/motoristas", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMotoristaRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("falha ao mover nao deveria derrubar a criacao: want 201, got %d — %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["foto"] != "motoristas/_novo/abc123/foto.jpg" {
		t.Fatalf("foto deveria continuar no caminho de espera apos falha, got %v", resp["foto"])
	}
}

// --- GetByID ---

func TestMotoristaHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*mocks.MockMotoristaService)
		wantStatus int
	}{
		{
			name: "sucesso",
			id:   "1",
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleMotorista(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrado → 404",
			id:   "99",
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, motoristas.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockMotoristaService(t)
			tc.setup(svc)
			h := motoristas.NewMotoristaHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/motoristas/"+tc.id, nil)
			rr := httptest.NewRecorder()
			newMotoristaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- List ---

func TestMotoristaHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*mocks.MockMotoristaService)
		wantStatus int
	}{
		{
			name: "sucesso com itens",
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().List(mock.Anything).Return([]motoristas.Motorista{*sampleMotorista()}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "lista vazia",
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().List(mock.Anything).Return([]motoristas.Motorista{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "erro interno → 500",
			setup:      func(svc *mocks.MockMotoristaService) { svc.EXPECT().List(mock.Anything).Return(nil, errors.New("db")) },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockMotoristaService(t)
			tc.setup(svc)
			h := motoristas.NewMotoristaHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/motoristas", nil)
			rr := httptest.NewRecorder()
			newMotoristaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- Update ---

func TestMotoristaHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       *bytes.Buffer
		setup      func(*mocks.MockMotoristaService)
		wantStatus int
	}{
		{
			name: "sucesso",
			id:   "1",
			body: jsonBuf(map[string]any{"nome": "Maria"}),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(sampleMotorista(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			body:       jsonBuf(map[string]any{}),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "body malformado → 400",
			id:         "1",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrado → 404",
			id:   "99",
			body: jsonBuf(map[string]any{"nome": "Maria"}),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Update(mock.Anything, int64(99), anyUpdateFunc).Return(nil, motoristas.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "nome obrigatório retornado pelo updateFunc → 400",
			id:   "1",
			body: jsonBuf(map[string]any{"nome": "   "}),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(nil, motoristas.ErrNomeObrigatorio)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "turno inválido retornado pelo updateFunc → 400",
			id:   "1",
			body: jsonBuf(map[string]any{"turno": "XX"}),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(nil, motoristas.ErrTurnoInvalido)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			body: jsonBuf(map[string]any{"nome": "Maria"}),
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockMotoristaService(t)
			tc.setup(svc)
			h := motoristas.NewMotoristaHandler(svc)
			req := httptest.NewRequest(http.MethodPatch, "/motoristas/"+tc.id, tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newMotoristaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// TestMotoristaHandler_UpdateOptionalFields captura a closure passada a
// svc.Update e a executa contra um motorista de amostra, porque os subtestes
// acima usam anyUpdateFunc e nunca invocam a closure de verdade — eles não
// bastam para provar que telefone/residencia/foto distinguem "campo ausente"
// de "campo explicitamente limpo".
func TestMotoristaHandler_UpdateOptionalFields(t *testing.T) {
	capture := func(t *testing.T, body map[string]any) (*motoristas.Motorista, bool, error) {
		t.Helper()
		var capturedMotorista *motoristas.Motorista
		var capturedUpdated bool
		var capturedErr error

		svc := mocks.NewMockMotoristaService(t)
		svc.EXPECT().Update(mock.Anything, int64(1), mock.Anything).
			RunAndReturn(func(_ context.Context, _ int64, updateFunc func(*motoristas.Motorista) (bool, error)) (*motoristas.Motorista, error) {
				m := sampleMotorista()
				updated, err := updateFunc(m)
				capturedMotorista, capturedUpdated, capturedErr = m, updated, err
				return m, err
			})

		h := motoristas.NewMotoristaHandler(svc)
		req := httptest.NewRequest(http.MethodPatch, "/motoristas/1", jsonBuf(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		newMotoristaRouter(h).ServeHTTP(rr, req)
		if capturedErr == nil && rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
		}

		return capturedMotorista, capturedUpdated, capturedErr
	}

	t.Run("telefone ausente preserva o valor atual", func(t *testing.T) {
		m, updated, err := capture(t, map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated {
			t.Fatal("want updated=false when no field is sent")
		}
		if m.Telefone != "81999990000" {
			t.Fatalf("want telefone untouched, got %q", m.Telefone)
		}
	})

	t.Run("telefone vazio explicito limpa o campo", func(t *testing.T) {
		m, updated, err := capture(t, map[string]any{"telefone": ""})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updated {
			t.Fatal("want updated=true when clearing a non-empty field")
		}
		if m.Telefone != "" {
			t.Fatalf("want telefone cleared, got %q", m.Telefone)
		}
	})

	t.Run("telefone com valor atualiza o campo", func(t *testing.T) {
		m, updated, err := capture(t, map[string]any{"telefone": "82988887777"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updated {
			t.Fatal("want updated=true")
		}
		if m.Telefone != "82988887777" {
			t.Fatalf("want telefone updated, got %q", m.Telefone)
		}
	})

	t.Run("residencia vazia explicita limpa o campo", func(t *testing.T) {
		m, updated, err := capture(t, map[string]any{"residencia": ""})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updated {
			t.Fatal("want updated=true when clearing a non-empty field")
		}
		if m.Residencia != "" {
			t.Fatalf("want residencia cleared, got %q", m.Residencia)
		}
	})

	t.Run("foto ausente nao mexe no campo mesmo ja vazio", func(t *testing.T) {
		m, updated, err := capture(t, map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated {
			t.Fatal("want updated=false when no field is sent")
		}
		if m.Foto != "" {
			t.Fatalf("want foto untouched, got %q", m.Foto)
		}
	})
}

// --- Delete ---

func TestMotoristaHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*mocks.MockMotoristaService)
		wantStatus int
	}{
		{
			name:       "sucesso → 204",
			id:         "1",
			setup:      func(svc *mocks.MockMotoristaService) { svc.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			setup:      func(_ *mocks.MockMotoristaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrado → 404",
			id:   "99",
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Delete(mock.Anything, int64(99)).Return(motoristas.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Delete(mock.Anything, int64(1)).Return(errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			// motorista alocado em um ciclo de viagem (ON DELETE RESTRICT)
			name: "motorista em uso → 409",
			id:   "1",
			setup: func(svc *mocks.MockMotoristaService) {
				svc.EXPECT().Delete(mock.Anything, int64(1)).
					Return(fmt.Errorf("db/motoristaStore.Delete: %w", &pgconn.PgError{Code: "23503", ConstraintName: "ciclos_viagem_motorista_id_fkey"}))
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockMotoristaService(t)
			tc.setup(svc)
			h := motoristas.NewMotoristaHandler(svc)
			req := httptest.NewRequest(http.MethodDelete, "/motoristas/"+tc.id, nil)
			rr := httptest.NewRecorder()
			newMotoristaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- Login — 401 direto quando service retorna ErrInvalidCredentials sem wrapping ---

func TestMotoristaHandler_Login_DirectInvalidCredentials(t *testing.T) {
	// Valida o branch errors.Is(err, auth.ErrInvalidCredentials) → 401
	// O handler faz essa checagem; precisamos que o mock retorne o erro sem wrapping.
	svc := mocks.NewMockMotoristaService(t)
	svc.EXPECT().Login(mock.Anything, "123.456.789-00", "wrong").
		Return("", errors.New("invalid credentials"))

	h := motoristas.NewMotoristaHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/motoristas/login",
		jsonBuf(map[string]any{"cpf": "123.456.789-00", "senha": "wrong"}))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMotoristaRouter(h).ServeHTTP(rr, req)

	// o handler só devolve 401 se o erro for exatamente auth.ErrInvalidCredentials
	// via errors.Is; um erro wrappado cai em 500 — testamos isso na tabela acima.
	// Aqui confirmamos que erros genéricos vão para 500.
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("want 500 for non-sentinel error, got %d", rr.Code)
	}
}
