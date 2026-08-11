package clientes

import (
	"context"
	"strings"
)

type vinculoService struct {
	store VinculoStore
}

func NewVinculoService(store VinculoStore) VinculoService {
	return &vinculoService{store: store}
}

func (s *vinculoService) Create(ctx context.Context, input VinculoInput) (*Vinculo, error) {
	if err := validateVinculo(input.Tipo, input.Turno, input.Curso, input.HorariosFixos); err != nil {
		return nil, err
	}
	return s.store.Create(ctx, input)
}

func (s *vinculoService) GetByID(ctx context.Context, vinculoID int64) (*Vinculo, error) {
	return s.store.GetByID(ctx, vinculoID)
}

func (s *vinculoService) List(ctx context.Context) ([]VinculoComCliente, error) {
	return s.store.List(ctx)
}

func (s *vinculoService) ListByCliente(ctx context.Context, clienteID int64) ([]Vinculo, error) {
	return s.store.ListByCliente(ctx, clienteID)
}

func (s *vinculoService) Update(ctx context.Context, vinculoID int64, input VinculoUpdateInput) (*Vinculo, error) {
	if err := validateVinculo(input.Tipo, input.Turno, input.Curso, input.HorariosFixos); err != nil {
		return nil, err
	}
	return s.store.Update(ctx, vinculoID, input)
}

func (s *vinculoService) Delete(ctx context.Context, vinculoID int64) error {
	return s.store.Delete(ctx, vinculoID)
}

func validateVinculo(tipo TipoConta, turno TurnoCliente, curso string, dias []DiaSemana) error {
	switch tipo {
	case TipoEstudante, TipoEstagio:
	default:
		return ErrTipoInvalido
	}

	switch turno {
	case TurnoMatutino, TurnoVespertino, TurnoNoturno, TurnoIntegral:
	default:
		return ErrTurnoInvalido
	}

	if tipo == TipoEstudante && strings.TrimSpace(curso) == "" {
		return ErrCursoObrigatorio
	}

	seen := map[DiaSemana]struct{}{}
	for _, dia := range dias {
		if dia < Segunda || dia > Sexta {
			return ErrDiaInvalido
		}
		if _, ok := seen[dia]; ok {
			return ErrDiaDuplicado
		}
		seen[dia] = struct{}{}
	}

	return nil
}
