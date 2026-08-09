package rotasinternas

import (
	"context"
	"fmt"
)

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

func (s *rotaInternaService) UpdateParadas(ctx context.Context, rotaInternaID int64, input UpdateParadasInput) (*RotaInterna, error) {
	const op = "service/rotaInternaService.UpdateParadas"

	if err := validateParadas(input.Paradas); err != nil {
		return nil, err
	}

	rota, err := s.store.UpdateParadas(ctx, rotaInternaID, input)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return rota, nil
}

func (s *rotaInternaService) GetByID(ctx context.Context, rotaInternaID int64) (*RotaInterna, error) {
	return s.store.GetByID(ctx, rotaInternaID)
}

func (s *rotaInternaService) List(ctx context.Context) ([]RotaInterna, error) {
	return s.store.List(ctx)
}

func (s *rotaInternaService) ListByCity(ctx context.Context, cidade string) ([]RotaInterna, error) {
	return s.store.ListByCity(ctx, cidade)
}

func (s *rotaInternaService) Delete(ctx context.Context, rotaInternaID int64) error {
	return s.store.Delete(ctx, rotaInternaID)
}

func validateParadas(paradas []ParadaInput) error {
	if len(paradas) == 0 {
		return ErrSemParadas
	}

	ordens := make(map[int]bool, len(paradas))
	for _, p := range paradas {
		if p.ParadaID <= 0 {
			return ErrParadaInvalida
		}
		if p.Ordem <= 0 {
			return ErrParadaInvalida
		}
		if ordens[p.Ordem] {
			return ErrOrdemDuplicada
		}
		ordens[p.Ordem] = true
	}

	return nil
}
