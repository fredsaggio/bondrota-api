package viagens_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

type fakeViagemStore struct {
	createViagemFn                           func(ctx context.Context, input viagens.ViagemInput) (*viagens.Viagem, error)
	getViagemByIDFn                          func(ctx context.Context, viagemID int64) (*viagens.ViagemComCiclo, error)
	listViagensFn                            func(ctx context.Context) ([]viagens.ViagemComCiclo, error)
	listViagensByCicloFn                     func(ctx context.Context, cicloID int64) ([]viagens.Viagem, error)
	listHorariosByViagemFn                   func(ctx context.Context, viagemID int64) ([]viagens.ViagemHorario, error)
	registrarHorarioViagemFn                 func(ctx context.Context, viagemID int64, tipo viagens.TipoHorarioViagem, horario time.Time) (*viagens.ViagemHorario, error)
	atualizarStatusERegistrarHorarioViagemFn func(ctx context.Context, viagemID int64, from viagens.StatusViagem, to viagens.StatusViagem, tipo viagens.TipoHorarioViagem, horario time.Time) (*viagens.Viagem, error)
	updateViagemFn                           func(ctx context.Context, viagemID int64, updateFunc func(*viagens.Viagem) (bool, error)) (*viagens.Viagem, error)
}

func (s fakeViagemStore) CreateViagem(ctx context.Context, input viagens.ViagemInput) (*viagens.Viagem, error) {
	return s.createViagemFn(ctx, input)
}

func (s fakeViagemStore) GetViagemByID(ctx context.Context, viagemID int64) (*viagens.ViagemComCiclo, error) {
	return s.getViagemByIDFn(ctx, viagemID)
}

func (s fakeViagemStore) ListViagens(ctx context.Context) ([]viagens.ViagemComCiclo, error) {
	return s.listViagensFn(ctx)
}

func (s fakeViagemStore) ListViagensByCiclo(ctx context.Context, cicloID int64) ([]viagens.Viagem, error) {
	return s.listViagensByCicloFn(ctx, cicloID)
}

func (s fakeViagemStore) ListHorariosByViagem(ctx context.Context, viagemID int64) ([]viagens.ViagemHorario, error) {
	return s.listHorariosByViagemFn(ctx, viagemID)
}

func (s fakeViagemStore) RegistrarHorarioViagem(ctx context.Context, viagemID int64, tipo viagens.TipoHorarioViagem, horario time.Time) (*viagens.ViagemHorario, error) {
	return s.registrarHorarioViagemFn(ctx, viagemID, tipo, horario)
}

func (s fakeViagemStore) AtualizarStatusERegistrarHorarioViagem(ctx context.Context, viagemID int64, from viagens.StatusViagem, to viagens.StatusViagem, tipo viagens.TipoHorarioViagem, horario time.Time) (*viagens.Viagem, error) {
	return s.atualizarStatusERegistrarHorarioViagemFn(ctx, viagemID, from, to, tipo, horario)
}

func (s fakeViagemStore) UpdateViagem(ctx context.Context, viagemID int64, updateFunc func(*viagens.Viagem) (bool, error)) (*viagens.Viagem, error) {
	return s.updateViagemFn(ctx, viagemID, updateFunc)
}

func TestViagemService_GetAndList(t *testing.T) {
	t.Run("get by id delegates to store", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{
			getViagemByIDFn: func(_ context.Context, viagemID int64) (*viagens.ViagemComCiclo, error) {
				if viagemID != 10 {
					t.Fatalf("unexpected viagemID: %d", viagemID)
				}
				v := sampleViagemComCiclo()
				return &v, nil
			},
		})

		viagem, err := svc.GetByID(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if viagem.Viagem.ID != 10 {
			t.Fatalf("unexpected viagem: %+v", viagem)
		}
	})

	t.Run("list delegates to store", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{
			listViagensFn: func(_ context.Context) ([]viagens.ViagemComCiclo, error) {
				return []viagens.ViagemComCiclo{sampleViagemComCiclo()}, nil
			},
		})

		viagensList, err := svc.List(context.Background())

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(viagensList) != 1 {
			t.Fatalf("expected 1 viagem, got %d", len(viagensList))
		}
	})
}

func TestViagemService_ListHorariosByViagem(t *testing.T) {
	t.Run("requires valid id", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{})

		_, err := svc.ListHorariosByViagem(context.Background(), 0)

		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("delegates to store", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{
			listHorariosByViagemFn: func(_ context.Context, viagemID int64) ([]viagens.ViagemHorario, error) {
				if viagemID != 10 {
					t.Fatalf("unexpected viagemID: %d", viagemID)
				}
				return []viagens.ViagemHorario{
					{ID: 1, ViagemID: 10, Tipo: viagens.TipoHorarioPartidaPrevista, Horario: testTime()},
				}, nil
			},
		})

		horarios, err := svc.ListHorariosByViagem(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(horarios) != 1 {
			t.Fatalf("expected 1 horario, got %d", len(horarios))
		}
	})
}

