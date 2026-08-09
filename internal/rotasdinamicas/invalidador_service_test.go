package rotasdinamicas_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
)

type fakeInvalidadorRotaDinamicaStore struct {
	deleteFn func(ctx context.Context, reservaID int64, now time.Time, janelaBloqueio time.Duration) error
}

func (s fakeInvalidadorRotaDinamicaStore) DeleteRotasPorReservaAntesDoBloqueio(ctx context.Context, reservaID int64, now time.Time, janelaBloqueio time.Duration) error {
	return s.deleteFn(ctx, reservaID, now, janelaBloqueio)
}

func TestInvalidadorRotaDinamicaService_InvalidarPorReserva(t *testing.T) {
	svc := rotasdinamicas.NewInvalidadorRotaDinamicaService(fakeInvalidadorRotaDinamicaStore{
		deleteFn: func(_ context.Context, reservaID int64, now time.Time, janelaBloqueio time.Duration) error {
			if reservaID != 20 {
				t.Fatalf("unexpected reservaID: %d", reservaID)
			}
			if now.IsZero() {
				t.Fatal("expected non-zero now")
			}
			if janelaBloqueio != 30*time.Minute {
				t.Fatalf("unexpected lock window: %s", janelaBloqueio)
			}
			return nil
		},
	}, 0)

	if err := svc.InvalidarPorReserva(context.Background(), 20); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestInvalidadorRotaDinamicaService_InvalidarPorReservaValidation(t *testing.T) {
	svc := rotasdinamicas.NewInvalidadorRotaDinamicaService(fakeInvalidadorRotaDinamicaStore{
		deleteFn: func(_ context.Context, _ int64, _ time.Time, _ time.Duration) error {
			t.Fatal("store should not be called")
			return nil
		},
	}, 0)

	err := svc.InvalidarPorReserva(context.Background(), 0)

	if !errors.Is(err, brerror.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
