package rotasdinamicas_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
)

type fakeRotaDinamicaStore struct {
	createFn       func(ctx context.Context, input rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error)
	getByViagemFn  func(ctx context.Context, viagemID int64) (*rotasdinamicas.RotaDinamicaComDestinos, error)
	listDestinosFn func(ctx context.Context, rotaDinamicaID int64) ([]rotasdinamicas.RotaDinamicaDestino, error)
	deleteFn       func(ctx context.Context, viagemID int64) error
}

func (s fakeRotaDinamicaStore) Create(ctx context.Context, input rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
	return s.createFn(ctx, input)
}

func (s fakeRotaDinamicaStore) GetByViagem(ctx context.Context, viagemID int64) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
	return s.getByViagemFn(ctx, viagemID)
}

func (s fakeRotaDinamicaStore) ListDestinos(ctx context.Context, rotaDinamicaID int64) ([]rotasdinamicas.RotaDinamicaDestino, error) {
	return s.listDestinosFn(ctx, rotaDinamicaID)
}

func (s fakeRotaDinamicaStore) DeleteByViagem(ctx context.Context, viagemID int64) error {
	return s.deleteFn(ctx, viagemID)
}

func validRotaInput() rotasdinamicas.RotaDinamicaInput {
	return rotasdinamicas.RotaDinamicaInput{
		ViagemID: 10,
		Origem: rotasdinamicas.PontoRota{
			Nome:      " Ultima parada ",
			Latitude:  -9.780000,
			Longitude: -36.350000,
		},
		DestinoFinal: rotasdinamicas.PontoRota{
			Nome:      " UFAL ",
			Latitude:  -9.558000,
			Longitude: -35.775000,
		},
		DistanciaMetros: 100000,
		DuracaoSegundos: 7200,
		Geometry:        []byte(`{"type":"LineString","coordinates":[[-36.35,-9.78],[-35.775,-9.558]]}`),
		ExpiresAt:       time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		Destinos: []rotasdinamicas.RotaDinamicaDestinoInput{
			{DestinoID: 5},
			{DestinoID: 8, Ordem: 99},
		},
	}
}

func sampleRota(input rotasdinamicas.RotaDinamicaInput) *rotasdinamicas.RotaDinamicaComDestinos {
	destinos := make([]rotasdinamicas.RotaDinamicaDestino, 0, len(input.Destinos))
	for i, destino := range input.Destinos {
		destinos = append(destinos, rotasdinamicas.RotaDinamicaDestino{
			ID:             int64(i + 1),
			RotaDinamicaID: 1,
			DestinoID:      destino.DestinoID,
			Ordem:          destino.Ordem,
		})
	}

	return &rotasdinamicas.RotaDinamicaComDestinos{
		Rota: rotasdinamicas.RotaDinamica{
			ID:                    1,
			ViagemID:              input.ViagemID,
			Provider:              input.Provider,
			OrigemNome:            input.Origem.Nome,
			OrigemLatitude:        input.Origem.Latitude,
			OrigemLongitude:       input.Origem.Longitude,
			DestinoFinalNome:      input.DestinoFinal.Nome,
			DestinoFinalLatitude:  input.DestinoFinal.Latitude,
			DestinoFinalLongitude: input.DestinoFinal.Longitude,
			DistanciaMetros:       input.DistanciaMetros,
			DuracaoSegundos:       input.DuracaoSegundos,
			Geometry:              input.Geometry,
			ExpiresAt:             input.ExpiresAt,
			CreatedAt:             time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
			UpdatedAt:             time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		},
		Destinos: destinos,
	}
}

