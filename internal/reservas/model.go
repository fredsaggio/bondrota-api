package reservas

import (
	"context"
	"errors"
	"time"
)

var (
	ErrReservaNotFound      = errors.New("reserva not found")
	ErrVinculoNotFound      = errors.New("vinculo not found")
	ErrDataObrigatoria      = errors.New("data_viagem is required")
	ErrDataInvalida         = errors.New("data_viagem must be in format YYYY-MM-DD")
	ErrSentidoInvalido      = errors.New("sentido must be ida or volta")
	ErrStatusInvalido       = errors.New("status must be confirmada or cancelada")
	ErrTurnoInvalido        = errors.New("turno must be MT, VT or NT")
	ErrTurnoObrigatorio     = errors.New("turno is required for vinculo integral")
	ErrTurnoIncompativel    = errors.New("turno is incompatible with vinculo")
	ErrVinculoIDObrigatorio = errors.New("vinculo_id is required")
)

type TurnoReserva string
type SentidoReserva string
type StatusReserva string

const (
	TurnoMatutino   TurnoReserva = "MT"
	TurnoVespertino TurnoReserva = "VT"
	TurnoNoturno    TurnoReserva = "NT"
	TurnoIntegral   TurnoReserva = "IN"

	SentidoIda   SentidoReserva = "ida"
	SentidoVolta SentidoReserva = "volta"

	StatusConfirmada StatusReserva = "confirmada"
	StatusCancelada  StatusReserva = "cancelada"
)

type Reserva struct {
	ID            int64
	ClienteID     int64
	VinculoID     int64
	DataViagem    time.Time
	Turno         TurnoReserva
	DestinoID     int64
	RotaInternaID int64
	Cidade        string
	Sentido       SentidoReserva
	Status        StatusReserva
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ReservaInput struct {
	ClienteID     int64
	VinculoID     int64
	DataViagem    time.Time
	Turno         TurnoReserva
	DestinoID     int64
	RotaInternaID int64
	Cidade        string
	Sentido       SentidoReserva
}

type vinculoSnapshot struct {
	ClienteID     int64
	Turno         TurnoReserva
	DestinoID     int64
	RotaInternaID int64
	Cidade        string
}

type ReservaStore interface {
	Create(ctx context.Context, input ReservaInput) (*Reserva, error)
	GetByID(ctx context.Context, reservaID int64) (*Reserva, error)
	GetVinculoSnapshot(ctx context.Context, vinculoID int64) (vinculoSnapshot, error)
	List(ctx context.Context) ([]Reserva, error)
	ListByCliente(ctx context.Context, clienteID int64) ([]Reserva, error)
	ListByVinculo(ctx context.Context, clienteID, vinculoID int64) ([]Reserva, error)
	Update(ctx context.Context, reservaID int64, updateFunc func(*Reserva) (bool, error)) (*Reserva, error)
	Delete(ctx context.Context, reservaID int64) error
}

type ReservaService interface {
	Create(ctx context.Context, input ReservaInput) (*Reserva, error)
	GetByID(ctx context.Context, reservaID int64) (*Reserva, error)
	List(ctx context.Context) ([]Reserva, error)
	ListByCliente(ctx context.Context, clienteID int64) ([]Reserva, error)
	ListByVinculo(ctx context.Context, clienteID, vinculoID int64) ([]Reserva, error)
	Update(ctx context.Context, reservaID int64, updateFunc func(*Reserva) (bool, error)) (*Reserva, error)
	Cancel(ctx context.Context, reservaID int64) (*Reserva, error)
	Delete(ctx context.Context, reservaID int64) error
}
