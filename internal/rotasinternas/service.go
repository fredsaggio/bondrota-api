package rotasinternas

import (
	"context"
	"errors"
	"fmt"
)

var ErrOrdemDuplicada = errors.New("ordens das paradas devem ser únicas")

type rotaInternaService struct {
	store RotaInternaStore
}

func NewRotaInternaService(store RotaInternaStore) RotaInternaService {
	return &rotaInternaService{store: store}
}

func (s *rotaInternaService) Create(ctx context.Context, input CreateRotaInternaInput) (*RotaInterna, error) {
	const op = "service/rotaInternaService.Create"

	if err := validateParadas(input.Paradas); err != nil {
		return nil, err
	}

	rota, err := s.store.Create(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return rota, nil
}

func validateParadas(paradas []ParadaInput) error {
	if len(paradas) == 0 {
		return errors.New("rota deve ter ao menos uma parada")
	}

	ordens := make(map[int]bool, len(paradas))
	for _, p := range paradas {
		if p.Nome == "" {
			return errors.New("nome da parada é obrigatório")
		}
		if ordens[p.Ordem] {
			return ErrOrdemDuplicada
		}
		ordens[p.Ordem] = true
	}

	return nil
}
