package viagens_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

type fakeExecucaoPlanejamentoFalhaStore struct {
	listFn func(context.Context, int) ([]viagens.ExecucaoPlanejamento, error)
}

func (s fakeExecucaoPlanejamentoFalhaStore) ListFalhas(ctx context.Context, limit int) ([]viagens.ExecucaoPlanejamento, error) {
	return s.listFn(ctx, limit)
}

func TestExecucaoPlanejamentoHandler_ListFalhas(t *testing.T) {
	t.Run("returns failures", func(t *testing.T) {
		agora := time.Date(2026, time.August, 12, 16, 30, 0, 0, time.UTC)
		erro := "vehicles unavailable"
		handler := viagens.NewExecucaoPlanejamentoHandler(fakeExecucaoPlanejamentoFalhaStore{
			listFn: func(_ context.Context, limit int) ([]viagens.ExecucaoPlanejamento, error) {
				if limit != 10 {
					t.Fatalf("expected limit 10, got %d", limit)
				}
				return []viagens.ExecucaoPlanejamento{{
					ID:                 7,
					DataViagem:         agora,
					Turno:              viagens.TurnoNoturno,
					MunicipioDestinoID: 2704302,
					RotaInternaID:      3,
					Sentido:            viagens.SentidoIda,
					PartidaEm:          agora.Add(30 * time.Minute),
					FechamentoEm:       agora,
					Status:             viagens.StatusExecucaoFalhou,
					Tentativas:         2,
					UltimoErro:         &erro,
					ProximaTentativaEm: ptrTime(agora.Add(2 * time.Minute)),
					FinalizadoEm:       &agora,
				}}, nil
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/planejamentos/execucoes/falhas?limit=10", nil)
		rr := httptest.NewRecorder()

		handler.ListFalhas(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var response []viagens.ExecucaoPlanejamentoFalhaResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(response) != 1 || response[0].ID != 7 || response[0].Tentativas != 2 || response[0].ProximaTentativaEm == nil {
			t.Fatalf("unexpected response: %+v", response)
		}
	})

	t.Run("rejects invalid limit", func(t *testing.T) {
		handler := viagens.NewExecucaoPlanejamentoHandler(fakeExecucaoPlanejamentoFalhaStore{
			listFn: func(context.Context, int) ([]viagens.ExecucaoPlanejamento, error) {
				t.Fatal("store should not be called")
				return nil, nil
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/planejamentos/execucoes/falhas?limit=101", nil)
		rr := httptest.NewRecorder()

		handler.ListFalhas(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("returns internal server error", func(t *testing.T) {
		handler := viagens.NewExecucaoPlanejamentoHandler(fakeExecucaoPlanejamentoFalhaStore{
			listFn: func(context.Context, int) ([]viagens.ExecucaoPlanejamento, error) {
				return nil, errors.New("db")
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/planejamentos/execucoes/falhas", nil)
		rr := httptest.NewRecorder()

		handler.ListFalhas(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
