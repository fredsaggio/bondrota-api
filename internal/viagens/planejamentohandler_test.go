package viagens_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/go-chi/chi/v5"
)

type fakePlanejamentoService struct {
	planejarFn func(ctx context.Context, input viagens.PlanejamentoViagensInput, partidas map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error)
}

func (s fakePlanejamentoService) Planejar(ctx context.Context, input viagens.PlanejamentoViagensInput, partidas map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error) {
	return s.planejarFn(ctx, input, partidas)
}

func newPlanejamentoRouter(h *viagens.PlanejamentoHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/planejamentos/viagens", h.PlanejarViagens)
	return r
}

func validPlanejamentoBody() map[string]any {
	return map[string]any{
		"data_viagem":     "2026-06-10",
		"turno":           "NT",
		"cidade":          "Campo Alegre",
		"rota_interna_id": 2,
		"expires_at":      "2026-09-10T00:00:00Z",
		"partida_ida":     "2026-06-10T18:00:00Z",
		"partida_volta":   "2026-06-10T22:00:00Z",
	}
}

func TestPlanejamentoHandler_PlanejarViagens(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		svc        fakePlanejamentoService
		wantStatus int
	}{
		{
			name: "success",
			body: validPlanejamentoBody(),
			svc: fakePlanejamentoService{
				planejarFn: func(_ context.Context, input viagens.PlanejamentoViagensInput, partidas map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error) {
					if input.Cidade != "Campo Alegre" || input.Turno != viagens.TurnoNoturno {
						t.Fatalf("unexpected input: %+v", input)
					}
					if partidas[viagens.SentidoIda].IsZero() || partidas[viagens.SentidoVolta].IsZero() {
						t.Fatalf("expected both partidas: %+v", partidas)
					}
					ciclo := sampleCicloComViagens()
					return &viagens.PlanejamentoViagens{
						Ciclos:                  []viagens.CicloComViagens{ciclo},
						QuantidadeReservasIda:   1,
						QuantidadeReservasVolta: 1,
						CapacidadeTotal:         7,
					}, nil
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid date",
			body:       map[string]any{"data_viagem": "10-06-2026"},
			svc:        fakePlanejamentoService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing rota interna",
			body: func() map[string]any {
				in := validPlanejamentoBody()
				in["rota_interna_id"] = 0
				return in
			}(),
			svc:        fakePlanejamentoService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "already exists",
			body: validPlanejamentoBody(),
			svc: fakePlanejamentoService{
				planejarFn: func(_ context.Context, _ viagens.PlanejamentoViagensInput, _ map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error) {
					return nil, brerror.ErrAlreadyExists
				},
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "not found",
			body: validPlanejamentoBody(),
			svc: fakePlanejamentoService{
				planejarFn: func(_ context.Context, _ viagens.PlanejamentoViagensInput, _ map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error) {
					return nil, brerror.ErrNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "internal error",
			body: validPlanejamentoBody(),
			svc: fakePlanejamentoService{
				planejarFn: func(_ context.Context, _ viagens.PlanejamentoViagensInput, _ map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error) {
					return nil, errors.New("db")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := viagens.NewPlanejamentoHandler(tc.svc)
			req := httptest.NewRequest(http.MethodPost, "/planejamentos/viagens", body(tc.body))
			rr := httptest.NewRecorder()

			newPlanejamentoRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
