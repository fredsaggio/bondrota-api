package reservas

import (
	"context"
	"time"
)

type reservaService struct {
	store ReservaStore
}

func NewReservaService(store ReservaStore) ReservaService {
	return &reservaService{store: store}
}

func (s *reservaService) Create(ctx context.Context, input ReservaInput) (*Reserva, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	snapshot, err := s.store.GetVinculoSnapshot(ctx, input.VinculoID)
	if err != nil {
		return nil, err
	}
	if input.ClienteID > 0 && snapshot.ClienteID != input.ClienteID {
		return nil, ErrVinculoNotFound
	}

	turno, err := resolveTurno(snapshot.Turno, input.Turno)
	if err != nil {
		return nil, err
	}

	input.ClienteID = snapshot.ClienteID
	input.Turno = turno
	input.DestinoID = snapshot.DestinoID
	input.RotaInternaID = snapshot.RotaInternaID
	input.Cidade = snapshot.Cidade

	return s.store.Create(ctx, input)
}

func (s *reservaService) GetByID(ctx context.Context, reservaID int64) (*Reserva, error) {
	return s.store.GetByID(ctx, reservaID)
}

func (s *reservaService) List(ctx context.Context) ([]Reserva, error) {
	return s.store.List(ctx)
}

func (s *reservaService) ListByCliente(ctx context.Context, clienteID int64) ([]Reserva, error) {
	return s.store.ListByCliente(ctx, clienteID)
}

func (s *reservaService) ListByVinculo(ctx context.Context, clienteID, vinculoID int64) ([]Reserva, error) {
	snapshot, err := s.store.GetVinculoSnapshot(ctx, vinculoID)
	if err != nil {
		return nil, err
	}
	if snapshot.ClienteID != clienteID {
		return nil, ErrVinculoNotFound
	}
	return s.store.ListByVinculo(ctx, clienteID, vinculoID)
}

func (s *reservaService) Update(ctx context.Context, reservaID int64, updateFunc func(*Reserva) (bool, error)) (*Reserva, error) {
	return s.store.Update(ctx, reservaID, func(r *Reserva) (bool, error) {
		changed, err := updateFunc(r)
		if err != nil || !changed {
			return changed, err
		}
		if err := s.validateReserva(ctx, r); err != nil {
			return false, err
		}
		return true, nil
	})
}

func (s *reservaService) Cancel(ctx context.Context, reservaID int64) (*Reserva, error) {
	return s.store.Update(ctx, reservaID, func(r *Reserva) (bool, error) {
		if r.Status == StatusCancelada {
			return false, nil
		}
		r.Status = StatusCancelada
		return true, nil
	})
}

func (s *reservaService) Delete(ctx context.Context, reservaID int64) error {
	return s.store.Delete(ctx, reservaID)
}

func validateCreateInput(input ReservaInput) error {
	if input.VinculoID <= 0 {
		return ErrVinculoIDObrigatorio
	}
	if input.DataViagem.IsZero() {
		return ErrDataObrigatoria
	}
	if !isValidSentido(input.Sentido) {
		return ErrSentidoInvalido
	}
	if input.Turno != "" && !isOperationalTurno(input.Turno) {
		return ErrTurnoInvalido
	}
	return nil
}

func (s *reservaService) validateReserva(ctx context.Context, reserva *Reserva) error {
	if reserva.DataViagem.IsZero() {
		return ErrDataObrigatoria
	}
	if !isValidSentido(reserva.Sentido) {
		return ErrSentidoInvalido
	}
	if !isValidStatus(reserva.Status) {
		return ErrStatusInvalido
	}

	snapshot, err := s.store.GetVinculoSnapshot(ctx, reserva.VinculoID)
	if err != nil {
		return err
	}

	turno, err := resolveTurno(snapshot.Turno, reserva.Turno)
	if err != nil {
		return err
	}
	reserva.Turno = turno

	return nil
}

func resolveTurno(vinculoTurno, requestedTurno TurnoReserva) (TurnoReserva, error) {
	if vinculoTurno == TurnoIntegral {
		if requestedTurno == "" {
			return "", ErrTurnoObrigatorio
		}
		if !isOperationalTurno(requestedTurno) {
			return "", ErrTurnoInvalido
		}
		return requestedTurno, nil
	}

	if !isOperationalTurno(vinculoTurno) {
		return "", ErrTurnoInvalido
	}
	if requestedTurno != "" && requestedTurno != vinculoTurno {
		return "", ErrTurnoIncompativel
	}

	return vinculoTurno, nil
}

func isOperationalTurno(turno TurnoReserva) bool {
	return turno == TurnoMatutino || turno == TurnoVespertino || turno == TurnoNoturno
}

func isValidSentido(sentido SentidoReserva) bool {
	return sentido == SentidoIda || sentido == SentidoVolta
}

func isValidStatus(status StatusReserva) bool {
	return status == StatusConfirmada || status == StatusCancelada
}

func sameDate(a, b time.Time) bool {
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}
