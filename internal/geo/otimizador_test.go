package geo_test

import (
	"errors"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/geo"
)

func TestOtimizadorRota_OrdenarDestinosForcaBruta(t *testing.T) {
	otimizador := geo.NewOtimizadorRota()

	result, err := otimizador.OrdenarDestinos(geo.OtimizacaoRotaInput{
		Origem: geo.Coordenada{Latitude: -10, Longitude: -36},
		Destinos: []geo.PontoRoteirizacao{
			{ID: 1, Coordenada: geo.Coordenada{Latitude: -13, Longitude: -39}},
			{ID: 2, Coordenada: geo.Coordenada{Latitude: -12, Longitude: -39}},
			{ID: 3, Coordenada: geo.Coordenada{Latitude: -11, Longitude: -39}},
			{ID: 4, Coordenada: geo.Coordenada{Latitude: -9, Longitude: -39}},
		},
	})

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	assertIDs(t, result.Destinos, []int64{4, 3, 2, 1})
	if result.DistanciaEstimadaMetros <= 0 {
		t.Fatalf("expected positive estimated distance, got %d", result.DistanciaEstimadaMetros)
	}
}

func TestOtimizadorRota_OrdenarDestinosComDestinoFinal(t *testing.T) {
	otimizador := geo.NewOtimizadorRota()

	result, err := otimizador.OrdenarDestinos(geo.OtimizacaoRotaInput{
		Origem: geo.Coordenada{Latitude: -10, Longitude: -36},
		Destinos: []geo.PontoRoteirizacao{
			{ID: 1, Coordenada: geo.Coordenada{Latitude: -13, Longitude: -39}},
			{ID: 2, Coordenada: geo.Coordenada{Latitude: -12, Longitude: -39}},
			{ID: 3, Coordenada: geo.Coordenada{Latitude: -11, Longitude: -39}},
			{ID: 4, Coordenada: geo.Coordenada{Latitude: -9, Longitude: -39}},
		},
		DestinoFinal:     geo.Coordenada{Latitude: -14, Longitude: -39},
		UsarDestinoFinal: true,
	})

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	assertIDs(t, result.Destinos, []int64{4, 3, 2, 1})
}

func TestOtimizadorRota_OrdenarDestinosVizinhoMaisProximoCom2Opt(t *testing.T) {
	otimizador := geo.NewOtimizadorRota()

	result, err := otimizador.OrdenarDestinos(geo.OtimizacaoRotaInput{
		Origem: geo.Coordenada{Latitude: -9.78, Longitude: -36.35},
		Destinos: []geo.PontoRoteirizacao{
			{ID: 1, Coordenada: geo.Coordenada{Latitude: -9.70, Longitude: -36.30}},
			{ID: 2, Coordenada: geo.Coordenada{Latitude: -9.60, Longitude: -36.20}},
			{ID: 3, Coordenada: geo.Coordenada{Latitude: -9.50, Longitude: -36.10}},
			{ID: 4, Coordenada: geo.Coordenada{Latitude: -9.40, Longitude: -36.00}},
			{ID: 5, Coordenada: geo.Coordenada{Latitude: -9.30, Longitude: -35.90}},
			{ID: 6, Coordenada: geo.Coordenada{Latitude: -9.20, Longitude: -35.80}},
			{ID: 7, Coordenada: geo.Coordenada{Latitude: -9.10, Longitude: -35.70}},
			{ID: 8, Coordenada: geo.Coordenada{Latitude: -9.00, Longitude: -35.60}},
			{ID: 9, Coordenada: geo.Coordenada{Latitude: -8.90, Longitude: -35.50}},
		},
	})

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(result.Destinos) != 9 {
		t.Fatalf("expected 9 destinos, got %d", len(result.Destinos))
	}
	assertUniqueIDs(t, result.Destinos)
	if result.DistanciaEstimadaMetros <= 0 {
		t.Fatalf("expected positive estimated distance, got %d", result.DistanciaEstimadaMetros)
	}
}

func TestOtimizadorRota_OrdenarDestinosValidation(t *testing.T) {
	otimizador := geo.NewOtimizadorRota()

	tests := []struct {
		name  string
		input geo.OtimizacaoRotaInput
	}{
		{
			name: "requires destinos",
			input: geo.OtimizacaoRotaInput{
				Origem: geo.Coordenada{Latitude: -9.78, Longitude: -36.35},
			},
		},
		{
			name: "rejects duplicated id",
			input: geo.OtimizacaoRotaInput{
				Origem: geo.Coordenada{Latitude: -9.78, Longitude: -36.35},
				Destinos: []geo.PontoRoteirizacao{
					{ID: 1, Coordenada: geo.Coordenada{Latitude: -9.7, Longitude: -36.3}},
					{ID: 1, Coordenada: geo.Coordenada{Latitude: -9.6, Longitude: -36.2}},
				},
			},
		},
		{
			name: "rejects invalid latitude",
			input: geo.OtimizacaoRotaInput{
				Origem: geo.Coordenada{Latitude: -9.78, Longitude: -36.35},
				Destinos: []geo.PontoRoteirizacao{
					{ID: 1, Coordenada: geo.Coordenada{Latitude: -91, Longitude: -36.3}},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := otimizador.OrdenarDestinos(tc.input)
			if !errors.Is(err, brerror.ErrInvalidInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}

func assertIDs(t *testing.T, pontos []geo.PontoRoteirizacao, want []int64) {
	t.Helper()

	if len(pontos) != len(want) {
		t.Fatalf("expected %d pontos, got %d", len(want), len(pontos))
	}
	for i := range pontos {
		if pontos[i].ID != want[i] {
			t.Fatalf("unexpected order at %d: want %d, got %d", i, want[i], pontos[i].ID)
		}
	}
}

func assertUniqueIDs(t *testing.T, pontos []geo.PontoRoteirizacao) {
	t.Helper()

	seen := make(map[int64]struct{}, len(pontos))
	for _, ponto := range pontos {
		if _, ok := seen[ponto.ID]; ok {
			t.Fatalf("duplicated id %d in %+v", ponto.ID, pontos)
		}
		seen[ponto.ID] = struct{}{}
	}
}
