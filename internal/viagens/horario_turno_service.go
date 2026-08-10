package viagens

import (
	"context"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

type horarioTurnoViagemService struct {
	store HorarioTurnoViagemStore
}

func NewHorarioTurnoViagemService(store HorarioTurnoViagemStore) HorarioTurnoViagemService {
	return &horarioTurnoViagemService{store: store}
}

func (s *horarioTurnoViagemService) Create(ctx context.Context, input HorarioTurnoViagemInput) (*HorarioTurnoViagem, error) {
	input = normalizeHorarioTurnoInput(input)
	if err := validateHorarioTurnoInput(input); err != nil {
		return nil, err
	}

	return s.store.Create(ctx, input)
}

func (s *horarioTurnoViagemService) GetByID(ctx context.Context, id int64) (*HorarioTurnoViagem, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id is required", brerror.ErrInvalidInput)
	}
	return s.store.GetByID(ctx, id)
}

func (s *horarioTurnoViagemService) List(ctx context.Context) ([]HorarioTurnoViagem, error) {
	return s.store.List(ctx)
}

func (s *horarioTurnoViagemService) Update(ctx context.Context, id int64, updateFunc func(*HorarioTurnoViagem) (bool, error)) (*HorarioTurnoViagem, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id is required", brerror.ErrInvalidInput)
	}

	return s.store.Update(ctx, id, func(horario *HorarioTurnoViagem) (bool, error) {
		changed, err := updateFunc(horario)
		if err != nil {
			return false, err
		}
		if !changed {
			return false, nil
		}

		input := HorarioTurnoViagemInput{
			MunicipioDestinoID: horario.MunicipioDestinoID,
			Turno:              horario.Turno,
			HorarioIda:         horario.HorarioIda,
			HorarioVolta:       horario.HorarioVolta,
		}
		input = normalizeHorarioTurnoInput(input)
		if err := validateHorarioTurnoInput(input); err != nil {
			return false, err
		}

		horario.MunicipioDestinoID = input.MunicipioDestinoID
		horario.Turno = input.Turno
		horario.HorarioIda = input.HorarioIda
		horario.HorarioVolta = input.HorarioVolta

		return true, nil
	})
}

func (s *horarioTurnoViagemService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id is required", brerror.ErrInvalidInput)
	}
	return s.store.Delete(ctx, id)
}

func normalizeHorarioTurnoInput(input HorarioTurnoViagemInput) HorarioTurnoViagemInput {
	return input
}

func validateHorarioTurnoInput(input HorarioTurnoViagemInput) error {
	if input.MunicipioDestinoID <= 0 {
		return fmt.Errorf("%w: municipio_destino_id is required", brerror.ErrInvalidInput)
	}
	if !isOperationalTurnoViagem(input.Turno) {
		return fmt.Errorf("%w: turno must be MT, VT or NT", brerror.ErrInvalidInput)
	}
	if input.HorarioIda < 0 || input.HorarioIda >= 24*time.Hour {
		return fmt.Errorf("%w: horario_ida must be between 00:00 and 23:59", brerror.ErrInvalidInput)
	}
	if input.HorarioVolta < 0 || input.HorarioVolta >= 24*time.Hour {
		return fmt.Errorf("%w: horario_volta must be between 00:00 and 23:59", brerror.ErrInvalidInput)
	}
	if input.HorarioVolta <= input.HorarioIda {
		return fmt.Errorf("%w: horario_volta must be after horario_ida", brerror.ErrInvalidInput)
	}
	return nil
}
