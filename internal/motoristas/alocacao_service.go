package motoristas

import (
	"context"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

type AlocacaoService struct {
	store AlocacaoMotoristaStore
}

func NewAlocacaoService(store AlocacaoMotoristaStore) *AlocacaoService {
	return &AlocacaoService{store: store}
}

func (s *AlocacaoService) Alocar(ctx context.Context, input AlocarMotoristasInput) ([]Motorista, error) {
	if err := validateAlocarMotoristasInput(input); err != nil {
		return nil, err
	}

	motoristas, err := s.store.ListDisponiveisParaAlocacao(ctx, MotoristasDisponiveisFiltro{
		MunicipioTrabalhoID: input.MunicipioTrabalhoID,
		DataViagem:          input.DataViagem,
		Turno:               input.Turno,
		Limit:               input.Quantidade,
	})
	if err != nil {
		return nil, err
	}
	if len(motoristas) < input.Quantidade {
		return nil, fmt.Errorf("%w: motoristas disponiveis insuficientes", brerror.ErrNotFound)
	}

	return motoristas, nil
}

func validateAlocarMotoristasInput(input AlocarMotoristasInput) error {
	if input.MunicipioTrabalhoID <= 0 {
		return fmt.Errorf("%w: municipio_trabalho_id is required", brerror.ErrInvalidInput)
	}
	if input.DataViagem.IsZero() {
		return fmt.Errorf("%w: data_viagem is required", brerror.ErrInvalidInput)
	}
	if !isTurnoOperacional(input.Turno) {
		return fmt.Errorf("%w: turno must be MT, VT or NT", brerror.ErrInvalidInput)
	}
	if input.Quantidade <= 0 {
		return fmt.Errorf("%w: quantidade must be greater than zero", brerror.ErrInvalidInput)
	}
	return nil
}

func isTurnoOperacional(turno Turno) bool {
	switch turno {
	case TurnoMatutino, TurnoVespertino, TurnoNoturno:
		return true
	default:
		return false
	}
}
