package viagens

import (
	"context"
	"errors"
	"time"
)

type viagemService struct {
	store ViagemStore
	now   func() time.Time
}

func NewViagemService(store ViagemStore) ViagemService {
	return &viagemService{
		store: store,
		now:   time.Now,
	}
}

func (s *viagemService) GetByID(ctx context.Context, viagemID int64) (*ViagemComCiclo, error) {
	return s.store.GetViagemByID(ctx, viagemID)
}

func (s *viagemService) List(ctx context.Context) ([]ViagemComCiclo, error) {
	return s.store.ListViagens(ctx)
}

func (s *viagemService) ListHorariosByViagem(ctx context.Context, viagemID int64) ([]ViagemHorario, error) {
	if viagemID <= 0 {
		return nil, errors.New("viagem_id is required")
	}

	return s.store.ListHorariosByViagem(ctx, viagemID)
}

func (s *viagemService) Iniciar(ctx context.Context, viagemID int64) (*Viagem, error) {
	if viagemID <= 0 {
		return nil, errors.New("viagem_id is required")
	}

	return s.store.AtualizarStatusERegistrarHorarioViagem(
		ctx,
		viagemID,
		StatusViagemProgramada,
		StatusViagemEmAndamento,
		TipoHorarioInicioReal,
		s.now(),
	)
}

func (s *viagemService) Concluir(ctx context.Context, viagemID int64) (*Viagem, error) {
	if viagemID <= 0 {
		return nil, errors.New("viagem_id is required")
	}

	return s.store.AtualizarStatusERegistrarHorarioViagem(
		ctx,
		viagemID,
		StatusViagemEmAndamento,
		StatusViagemConcluida,
		TipoHorarioFimReal,
		s.now(),
	)
}

func (s *viagemService) Cancelar(ctx context.Context, viagemID int64) (*Viagem, error) {
	if viagemID <= 0 {
		return nil, errors.New("viagem_id is required")
	}

	return s.store.UpdateViagem(ctx, viagemID, func(v *Viagem) (bool, error) {
		switch v.Status {
		case StatusViagemCancelada:
			return false, nil
		case StatusViagemConcluida:
			return false, errors.New("viagem concluida nao pode ser cancelada")
		}

		v.Status = StatusViagemCancelada
		return true, nil
	})
}
