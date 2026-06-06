package viagens_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

type fakeViagemReservaStore struct {
	createViagemReservaFn func(ctx context.Context, input viagens.ViagemReservaInput) (*viagens.ViagemReserva, error)
	listReservasFn        func(ctx context.Context, viagemID int64) ([]viagens.ViagemReservaComReserva, error)
	updatePresencaFn      func(ctx context.Context, viagemID, reservaID int64, updateFunc func(*viagens.ViagemReserva) (bool, error)) (*viagens.ViagemReserva, error)
}

func (s fakeViagemReservaStore) CreateViagemReserva(ctx context.Context, input viagens.ViagemReservaInput) (*viagens.ViagemReserva, error) {
	return s.createViagemReservaFn(ctx, input)
}

func (s fakeViagemReservaStore) ListReservasByViagem(ctx context.Context, viagemID int64) ([]viagens.ViagemReservaComReserva, error) {
	return s.listReservasFn(ctx, viagemID)
}

func (s fakeViagemReservaStore) UpdatePresenca(ctx context.Context, viagemID, reservaID int64, updateFunc func(*viagens.ViagemReserva) (bool, error)) (*viagens.ViagemReserva, error) {
	return s.updatePresencaFn(ctx, viagemID, reservaID, updateFunc)
}

func TestPresencaService_ListReservasByViagem(t *testing.T) {
	t.Run("requires valid viagem id", func(t *testing.T) {
		svc := viagens.NewPresencaService(fakeViagemReservaStore{})

		_, err := svc.ListReservasByViagem(context.Background(), 0)

		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("delegates to store", func(t *testing.T) {
		svc := viagens.NewPresencaService(fakeViagemReservaStore{
			listReservasFn: func(_ context.Context, viagemID int64) ([]viagens.ViagemReservaComReserva, error) {
				if viagemID != 10 {
					t.Fatalf("unexpected viagemID: %d", viagemID)
				}
				return []viagens.ViagemReservaComReserva{sampleViagemReservaComReserva()}, nil
			},
		})

		reservas, err := svc.ListReservasByViagem(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(reservas) != 1 {
			t.Fatalf("expected 1 reserva, got %d", len(reservas))
		}
	})
}

func TestPresencaService_AtualizarPresenca(t *testing.T) {
	t.Run("requires valid ids and status", func(t *testing.T) {
		svc := viagens.NewPresencaService(fakeViagemReservaStore{})

		if _, err := svc.AtualizarPresenca(context.Background(), 0, 20, viagens.StatusPresencaEmbarcou); err == nil {
			t.Fatal("expected viagem id error")
		}
		if _, err := svc.AtualizarPresenca(context.Background(), 10, 0, viagens.StatusPresencaEmbarcou); err == nil {
			t.Fatal("expected reserva id error")
		}
		if _, err := svc.AtualizarPresenca(context.Background(), 10, 20, viagens.StatusPresencaAguardando); err == nil {
			t.Fatal("expected status error")
		}
	})

	t.Run("updates status when changed", func(t *testing.T) {
		svc := viagens.NewPresencaService(fakeViagemReservaStore{
			updatePresencaFn: func(_ context.Context, viagemID, reservaID int64, updateFunc func(*viagens.ViagemReserva) (bool, error)) (*viagens.ViagemReserva, error) {
				if viagemID != 10 || reservaID != 20 {
					t.Fatalf("unexpected ids: %d %d", viagemID, reservaID)
				}
				vr := sampleViagemReserva()
				changed, err := updateFunc(&vr)
				if err != nil {
					t.Fatalf("unexpected update error: %v", err)
				}
				if !changed || vr.StatusPresenca != viagens.StatusPresencaEmbarcou {
					t.Fatalf("expected embarcou change, changed=%v status=%s", changed, vr.StatusPresenca)
				}
				return &vr, nil
			},
		})

		vr, err := svc.AtualizarPresenca(context.Background(), 10, 20, viagens.StatusPresencaEmbarcou)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if vr.StatusPresenca != viagens.StatusPresencaEmbarcou {
			t.Fatalf("unexpected status: %s", vr.StatusPresenca)
		}
	})

	t.Run("same status does not change", func(t *testing.T) {
		svc := viagens.NewPresencaService(fakeViagemReservaStore{
			updatePresencaFn: func(_ context.Context, _ int64, _ int64, updateFunc func(*viagens.ViagemReserva) (bool, error)) (*viagens.ViagemReserva, error) {
				vr := sampleViagemReserva()
				vr.StatusPresenca = viagens.StatusPresencaFaltou
				changed, err := updateFunc(&vr)
				if err != nil {
					t.Fatalf("unexpected update error: %v", err)
				}
				if changed {
					t.Fatal("expected no change")
				}
				return &vr, nil
			},
		})

		_, err := svc.AtualizarPresenca(context.Background(), 10, 20, viagens.StatusPresencaFaltou)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("canceled presence cannot be changed", func(t *testing.T) {
		svc := viagens.NewPresencaService(fakeViagemReservaStore{
			updatePresencaFn: func(_ context.Context, _ int64, _ int64, updateFunc func(*viagens.ViagemReserva) (bool, error)) (*viagens.ViagemReserva, error) {
				vr := sampleViagemReserva()
				vr.StatusPresenca = viagens.StatusPresencaCancelado
				_, err := updateFunc(&vr)
				if err == nil {
					t.Fatal("expected update error")
				}
				return nil, err
			},
		})

		_, err := svc.AtualizarPresenca(context.Background(), 10, 20, viagens.StatusPresencaEmbarcou)

		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("store error is returned", func(t *testing.T) {
		storeErr := errors.New("db")
		svc := viagens.NewPresencaService(fakeViagemReservaStore{
			updatePresencaFn: func(_ context.Context, _ int64, _ int64, _ func(*viagens.ViagemReserva) (bool, error)) (*viagens.ViagemReserva, error) {
				return nil, storeErr
			},
		})

		_, err := svc.AtualizarPresenca(context.Background(), 10, 20, viagens.StatusPresencaFaltou)

		if !errors.Is(err, storeErr) {
			t.Fatalf("expected store error, got %v", err)
		}
	})
}