func TestViagemService_Iniciar(t *testing.T) {
	t.Run("requires valid id", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{})

		_, err := svc.Iniciar(context.Background(), 0)

		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("transitions from programada to em andamento and records start time", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{
			atualizarStatusERegistrarHorarioViagemFn: func(_ context.Context, viagemID int64, from viagens.StatusViagem, to viagens.StatusViagem, tipo viagens.TipoHorarioViagem, horario time.Time) (*viagens.Viagem, error) {
				if viagemID != 10 || from != viagens.StatusViagemProgramada || to != viagens.StatusViagemEmAndamento || tipo != viagens.TipoHorarioInicioReal {
					t.Fatalf("unexpected status update: %d %s %s %s", viagemID, from, to, tipo)
				}
				if horario.IsZero() {
					t.Fatal("expected non-zero horario")
				}
				v := sampleViagem()
				v.Status = to
				return &v, nil
			},
		})

		viagem, err := svc.Iniciar(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if viagem.Status != viagens.StatusViagemEmAndamento {
			t.Fatalf("unexpected status: %s", viagem.Status)
		}
	})
}

func TestViagemService_Concluir(t *testing.T) {
	t.Run("transitions from em andamento to concluida and records end time", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{
			atualizarStatusERegistrarHorarioViagemFn: func(_ context.Context, viagemID int64, from viagens.StatusViagem, to viagens.StatusViagem, tipo viagens.TipoHorarioViagem, horario time.Time) (*viagens.Viagem, error) {
				if viagemID != 10 || from != viagens.StatusViagemEmAndamento || to != viagens.StatusViagemConcluida || tipo != viagens.TipoHorarioFimReal {
					t.Fatalf("unexpected status update: %d %s %s %s", viagemID, from, to, tipo)
				}
				if horario.IsZero() {
					t.Fatal("expected non-zero horario")
				}
				v := sampleViagem()
				v.Status = to
				return &v, nil
			},
		})

		viagem, err := svc.Concluir(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if viagem.Status != viagens.StatusViagemConcluida {
			t.Fatalf("unexpected status: %s", viagem.Status)
		}
	})
}

func TestViagemService_Cancelar(t *testing.T) {
	t.Run("changes programada to cancelada", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{
			updateViagemFn: func(_ context.Context, viagemID int64, updateFunc func(*viagens.Viagem) (bool, error)) (*viagens.Viagem, error) {
				if viagemID != 10 {
					t.Fatalf("unexpected viagemID: %d", viagemID)
				}
				v := sampleViagem()
				changed, err := updateFunc(&v)
				if err != nil {
					t.Fatalf("unexpected update error: %v", err)
				}
				if !changed || v.Status != viagens.StatusViagemCancelada {
					t.Fatalf("expected cancel change, changed=%v status=%s", changed, v.Status)
				}
				return &v, nil
			},
		})

		viagem, err := svc.Cancelar(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if viagem.Status != viagens.StatusViagemCancelada {
			t.Fatalf("unexpected status: %s", viagem.Status)
		}
	})

	t.Run("does not change already canceled", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{
			updateViagemFn: func(_ context.Context, _ int64, updateFunc func(*viagens.Viagem) (bool, error)) (*viagens.Viagem, error) {
				v := sampleViagem()
				v.Status = viagens.StatusViagemCancelada
				changed, err := updateFunc(&v)
				if err != nil {
					t.Fatalf("unexpected update error: %v", err)
				}
				if changed {
					t.Fatal("expected no change")
				}
				return &v, nil
			},
		})

		_, err := svc.Cancelar(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("cannot cancel concluded viagem", func(t *testing.T) {
		svc := viagens.NewViagemService(fakeViagemStore{
			updateViagemFn: func(_ context.Context, _ int64, updateFunc func(*viagens.Viagem) (bool, error)) (*viagens.Viagem, error) {
				v := sampleViagem()
				v.Status = viagens.StatusViagemConcluida
				_, err := updateFunc(&v)
				if err == nil {
					t.Fatal("expected update error")
				}
				return nil, err
			},
		})

		_, err := svc.Cancelar(context.Background(), 10)

		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("store error is returned", func(t *testing.T) {
		storeErr := errors.New("db")
		svc := viagens.NewViagemService(fakeViagemStore{
			updateViagemFn: func(_ context.Context, _ int64, _ func(*viagens.Viagem) (bool, error)) (*viagens.Viagem, error) {
				return nil, storeErr
			},
		})

		_, err := svc.Cancelar(context.Background(), 10)

		if !errors.Is(err, storeErr) {
			t.Fatalf("expected store error, got %v", err)
		}
	})
}
