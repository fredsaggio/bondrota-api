package viagens_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/go-chi/chi/v5"
)

type fakeViagemService struct {
	getFn      func(ctx context.Context, viagemID int64) (*viagens.ViagemComCiclo, error)
	listFn     func(ctx context.Context) ([]viagens.ViagemComCiclo, error)
	iniciarFn  func(ctx context.Context, viagemID int64) (*viagens.Viagem, error)
	concluirFn func(ctx context.Context, viagemID int64) (*viagens.Viagem, error)
	cancelarFn func(ctx context.Context, viagemID int64) (*viagens.Viagem, error)
}

func (s fakeViagemService) GetByID(ctx context.Context, viagemID int64) (*viagens.ViagemComCiclo, error) {
	return s.getFn(ctx, viagemID)
}

func (s fakeViagemService) List(ctx context.Context) ([]viagens.ViagemComCiclo, error) {
	return s.listFn(ctx)
}

func (s fakeViagemService) Iniciar(ctx context.Context, viagemID int64) (*viagens.Viagem, error) {
	return s.iniciarFn(ctx, viagemID)
}

func (s fakeViagemService) Concluir(ctx context.Context, viagemID int64) (*viagens.Viagem, error) {
	return s.concluirFn(ctx, viagemID)
}

func (s fakeViagemService) Cancelar(ctx context.Context, viagemID int64) (*viagens.Viagem, error) {
	return s.cancelarFn(ctx, viagemID)
}

type fakePresencaService struct {
	listReservasFn      func(ctx context.Context, viagemID int64) ([]viagens.ViagemReservaComReserva, error)
	atualizarPresencaFn func(ctx context.Context, viagemID, reservaID int64, status viagens.StatusPresencaViagem) (*viagens.ViagemReserva, error)
}

func (s fakePresencaService) ListReservasByViagem(ctx context.Context, viagemID int64) ([]viagens.ViagemReservaComReserva, error) {
	return s.listReservasFn(ctx, viagemID)
}

func (s fakePresencaService) AtualizarPresenca(ctx context.Context, viagemID, reservaID int64, status viagens.StatusPresencaViagem) (*viagens.ViagemReserva, error) {
	return s.atualizarPresencaFn(ctx, viagemID, reservaID, status)
}

func newViagemRouter(h *viagens.ViagemHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/viagens", h.List)
	r.Get("/viagens/{viagemID}", h.GetByID)
	r.Post("/viagens/{viagemID}/iniciar", h.Iniciar)
	r.Post("/viagens/{viagemID}/concluir", h.Concluir)
	r.Post("/viagens/{viagemID}/cancelar", h.Cancelar)
	r.Get("/viagens/{viagemID}/reservas", h.ListReservas)
	r.Put("/viagens/{viagemID}/reservas/{reservaID}/presenca", h.AtualizarPresenca)
	return r
}

