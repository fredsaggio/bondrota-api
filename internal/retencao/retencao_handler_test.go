package retencao_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/retencao"
)

type fakeService struct {
	resumo retencao.ResumoLimpeza
	err    error
}

func (s fakeService) Limpar(context.Context) (retencao.ResumoLimpeza, error) {
	return s.resumo, s.err
}

func chamar(t *testing.T, svc retencao.Service) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	retencao.NewHandler(svc).Limpar(rr, httptest.NewRequest(http.MethodPost, "/internal/retencao/limpar", nil))
	return rr
}

func TestHandlerLimparRetornaResumo(t *testing.T) {
	corte := time.Date(2030, 3, 15, 0, 0, 0, 0, time.UTC)
	rr := chamar(t, fakeService{resumo: retencao.ResumoLimpeza{
		Corte:              corte,
		CiclosRemovidos:    4,
		ReservasRemovidas:  120,
		ExecucoesRemovidas: 8,
		LoteSaturado:       true,
	}})

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["corte"] != corte.Format(time.RFC3339) {
		t.Fatalf("want corte %s, got %v", corte.Format(time.RFC3339), body["corte"])
	}
	if body["ciclos_removidos"] != float64(4) || body["reservas_removidas"] != float64(120) || body["execucoes_removidas"] != float64(8) {
		t.Fatalf("unexpected counters: %v", body)
	}
	if body["lote_saturado"] != true {
		t.Fatalf("want lote_saturado true, got %v", body["lote_saturado"])
	}
}

func TestHandlerLimparTraduzFalhaPara500(t *testing.T) {
	rr := chamar(t, fakeService{err: errors.New("db")})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rr.Code)
	}
}
