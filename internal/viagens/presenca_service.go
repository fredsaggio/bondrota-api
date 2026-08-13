package viagens

import (
	"context"
	"errors"
)

type presencaService struct {
	store ViagemReservaStore
}

func NewPresencaService(store ViagemReservaStore) PresencaService {
	return &presencaService{store: store}
}

func (s *presencaService) ListReservasByViagem(ctx context.Context, viagemID int64) ([]ViagemReservaComReserva, error) {
	if viagemID <= 0 {
		return nil, errors.New("Selecione a viagem.")
	}

	return s.store.ListReservasByViagem(ctx, viagemID)
}

func (s *presencaService) AtualizarPresenca(ctx context.Context, viagemID, reservaID int64, status StatusPresencaViagem) (*ViagemReserva, error) {
	if viagemID <= 0 {
		return nil, errors.New("Selecione a viagem.")
	}
	if reservaID <= 0 {
		return nil, errors.New("Selecione a reserva.")
	}
	if !isValidStatusPresencaUpdate(status) {
		return nil, errors.New("Selecione uma situação de presença válida.")
	}

	return s.store.UpdatePresenca(ctx, viagemID, reservaID, func(vr *ViagemReserva) (bool, error) {
		if vr.StatusPresenca == status {
			return false, nil
		}
		if vr.StatusPresenca == StatusPresencaCancelado {
			return false, errors.New("Uma presença cancelada não pode ser alterada.")
		}

		vr.StatusPresenca = status
		return true, nil
	})
}

func isValidStatusPresencaUpdate(status StatusPresencaViagem) bool {
	return status == StatusPresencaEmbarcou ||
		status == StatusPresencaFaltou ||
		status == StatusPresencaCancelado
}