func TestViagemHandler_List(t *testing.T) {
	h := viagens.NewViagemHandler(fakeViagemService{
		listFn: func(_ context.Context) ([]viagens.ViagemComCiclo, error) {
			return []viagens.ViagemComCiclo{sampleViagemComCiclo()}, nil
		},
	}, fakePresencaService{})

	req := httptest.NewRequest(http.MethodGet, "/viagens", nil)
	rr := httptest.NewRecorder()

	newViagemRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestViagemHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		svc        fakeViagemService
		wantStatus int
	}{
		{
			name: "success",
			path: "/viagens/10",
			svc: fakeViagemService{
				getFn: func(_ context.Context, viagemID int64) (*viagens.ViagemComCiclo, error) {
					if viagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", viagemID)
					}
					v := sampleViagemComCiclo()
					return &v, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			path:       "/viagens/abc",
			svc:        fakeViagemService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			path: "/viagens/99",
			svc: fakeViagemService{
				getFn: func(_ context.Context, _ int64) (*viagens.ViagemComCiclo, error) {
					return nil, brerror.ErrNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "internal error",
			path: "/viagens/10",
			svc: fakeViagemService{
				getFn: func(_ context.Context, _ int64) (*viagens.ViagemComCiclo, error) {
					return nil, errors.New("db")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := viagens.NewViagemHandler(tc.svc, fakePresencaService{})
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			newViagemRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestViagemHandler_StatusActions(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		svc        fakeViagemService
		wantStatus int
	}{
		{
			name:   "iniciar success",
			method: http.MethodPost,
			path:   "/viagens/10/iniciar",
			svc: fakeViagemService{
				iniciarFn: func(_ context.Context, viagemID int64) (*viagens.Viagem, error) {
					if viagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", viagemID)
					}
					v := sampleViagem()
					v.Status = viagens.StatusViagemEmAndamento
					return &v, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "iniciar invalid id",
			method:     http.MethodPost,
			path:       "/viagens/abc/iniciar",
			svc:        fakeViagemService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "concluir success",
			method: http.MethodPost,
			path:   "/viagens/10/concluir",
			svc: fakeViagemService{
				concluirFn: func(_ context.Context, viagemID int64) (*viagens.Viagem, error) {
					if viagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", viagemID)
					}
					v := sampleViagem()
					v.Status = viagens.StatusViagemConcluida
					return &v, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "cancelar conflict",
			method: http.MethodPost,
			path:   "/viagens/10/cancelar",
			svc: fakeViagemService{
				cancelarFn: func(_ context.Context, _ int64) (*viagens.Viagem, error) {
					return nil, brerror.ErrAlreadyExists
				},
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := viagens.NewViagemHandler(tc.svc, fakePresencaService{})
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			newViagemRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestViagemHandler_ListReservas(t *testing.T) {
	h := viagens.NewViagemHandler(fakeViagemService{}, fakePresencaService{
		listReservasFn: func(_ context.Context, viagemID int64) ([]viagens.ViagemReservaComReserva, error) {
			if viagemID != 10 {
				t.Fatalf("unexpected viagemID: %d", viagemID)
			}
			return []viagens.ViagemReservaComReserva{sampleViagemReservaComReserva()}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/viagens/10/reservas", nil)
	rr := httptest.NewRecorder()

	newViagemRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestViagemHandler_AtualizarPresenca(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       map[string]any
		svc        fakePresencaService
		wantStatus int
	}{
		{
			name: "success",
			path: "/viagens/10/reservas/20/presenca",
			body: map[string]any{"status_presenca": "embarcou"},
			svc: fakePresencaService{
				atualizarPresencaFn: func(_ context.Context, viagemID, reservaID int64, status viagens.StatusPresencaViagem) (*viagens.ViagemReserva, error) {
					if viagemID != 10 || reservaID != 20 || status != viagens.StatusPresencaEmbarcou {
						t.Fatalf("unexpected update: %d %d %s", viagemID, reservaID, status)
					}
					vr := sampleViagemReserva()
					vr.StatusPresenca = viagens.StatusPresencaEmbarcou
					return &vr, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid ids",
			path:       "/viagens/abc/reservas/20/presenca",
			body:       map[string]any{"status_presenca": "embarcou"},
			svc:        fakePresencaService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			path:       "/viagens/10/reservas/20/presenca",
			body:       nil,
			svc:        fakePresencaService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "service not found",
			path: "/viagens/10/reservas/20/presenca",
			body: map[string]any{"status_presenca": "faltou"},
			svc: fakePresencaService{
				atualizarPresencaFn: func(_ context.Context, _, _ int64, _ viagens.StatusPresencaViagem) (*viagens.ViagemReserva, error) {
					return nil, brerror.ErrNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := viagens.NewViagemHandler(fakeViagemService{}, tc.svc)
			var req *http.Request
			if tc.body == nil {
				req = httptest.NewRequest(http.MethodPut, tc.path, bytesBuffer("{"))
			} else {
				req = httptest.NewRequest(http.MethodPut, tc.path, body(tc.body))
			}
			rr := httptest.NewRecorder()

			newViagemRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func bytesBuffer(s string) *strings.Reader {
	return strings.NewReader(s)
}
