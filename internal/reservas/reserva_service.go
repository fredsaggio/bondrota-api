package reservas

import (
	"context"
	"time"
)

type reservaService struct {
	store                  ReservaStore
	rotaInvalidator        RotaDinamicaInvalidator
	location               *time.Location
	now                    func() time.Time
	antecedenciaFechamento time.Duration
}

const DefaultAntecedenciaFechamentoReserva = 30 * time.Minute

func NewReservaService(store ReservaStore, config ReservaServiceConfig, rotaInvalidator ...RotaDinamicaInvalidator) ReservaService {
	if config.Location == nil {
		config.Location = time.UTC
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.AntecedenciaFechamento <= 0 {
		config.AntecedenciaFechamento = DefaultAntecedenciaFechamentoReserva
	}

	s := &reservaService{
		store:                  store,
		location:               config.Location,
		now:                    config.Now,
		antecedenciaFechamento: config.AntecedenciaFechamento,
	}
	if len(rotaInvalidator) > 0 {
		s.rotaInvalidator = rotaInvalidator[0]
	}
	return s
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
	if err := s.validarPrazoReserva(ctx, input.DataViagem, input.Turno, input.DestinoID, input.Sentido); err != nil {
		return nil, err
	}

	return s.store.Create(ctx, input)
}

func (s *reservaService) ConsultarDisponibilidade(ctx context.Context, input DisponibilidadeReservaInput) (*DisponibilidadeReserva, error) {
	if err := validateDisponibilidadeInput(input); err != nil {
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

	return s.consultarDisponibilidade(ctx, input.DataViagem, turno, snapshot.DestinoID, input.Sentido)
}

func (s *reservaService) GetByID(ctx context.Context, reservaID int64) (*Reserva, error) {
	return s.store.GetByID(ctx, reservaID)
}

func (s *reservaService) List(ctx context.Context, params ReservaListParams) (ReservaListResult, error) {
	return s.store.List(ctx, params)
}

func (s *reservaService) Resumo(ctx context.Context) (ReservaResumo, error) {
	return s.store.Resumo(ctx)
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
	reserva, err := s.store.Update(ctx, reservaID, func(r *Reserva) (bool, error) {
		changed, err := updateFunc(r)
		if err != nil || !changed {
			return changed, err
		}
		if err := s.validateReserva(ctx, r); err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.invalidarRotaDinamicaPorReserva(ctx, reserva.ID); err != nil {
		return nil, err
	}
	return reserva, nil
}

func (s *reservaService) Cancel(ctx context.Context, reservaID int64) (*Reserva, error) {
	reserva, err := s.store.Update(ctx, reservaID, func(r *Reserva) (bool, error) {
		if r.Status == StatusCancelada {
			return false, nil
		}
		r.Status = StatusCancelada
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.invalidarRotaDinamicaPorReserva(ctx, reserva.ID); err != nil {
		return nil, err
	}
	return reserva, nil
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
	if reserva.Status == StatusConfirmada {
		return s.validarPrazoReserva(ctx, reserva.DataViagem, reserva.Turno, reserva.DestinoID, reserva.Sentido)
	}

	return nil
}

func validateDisponibilidadeInput(input DisponibilidadeReservaInput) error {
	return validateCreateInput(ReservaInput{
		VinculoID:  input.VinculoID,
		DataViagem: input.DataViagem,
		Turno:      input.Turno,
		Sentido:    input.Sentido,
	})
}

func (s *reservaService) validarPrazoReserva(ctx context.Context, dataViagem time.Time, turno TurnoReserva, destinoID int64, sentido SentidoReserva) error {
	disponibilidade, err := s.consultarDisponibilidade(ctx, dataViagem, turno, destinoID, sentido)
	if err != nil {
		return err
	}
	if !disponibilidade.Disponivel {
		return ErrPrazoReservaEncerrado
	}
	return nil
}

func (s *reservaService) consultarDisponibilidade(ctx context.Context, dataViagem time.Time, turno TurnoReserva, destinoID int64, sentido SentidoReserva) (*DisponibilidadeReserva, error) {
	horarioPartida, err := s.store.GetHorarioPartida(ctx, destinoID, turno, sentido)
	if err != nil {
		return nil, err
	}

	partidaEm := montarHorarioNaData(dataViagem, horarioPartida, s.location)
	fechamentoEm := partidaEm.Add(-s.antecedenciaFechamento)
	consultadoEm := s.now().In(s.location)

	return &DisponibilidadeReserva{
		DataViagem:   dataViagem,
		Turno:        turno,
		Sentido:      sentido,
		PartidaEm:    partidaEm,
		FechamentoEm: fechamentoEm,
		ConsultadoEm: consultadoEm,
		Disponivel:   consultadoEm.Before(fechamentoEm),
	}, nil
}

func montarHorarioNaData(data time.Time, horario time.Duration, location *time.Location) time.Time {
	horas := int(horario / time.Hour)
	minutos := int((horario % time.Hour) / time.Minute)
	segundos := int((horario % time.Minute) / time.Second)
	return time.Date(data.Year(), data.Month(), data.Day(), horas, minutos, segundos, 0, location)
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

func (s *reservaService) invalidarRotaDinamicaPorReserva(ctx context.Context, reservaID int64) error {
	if s.rotaInvalidator == nil {
		return nil
	}
	return s.rotaInvalidator.InvalidarPorReserva(ctx, reservaID)
}
