package geo_test

import (
	"errors"
	"math"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/geo"
)

func TestOtimizadorRota_OrdenarDestinosHeldKarp(t *testing.T) {
	destinos := sampleDestinos(3)

	t.Run("uses costs from fixed origin", func(t *testing.T) {
		result, err := geo.NewOtimizadorRota().OrdenarDestinosPorMatriz(geo.OtimizacaoRotaMatrizInput{
			Destinos: destinos,
			CustosEntreDestinos: [][]float64{
				{0, 1, 20},
				{20, 0, 1},
				{20, 20, 0},
			},
			CustosOrigem: []float64{1, 10, 10},
		})

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		assertIDs(t, result.Destinos, []int64{1, 2, 3})
		if result.CustoEstimado != 3 {
			t.Fatalf("expected cost 3, got %f", result.CustoEstimado)
		}
	})

	t.Run("uses free origin and fixed final destination", func(t *testing.T) {
		result, err := geo.NewOtimizadorRota().OrdenarDestinosPorMatriz(geo.OtimizacaoRotaMatrizInput{
			Destinos: destinos,
			CustosEntreDestinos: [][]float64{
				{0, 20, 20},
				{1, 0, 20},
				{20, 1, 0},
			},
			CustosDestinoFinal: []float64{1, 10, 10},
		})

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		assertIDs(t, result.Destinos, []int64{3, 2, 1})
		if result.CustoEstimado != 3 {
			t.Fatalf("expected cost 3, got %f", result.CustoEstimado)
		}
	})
}

func TestOtimizadorRota_OrdenarDestinosHeuristicaAcimaDoLimite(t *testing.T) {
	const quantidade = 13
	custos := make([][]float64, quantidade)
	for i := 0; i < quantidade; i++ {
		custos[i] = make([]float64, quantidade)
		for j := 0; j < quantidade; j++ {
			custos[i][j] = math.Abs(float64(i - j))
		}
	}

	result, err := geo.NewOtimizadorRota().OrdenarDestinosPorMatriz(geo.OtimizacaoRotaMatrizInput{
		Destinos:            sampleDestinos(quantidade),
		CustosEntreDestinos: custos,
		CustosOrigem:        custos[0],
	})

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(result.Destinos) != quantidade {
		t.Fatalf("expected %d destinos, got %d", quantidade, len(result.Destinos))
	}
	assertUniqueIDs(t, result.Destinos)
}

func TestOtimizadorRota_OrdenarDestinosPorMatrizValidation(t *testing.T) {
	tests := []struct {
		name  string
		input geo.OtimizacaoRotaMatrizInput
	}{
		{name: "requires destinos", input: geo.OtimizacaoRotaMatrizInput{}},
		{
			name: "rejects duplicated id",
			input: geo.OtimizacaoRotaMatrizInput{
				Destinos: []geo.PontoRoteirizacao{
					{ID: 1, Coordenada: geo.Coordenada{Latitude: -9.7, Longitude: -36.3}},
					{ID: 1, Coordenada: geo.Coordenada{Latitude: -9.6, Longitude: -36.2}},
				},
			},
		},
		{
			name: "rejects invalid latitude",
			input: geo.OtimizacaoRotaMatrizInput{
				Destinos: []geo.PontoRoteirizacao{
					{ID: 1, Coordenada: geo.Coordenada{Latitude: -91, Longitude: -36.3}},
				},
			},
		},
		{
			name: "requires square matrix",
			input: geo.OtimizacaoRotaMatrizInput{
				Destinos:            sampleDestinos(2),
				CustosEntreDestinos: [][]float64{{0, 1}},
			},
		},
		{
			name: "rejects invalid cost",
			input: geo.OtimizacaoRotaMatrizInput{
				Destinos:            sampleDestinos(2),
				CustosEntreDestinos: [][]float64{{0, -1}, {1, 0}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := geo.NewOtimizadorRota().OrdenarDestinosPorMatriz(tc.input)
			if !errors.Is(err, brerror.ErrInvalidInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}

func sampleDestinos(quantidade int) []geo.PontoRoteirizacao {
	destinos := make([]geo.PontoRoteirizacao, 0, quantidade)
	for i := 0; i < quantidade; i++ {
		destinos = append(destinos, geo.PontoRoteirizacao{
			ID: int64(i + 1),
			Coordenada: geo.Coordenada{
				Latitude:  -9.70 + float64(i)*0.01,
				Longitude: -36.30 + float64(i)*0.01,
			},
		})
	}
	return destinos
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
