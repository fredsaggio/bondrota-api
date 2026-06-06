package rotasdinamicas_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/geo"
	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
)

type fakeCalculadorRotaDinamicaStore struct {
	getDadosFn func(ctx context.Context, viagemID int64) (*rotasdinamicas.DadosCalculoRota, error)
}

func (s fakeCalculadorRotaDinamicaStore) GetDadosCalculo(ctx context.Context, viagemID int64) (*rotasdinamicas.DadosCalculoRota, error) {
	return s.getDadosFn(ctx, viagemID)
}

type fakeRoteador struct {
	calcularFn func(ctx context.Context, coordenadas []geo.Coordenada) (*geo.RotaCalculada, error)
}

func (r fakeRoteador) CalcularRota(ctx context.Context, coordenadas []geo.Coordenada) (*geo.RotaCalculada, error) {
	return r.calcularFn(ctx, coordenadas)
}

func dadosCalculo(sentido string) *rotasdinamicas.DadosCalculoRota {
	return &rotasdinamicas.DadosCalculoRota{
		ViagemID:  10,
		Sentido:   sentido,
		ExpiresAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		Paradas: []rotasdinamicas.PontoCalculoRota{
			{ID: 1, Nome: "Primeira parada", Latitude: -9.80, Longitude: -36.40, Ordem: 1},
			{ID: 2, Nome: "Ultima parada", Latitude: -9.78, Longitude: -36.35, Ordem: 2},
		},
		Destinos: []rotasdinamicas.DestinoCalculoRota{
			{ID: 5, Nome: "UFAL", Latitude: -9.558, Longitude: -35.775},
		},
	}
}

func rotaCalculada() *geo.RotaCalculada {
	return &geo.RotaCalculada{
		DistanciaMetros: 100000,
		DuracaoSegundos: 7200,
		Geometry:        json.RawMessage(`{"type":"LineString","coordinates":[[-36.35,-9.78],[-35.775,-9.558]]}`),
	}
}

func TestCalculadorRotaDinamicaService_CalcularIda(t *testing.T) {
	svc := rotasdinamicas.NewCalculadorRotaDinamicaService(
		fakeCalculadorRotaDinamicaStore{
			getDadosFn: func(_ context.Context, viagemID int64) (*rotasdinamicas.DadosCalculoRota, error) {
				if viagemID != 10 {
					t.Fatalf("unexpected viagemID: %d", viagemID)
				}
				return dadosCalculo("ida"), nil
			},
		},
		fakeRotaDinamicaService{
			createFn: func(_ context.Context, input rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
				if input.Origem.Nome != "Ultima parada" {
					t.Fatalf("unexpected origem: %+v", input.Origem)
				}
				if input.DestinoFinal.Nome != "UFAL" {
					t.Fatalf("unexpected destino final: %+v", input.DestinoFinal)
				}
				if input.DistanciaMetros != 100000 || input.DuracaoSegundos != 7200 {
					t.Fatalf("unexpected route metrics: %+v", input)
				}
				if len(input.Destinos) != 1 || input.Destinos[0].DestinoID != 5 {
					t.Fatalf("unexpected destinos: %+v", input.Destinos)
				}
				input.Provider = "osrm"
				input.Destinos[0].Ordem = 1
				return sampleRota(input), nil
			},
		},
		fakeRoteador{
			calcularFn: func(_ context.Context, coordenadas []geo.Coordenada) (*geo.RotaCalculada, error) {
				if len(coordenadas) != 2 {
					t.Fatalf("expected 2 coordinates, got %+v", coordenadas)
				}
				if coordenadas[0].Latitude != -9.78 || coordenadas[0].Longitude != -36.35 {
					t.Fatalf("expected first coordinate to be last parada, got %+v", coordenadas[0])
				}
				return rotaCalculada(), nil
			},
		},
		nil,
	)

	rota, err := svc.Calcular(context.Background(), 10)

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if rota.Rota.ViagemID != 10 {
		t.Fatalf("unexpected viagemID: %d", rota.Rota.ViagemID)
	}
}

func TestCalculadorRotaDinamicaService_CalcularVolta(t *testing.T) {
	svc := rotasdinamicas.NewCalculadorRotaDinamicaService(
		fakeCalculadorRotaDinamicaStore{
			getDadosFn: func(_ context.Context, _ int64) (*rotasdinamicas.DadosCalculoRota, error) {
				return dadosCalculo("volta"), nil
			},
		},
		fakeRotaDinamicaService{
			createFn: func(_ context.Context, input rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
				if input.Origem.Nome != "UFAL" {
					t.Fatalf("unexpected origem: %+v", input.Origem)
				}
				if input.DestinoFinal.Nome != "Primeira parada" {
					t.Fatalf("unexpected destino final: %+v", input.DestinoFinal)
				}
				input.Provider = "osrm"
				input.Destinos[0].Ordem = 1
				return sampleRota(input), nil
			},
		},
		fakeRoteador{
			calcularFn: func(_ context.Context, coordenadas []geo.Coordenada) (*geo.RotaCalculada, error) {
				if len(coordenadas) != 2 {
					t.Fatalf("expected 2 coordinates, got %+v", coordenadas)
				}
				if coordenadas[0].Latitude != -9.558 || coordenadas[1].Latitude != -9.80 {
					t.Fatalf("unexpected volta coordinates: %+v", coordenadas)
				}
				return rotaCalculada(), nil
			},
		},
		nil,
	)

	_, err := svc.Calcular(context.Background(), 10)

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestCalculadorRotaDinamicaService_CalcularValidation(t *testing.T) {
	t.Run("requires valid viagem id", func(t *testing.T) {
		svc := rotasdinamicas.NewCalculadorRotaDinamicaService(fakeCalculadorRotaDinamicaStore{}, fakeRotaDinamicaService{}, fakeRoteador{}, nil)

		_, err := svc.Calcular(context.Background(), 0)

		if !errors.Is(err, brerror.ErrInvalidInput) {
			t.Fatalf("expected invalid input, got %v", err)
		}
	})

	t.Run("requires paradas", func(t *testing.T) {
		svc := rotasdinamicas.NewCalculadorRotaDinamicaService(
			fakeCalculadorRotaDinamicaStore{
				getDadosFn: func(_ context.Context, _ int64) (*rotasdinamicas.DadosCalculoRota, error) {
					dados := dadosCalculo("ida")
					dados.Paradas = nil
					return dados, nil
				},
			},
			fakeRotaDinamicaService{
				createFn: func(_ context.Context, _ rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
					t.Fatal("route should not be persisted")
					return nil, nil
				},
			},
			fakeRoteador{
				calcularFn: func(_ context.Context, _ []geo.Coordenada) (*geo.RotaCalculada, error) {
					t.Fatal("roteador should not be called")
					return nil, nil
				},
			},
			nil,
		)

		_, err := svc.Calcular(context.Background(), 10)

		if !errors.Is(err, brerror.ErrInvalidInput) {
			t.Fatalf("expected invalid input, got %v", err)
		}
	})

	t.Run("store not found is returned", func(t *testing.T) {
		svc := rotasdinamicas.NewCalculadorRotaDinamicaService(
			fakeCalculadorRotaDinamicaStore{
				getDadosFn: func(_ context.Context, _ int64) (*rotasdinamicas.DadosCalculoRota, error) {
					return nil, brerror.ErrNotFound
				},
			},
			fakeRotaDinamicaService{},
			fakeRoteador{},
			nil,
		)

		_, err := svc.Calcular(context.Background(), 10)

		if !errors.Is(err, brerror.ErrNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})
}
