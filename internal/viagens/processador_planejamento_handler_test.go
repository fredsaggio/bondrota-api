package viagens_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

type fakeProcessadorPlanejamento struct {
	processarFn func(context.Context) (viagens.ResumoProcessamentoPlanejamento, error)
}

func (p fakeProcessadorPlanejamento) Processar(ctx context.Context) (viagens.ResumoProcessamentoPlanejamento, error) {
	return p.processarFn(ctx)
}

func TestProcessadorPlanejamentoHandler_Processar(t *testing.T) {
	t.Run("returns processing summary", func(t *testing.T) {
		handler := viagens.NewProcessadorPlanejamentoHandler(fakeProcessadorPlanejamento{
			processarFn: func(context.Context) (viagens.ResumoProcessamentoPlanejamento, error) {
				return viagens.ResumoProcessamentoPlanejamento{
					Candidatos: 4,
					Devidos:    3,
					Adquiridos: 2,
					Concluidos: 1,
					SemDemanda: 1,
				}, nil
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/planejamentos/processar", nil)
		rr := httptest.NewRecorder()

		handler.Processar(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var response viagens.ResumoProcessamentoPlanejamentoResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.Candidatos != 4 || response.Devidos != 3 || response.Adquiridos != 2 || response.Concluidos != 1 || response.SemDemanda != 1 || response.Falhos != 0 {
			t.Fatalf("unexpected response: %+v", response)
		}
	})

	t.Run("returns internal server error", func(t *testing.T) {
		handler := viagens.NewProcessadorPlanejamentoHandler(fakeProcessadorPlanejamento{
			processarFn: func(context.Context) (viagens.ResumoProcessamentoPlanejamento, error) {
				return viagens.ResumoProcessamentoPlanejamento{Falhos: 1}, errors.New("db")
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/planejamentos/processar", nil)
		rr := httptest.NewRecorder()

		handler.Processar(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}
