package viagens_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

type fakeCicloViagemStore struct {
	createCicloFn           func(ctx context.Context, input viagens.CicloViagemInput) (*viagens.CicloViagem, error)
	createCicloComViagensFn func(ctx context.Context, input viagens.CicloViagemInput, partidas map[viagens.SentidoViagem]time.Time) (*viagens.CicloComViagens, error)
	getCicloByIDFn          func(ctx context.Context, cicloID int64) (*viagens.CicloViagem, error)
	listCiclosFn            func(ctx context.Context) ([]viagens.CicloViagem, error)
	updateCicloFn           func(ctx context.Context, cicloID int64, updateFunc func(*viagens.CicloViagem) (bool, error)) (*viagens.CicloViagem, error)
}

func (s fakeCicloViagemStore) CreateCiclo(ctx context.Context, input viagens.CicloViagemInput) (*viagens.CicloViagem, error) {
	return s.createCicloFn(ctx, input)
}

func (s fakeCicloViagemStore) CreateCicloComViagens(ctx context.Context, input viagens.CicloViagemInput, partidas map[viagens.SentidoViagem]time.Time) (*viagens.CicloComViagens, error) {
	return s.createCicloComViagensFn(ctx, input, partidas)
}

func (s fakeCicloViagemStore) GetCicloByID(ctx context.Context, cicloID int64) (*viagens.CicloViagem, error) {
	return s.getCicloByIDFn(ctx, cicloID)
}

func (s fakeCicloViagemStore) ListCiclos(ctx context.Context) ([]viagens.CicloViagem, error) {
	return s.listCiclosFn(ctx)
}

func (s fakeCicloViagemStore) UpdateCiclo(ctx context.Context, cicloID int64, updateFunc func(*viagens.CicloViagem) (bool, error)) (*viagens.CicloViagem, error) {
	return s.updateCicloFn(ctx, cicloID, updateFunc)
}

func validCicloInput() viagens.CicloViagemInput {
	return viagens.CicloViagemInput{
		DataViagem:    time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Turno:         viagens.TurnoNoturno,
		Cidade:        "Campo Alegre",
		RotaInternaID: 2,
		VeiculoID:     3,
		MotoristaID:   4,
		ExpiresAt:     time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
	}
}

func validPartidas() map[viagens.SentidoViagem]time.Time {
	return map[viagens.SentidoViagem]time.Time{
		viagens.SentidoIda:   time.Date(2026, 6, 10, 18, 0, 0, 0, time.UTC),
		viagens.SentidoVolta: time.Date(2026, 6, 10, 22, 0, 0, 0, time.UTC),
	}
}

func TestPlanejamentoService_Planejar(t *testing.T) {
	t.Run("valid input delegates to store", func(t *testing.T) {
		input := validCicloInput()
		partidas := validPartidas()
		svc := viagens.NewPlanejamentoService(fakeCicloViagemStore{
			createCicloComViagensFn: func(_ context.Context, gotInput viagens.CicloViagemInput, gotPartidas map[viagens.SentidoViagem]time.Time) (*viagens.CicloComViagens, error) {
				if gotInput != input {
					t.Fatalf("unexpected input: %+v", gotInput)
				}
				if !gotPartidas[viagens.SentidoVolta].Equal(partidas[viagens.SentidoVolta]) {
					t.Fatalf("unexpected partidas: %+v", gotPartidas)
				}
				ciclo := sampleCicloComViagens()
				return &ciclo, nil
			},
		})

		ciclo, err := svc.Planejar(context.Background(), input, partidas)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(ciclo.Viagens) != 2 {
			t.Fatalf("expected 2 viagens, got %d", len(ciclo.Viagens))
		}
	})

	t.Run("invalid input does not call store", func(t *testing.T) {
		svc := viagens.NewPlanejamentoService(fakeCicloViagemStore{
			createCicloComViagensFn: func(_ context.Context, _ viagens.CicloViagemInput, _ map[viagens.SentidoViagem]time.Time) (*viagens.CicloComViagens, error) {
				t.Fatal("store should not be called")
				return nil, nil
			},
		})

		_, err := svc.Planejar(context.Background(), viagens.CicloViagemInput{}, nil)

		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("store error is returned", func(t *testing.T) {
		svc := viagens.NewPlanejamentoService(fakeCicloViagemStore{
			createCicloComViagensFn: func(_ context.Context, _ viagens.CicloViagemInput, _ map[viagens.SentidoViagem]time.Time) (*viagens.CicloComViagens, error) {
				return nil, brerror.ErrAlreadyExists
			},
		})

		_, err := svc.Planejar(context.Background(), validCicloInput(), validPartidas())

		if !errors.Is(err, brerror.ErrAlreadyExists) {
			t.Fatalf("expected already exists, got %v", err)
		}
	})
}
