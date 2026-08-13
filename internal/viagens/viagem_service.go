package viagens

import (
	"context"
	"errors"
	"time"
)

type viagemService struct {
	store    ViagemStore
	now      func() time.Time
	location *time.Location
}

func NewViagemService(store ViagemStore, config ViagemServiceConfig) ViagemService {
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Location == nil {
		config.Location = time.UTC
	}
	return &viagemService{
		store:    store,
		now:      config.Now,
		location: config.Location,
	}
}

func (s *viagemService) GetByID(ctx context.Context, viagemID int64) (*ViagemComCiclo, error) {
	return s.store.GetViagemByID(ctx, viagemID)
}

func (s *viagemService) List(ctx context.Context, params ViagemListParams) (ViagemListResult, error) {
	return s.store.ListViagens(ctx, params)
}

func (s *viagemService) Resumo(ctx context.Context) (ViagemResumo, error) {
	// "Hoje" no fuso da operacao, nao no do servidor: perto da meia-noite os dois
	// divergem e a contagem do dia sairia de um dia errado.
	local := s.now().In(s.location)
	hoje := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.location)
	return s.store.ResumoViagens(ctx, hoje)
}

func (s *viagemService) ListHorariosByViagem(ctx context.Context, viagemID int64) ([]ViagemHorario, error) {
	if viagemID <= 0 {
		return nil, errors.New("Selecione a viagem.")
	}

	return s.store.ListHorariosByViagem(ctx, viagemID)
}

func (s *viagemService) Iniciar(ctx context.Context, viagemID int64) (*Viagem, error) {
	if viagemID <= 0 {
		return nil, errors.New("Selecione a viagem.")
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
		return nil, errors.New("Selecione a viagem.")
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
		return nil, errors.New("Selecione a viagem.")
	}

	return s.store.UpdateViagem(ctx, viagemID, func(v *Viagem) (bool, error) {
		switch v.Status {
		case StatusViagemCancelada:
			return false, nil
		case StatusViagemConcluida:
			return false, errors.New("Uma viagem já concluída não pode ser cancelada.")
		}

		v.Status = StatusViagemCancelada
		return true, nil
	})
}