func TestRotaDinamicaService_Create(t *testing.T) {
	t.Run("defaults provider, trims points and normalizes destination order", func(t *testing.T) {
		svc := rotasdinamicas.NewRotaDinamicaService(fakeRotaDinamicaStore{
			createFn: func(_ context.Context, input rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
				if input.Provider != "osrm" {
					t.Fatalf("expected osrm provider, got %q", input.Provider)
				}
				if input.Origem.Nome != "Ultima parada" || input.DestinoFinal.Nome != "UFAL" {
					t.Fatalf("expected trimmed point names, got origem=%q destino=%q", input.Origem.Nome, input.DestinoFinal.Nome)
				}
				if input.Destinos[0].Ordem != 1 || input.Destinos[1].Ordem != 2 {
					t.Fatalf("expected normalized order, got %+v", input.Destinos)
				}
				return sampleRota(input), nil
			},
		})

		rota, err := svc.Create(context.Background(), validRotaInput())

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if rota.Rota.Provider != "osrm" {
			t.Fatalf("unexpected provider: %s", rota.Rota.Provider)
		}
	})

	t.Run("preserves explicit provider", func(t *testing.T) {
		input := validRotaInput()
		input.Provider = "manual"
		svc := rotasdinamicas.NewRotaDinamicaService(fakeRotaDinamicaStore{
			createFn: func(_ context.Context, input rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
				if input.Provider != "manual" {
					t.Fatalf("expected manual provider, got %q", input.Provider)
				}
				return sampleRota(input), nil
			},
		})

		_, err := svc.Create(context.Background(), input)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("invalid input does not call store", func(t *testing.T) {
		input := validRotaInput()
		input.Geometry = []byte("{")
		svc := rotasdinamicas.NewRotaDinamicaService(fakeRotaDinamicaStore{
			createFn: func(_ context.Context, _ rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
				t.Fatal("store should not be called")
				return nil, nil
			},
		})

		_, err := svc.Create(context.Background(), input)

		if !errors.Is(err, brerror.ErrInvalidInput) {
			t.Fatalf("expected invalid input error, got %v", err)
		}
	})

	t.Run("duplicated destination is rejected before store", func(t *testing.T) {
		input := validRotaInput()
		input.Destinos[1].DestinoID = input.Destinos[0].DestinoID
		svc := rotasdinamicas.NewRotaDinamicaService(fakeRotaDinamicaStore{
			createFn: func(_ context.Context, _ rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
				t.Fatal("store should not be called")
				return nil, nil
			},
		})

		_, err := svc.Create(context.Background(), input)

		if !errors.Is(err, brerror.ErrInvalidInput) {
			t.Fatalf("expected invalid input error, got %v", err)
		}
	})

	t.Run("store error is returned", func(t *testing.T) {
		svc := rotasdinamicas.NewRotaDinamicaService(fakeRotaDinamicaStore{
			createFn: func(_ context.Context, _ rotasdinamicas.RotaDinamicaInput) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
				return nil, brerror.ErrAlreadyExists
			},
		})

		_, err := svc.Create(context.Background(), validRotaInput())

		if !errors.Is(err, brerror.ErrAlreadyExists) {
			t.Fatalf("expected already exists, got %v", err)
		}
	})
}

func TestRotaDinamicaService_GetByViagem(t *testing.T) {
	t.Run("requires valid viagem id", func(t *testing.T) {
		svc := rotasdinamicas.NewRotaDinamicaService(fakeRotaDinamicaStore{})

		_, err := svc.GetByViagem(context.Background(), 0)

		if !errors.Is(err, brerror.ErrInvalidInput) {
			t.Fatalf("expected invalid input error, got %v", err)
		}
	})

	t.Run("delegates to store", func(t *testing.T) {
		svc := rotasdinamicas.NewRotaDinamicaService(fakeRotaDinamicaStore{
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
		})

		rota, err := svc.GetByViagem(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if rota.Rota.ViagemID != 10 {
			t.Fatalf("unexpected viagemID: %d", rota.Rota.ViagemID)
		}
	})
}

func TestRotaDinamicaService_DeleteByViagem(t *testing.T) {
	t.Run("requires valid viagem id", func(t *testing.T) {
		svc := rotasdinamicas.NewRotaDinamicaService(fakeRotaDinamicaStore{})

		err := svc.DeleteByViagem(context.Background(), 0)

		if !errors.Is(err, brerror.ErrInvalidInput) {
			t.Fatalf("expected invalid input error, got %v", err)
		}
	})

	t.Run("delegates to store", func(t *testing.T) {
		svc := rotasdinamicas.NewRotaDinamicaService(fakeRotaDinamicaStore{
			deleteFn: func(_ context.Context, viagemID int64) error {
				if viagemID != 10 {
					t.Fatalf("unexpected viagemID: %d", viagemID)
				}
				return nil
			},
		})

		err := svc.DeleteByViagem(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})
}
