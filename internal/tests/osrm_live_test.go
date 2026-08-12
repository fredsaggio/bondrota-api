package tests

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestLiveOSRMRotaDinamica(t *testing.T) {
	if strings.TrimSpace(os.Getenv("OSRM_LIVE_TEST")) != "1" {
		t.Skip("set OSRM_LIVE_TEST=1 to run live OSRM tests")
	}

	h := newE2EHarness(t, e2eRouterOptions{})
	suffix := h.Suffix
	cidade := "e2e-osrm-live-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "99" + suffix[len(suffix)-5:],
		ClientePrefix:   "89" + suffix[len(suffix)-5:],
		// "O" sozinho não é uma placa válida — e2ePlaca completava com "XX",
		// gerando "OXX...", que este prefixo de 1 letra nunca batia na limpeza.
		PlacaPrefix:   e2ePlacaPrefix("OSR", suffix),
		CidadeDestino: cidade,
		DestinoPrefix: "Destino Base E2E " + suffix,
	})

	rotaInternaID, dataViagem := setupPlanejamentoBase(t, h, planejamentoBaseOptions{
		CidadeDestino:   cidade,
		Prefixo:         suffix,
		MotoristaPrefix: "99" + suffix[len(suffix)-5:],
		ClientePrefix:   "89" + suffix[len(suffix)-5:],
		PlacaPrefix:     e2ePlacaPrefix("OSR", suffix),
		CriarVeiculo:    true,
		CriarMotorista:  true,
		CriarHorario:    true,
	})

	planejamento := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/test/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":          dataViagem,
		"turno":                "NT",
		"sentido":              "ida",
		"municipio_destino_id": e2eMunicipioID,
		"rota_interna_id":      rotaInternaID,
	}, http.StatusCreated)
	viagemID := int64(planejamento["ciclos"].([]any)[0].(map[string]any)["viagens"].([]any)[0].(map[string]any)["id"].(float64))

	rota := doJSON[map[string]any](t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/rota-dinamica/calcular", viagemID), h.AdminToken, nil, http.StatusCreated)
	rotaData := rota["rota"].(map[string]any)

	distancia := int(rotaData["distancia_metros"].(float64))
	duracao := int(rotaData["duracao_segundos"].(float64))
	if distancia <= 0 {
		t.Fatalf("expected positive OSRM distance, got %d", distancia)
	}
	if duracao <= 0 {
		t.Fatalf("expected positive OSRM duration, got %d", duracao)
	}

	geometry, ok := rotaData["geometry"].(map[string]any)
	if !ok {
		t.Fatalf("expected geojson geometry object, got %T", rotaData["geometry"])
	}
	if geometry["type"] != "LineString" {
		t.Fatalf("expected LineString geometry, got %v", geometry["type"])
	}

	t.Logf("calculated live OSRM route: viagem_id=%d distancia_metros=%d duracao_segundos=%d", viagemID, distancia, duracao)
}
