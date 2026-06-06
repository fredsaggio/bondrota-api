package rotasdinamicas_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
	"github.com/go-chi/chi/v5"
)

type fakeRotaDinamicaService struct {
	createFn      func(ctx context.Context, input rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error)
	getByViagemFn func(ctx context.Context, viagemID int64) (*rotasdinamicas.RotaDinamicaComDestinos, error)
	deleteFn      func(ctx context.Context, viagemID int64) error
}

func (s fakeRotaDinamicaService) Create(ctx context.Context, input rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
	return s.createFn(ctx, input)
}

func (s fakeRotaDinamicaService) GetByViagem(ctx context.Context, viagemID int64) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
	return s.getByViagemFn(ctx, viagemID)
}

func (s fakeRotaDinamicaService) DeleteByViagem(ctx context.Context, viagemID int64) error {
	return s.deleteFn(ctx, viagemID)
}

func newRotaDinamicaRouter(h *rotasdinamicas.RotaDinamicaHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/viagens/{viagemID}/rota-dinamica", h.Create)
	r.Get("/viagens/{viagemID}/rota-dinamica", h.GetByViagem)
	r.Delete("/viagens/{viagemID}/rota-dinamica", h.DeleteByViagem)
	return r
}

func rotaBody(v any) *bytes.Buffer {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(v)
	return &buf
}

func validRotaBody() map[string]any {
	return map[string]any{
		"origem": map[string]any{
			"nome":      "Ultima parada",
			"latitude":  -9.780000,
			"longitude": -36.350000,
		},
		"destino_final": map[string]any{
			"nome":      "UFAL",
			"latitude":  -9.558000,
			"longitude": -35.775000,
		},
		"distancia_metros": 100000,
		"duracao_segundos": 7200,
		"geometry": map[string]any{
			"type":        "LineString",
			"coordinates": []any{[]float64{-36.35, -9.78}, []float64{-35.775, -9.558}},
		},
		"expires_at": "2026-09-10T00:00:00Z",
		"destinos": []map[string]any{
			{"destino_id": 5},
			{"destino_id": 8},
		},
	}
}

func TestRotaDinamicaHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       any
		svc        fakeRotaDinamicaService
		wantStatus int
	}{
		{
			name: "success",
			path: "/viagens/10/rota-dinamica",
			body: validRotaBody(),
			svc: fakeRotaDinamicaService{
				createFn: func(_ context.Context, input rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
					if input.ViagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", input.ViagemID)
					}
					if input.Destinos[0].DestinoID != 5 || input.Destinos[1].DestinoID != 8 {
						t.Fatalf("unexpected destinos: %+v", input.Destinos)
					}
					input.Provider = "osrm"
					return sampleRota(input), nil
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid viagem id",
			path:       "/viagens/abc/rota-dinamica",
			body:       validRotaBody(),
			svc:        fakeRotaDinamicaService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid timestamp",
			path:       "/viagens/10/rota-dinamica",
			body:       map[string]any{"expires_at": "2026-09-10"},
			svc:        fakeRotaDinamicaService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "validation error",
			path: "/viagens/10/rota-dinamica",
			body: func() map[string]any {
				in := validRotaBody()
				in["distancia_metros"] = 0
				return in
			}(),
			svc: fakeRotaDinamicaService{
				createFn: func(_ context.Context, _ rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
					return nil, errors.Join(brerror.ErrInvalidInput, errors.New("distancia_metros must be greater than zero"))
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "already exists",
			path: "/viagens/10/rota-dinamica",
			body: validRotaBody(),
			svc: fakeRotaDinamicaService{
				createFn: func(_ context.Context, _ rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
					return nil, brerror.ErrAlreadyExists
				},
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "internal error",
			path: "/viagens/10/rota-dinamica",
			body: validRotaBody(),
			svc: fakeRotaDinamicaService{
				createFn: func(_ context.Context, _ rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
					return nil, errors.New("db")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rotasdinamicas.NewRotaDinamicaHandler(tc.svc)
			req := httptest.NewRequest(http.MethodPost, tc.path, rotaBody(tc.body))
			rr := httptest.NewRecorder()

			newRotaDinamicaRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRotaDinamicaHandler_CreateInvalidJSON(t *testing.T) {
	h := rotasdinamicas.NewRotaDinamicaHandler(fakeRotaDinamicaService{})
	req := httptest.NewRequest(http.MethodPost, "/viagens/10/rota-dinamica", strings.NewReader("{"))
	rr := httptest.NewRecorder()

	newRotaDinamicaRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestRotaDinamicaHandler_GetByViagem(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		svc        fakeRotaDinamicaService
		wantStatus int
	}{
		{
			name: "success",
			path: "/viagens/10/rota-dinamica",
			svc: fakeRotaDinamicaService{
				getByViagemFn: func(_ context.Context, viagemID int64) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
					if viagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", viagemID)
					}
					input := validRotaInput()
					input.Provider = "osrm"
					input.Destinos[0].Ordem = 1
					input.Destinos[1].Ordem = 2
					return sampleRota(input), nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			path:       "/viagens/abc/rota-dinamica",
			svc:        fakeRotaDinamicaService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			path: "/viagens/99/rota-dinamica",
			svc: fakeRotaDinamicaService{
				getByViagemFn: func(_ context.Context, _ int64) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
					return nil, brerror.ErrNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rotasdinamicas.NewRotaDinamicaHandler(tc.svc)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			newRotaDinamicaRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRotaDinamicaHandler_DeleteByViagem(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		svc        fakeRotaDinamicaService
		wantStatus int
	}{
		{
			name: "success",
			path: "/viagens/10/rota-dinamica",
			svc: fakeRotaDinamicaService{
				deleteFn: func(_ context.Context, viagemID int64) error {
					if viagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", viagemID)
					}
					return nil
				},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid id",
			path:       "/viagens/abc/rota-dinamica",
			svc:        fakeRotaDinamicaService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			path: "/viagens/99/rota-dinamica",
			svc: fakeRotaDinamicaService{
				deleteFn: func(_ context.Context, _ int64) error {
					return brerror.ErrNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rotasdinamicas.NewRotaDinamicaHandler(tc.svc)
			req := httptest.NewRequest(http.MethodDelete, tc.path, nil)
			rr := httptest.NewRecorder()

			newRotaDinamicaRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRotaDinamicaHandler_ResponseDates(t *testing.T) {
	h := rotasdinamicas.NewRotaDinamicaHandler(fakeRotaDinamicaService{
		getByViagemFn: func(_ context.Context, _ int64) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
			input := validRotaInput()
			input.Provider = "osrm"
			input.ExpiresAt = time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
			input.Destinos[0].Ordem = 1
			input.Destinos[1].Ordem = 2
			return sampleRota(input), nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/viagens/10/rota-dinamica", nil)
	rr := httptest.NewRecorder()

	newRotaDinamicaRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"expires_at":"2026-09-10T00:00:00Z"`) {
		t.Fatalf("expected RFC3339 expires_at, got %s", rr.Body.String())
	}
}
