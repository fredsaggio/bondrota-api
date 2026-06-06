package rotasdinamicas_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
)

type fakeAgendadorRotaDinamicaStore struct {
	listFn func(ctx context.Context, now time.Time, janelaCalculo, janelaBloqueio time.Duration) ([]int64, error)
}

func (s fakeAgendadorRotaDinamicaStore) ListViagensPendentesCalculo(ctx context.Context, now time.Time, janelaCalculo, janelaBloqueio time.Duration) ([]int64, error) {
	return s.listFn(ctx, now, janelaCalculo, janelaBloqueio)
}

type fakeCalculadorWorkerService struct {
	calcularFn func(ctx context.Context, viagemID int64) (*rotasdinamicas.RotaDinamicaComDestinos, error)
}

func (s fakeCalculadorWorkerService) Calcular(ctx context.Context, viagemID int64) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
	return s.calcularFn(ctx, viagemID)
}

func TestRotaDinamicaWorker_Processar(t *testing.T) {
	called := make([]int64, 0, 2)
	worker := rotasdinamicas.NewRotaDinamicaWorker(
		fakeAgendadorRotaDinamicaStore{
			listFn: func(_ context.Context, _ time.Time, janelaCalculo, janelaBloqueio time.Duration) ([]int64, error) {
				if janelaCalculo != time.Hour {
					t.Fatalf("unexpected calculation window: %s", janelaCalculo)
				}
				if janelaBloqueio != 30*time.Minute {
					t.Fatalf("unexpected lock window: %s", janelaBloqueio)
				}
				return []int64{10, 11}, nil
			},
		},
		fakeCalculadorWorkerService{
			calcularFn: func(_ context.Context, viagemID int64) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
				called = append(called, viagemID)
				return &rotasdinamicas.RotaDinamicaComDestinos{}, nil
			},
		},
		rotasdinamicas.RotaDinamicaWorkerConfig{},
	)

	worker.Processar(context.Background())

	if len(called) != 2 || called[0] != 10 || called[1] != 11 {
		t.Fatalf("unexpected calculated viagens: %+v", called)
	}
}

func TestRotaDinamicaWorker_ProcessarListError(t *testing.T) {
	worker := rotasdinamicas.NewRotaDinamicaWorker(
		fakeAgendadorRotaDinamicaStore{
			listFn: func(_ context.Context, _ time.Time, _, _ time.Duration) ([]int64, error) {
				return nil, errors.New("db")
			},
		},
		fakeCalculadorWorkerService{
			calcularFn: func(_ context.Context, _ int64) (*rotasdinamicas.RotaDinamicaComDestinos, error) {
				t.Fatal("calculator should not be called")
				return nil, nil
			},
		},
		rotasdinamicas.RotaDinamicaWorkerConfig{},
	)

	worker.Processar(context.Background())
}
