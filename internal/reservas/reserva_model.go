package reservas

import (
	"context"
	"errors"
	"time"
)

var (
	ErrReservaNotFound       = errors.New("reserva not found")
	ErrVinculoNotFound       = errors.New("vinculo not found")
	ErrDataObrigatoria       = errors.New("data_viagem is required")
	ErrDataInvalida          = errors.New("data_viagem must be in format YYYY-MM-DD")
	ErrSentidoInvalido       = errors.New("sentido must be ida or volta")
	ErrStatusInvalido        = errors.New("status must be confirmada or cancelada")
	ErrTurnoInvalido         = errors.New("turno must be MT, VT or NT")
	ErrTurnoObrigatorio      = errors.New("turno is required for vinculo integral")
	ErrTurnoIncompativel     = errors.New("turno is incompatible with vinculo")
	ErrVinculoIDObrigatorio  = errors.New("vinculo_id is required")
	ErrHorarioNaoConfigurado = errors.New("trip schedule is not configured for this destination and turno")
	ErrPrazoReservaEncerrado = errors.New("reservation deadline has passed")
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
	Sentido       SentidoReserva
}

type DisponibilidadeReservaInput struct {
	ClienteID  int64
	VinculoID  int64
	DataViagem time.Time
	Turno      TurnoReserva
	Sentido    SentidoReserva
}

type DisponibilidadeReserva struct {
	DataViagem   time.Time
	Turno        TurnoReserva
	Sentido      SentidoReserva
	PartidaEm    time.Time
	FechamentoEm time.Time
	ConsultadoEm time.Time
	Disponivel   bool
}

// ReservaComNomes carrega os nomes do cliente e do destino resolvidos via JOIN,
// para a listagem administrativa nao obrigar o consumidor a buscar cada cliente e
// destino separadamente (mesmo padrao de VinculoComCliente).
type ReservaComNomes struct {
	Reserva
	ClienteNome string
	DestinoNome string
}

// ReservaCursor identifica a ultima linha vista pela pagina anterior, no mesmo
// criterio do ORDER BY (data_viagem DESC, id DESC), para retomar dali sem OFFSET.
type ReservaCursor struct {
	DataViagem time.Time
	ID         int64
}

type ReservaListParams struct {
	Cursor     *ReservaCursor
	Limit      int
	Busca      string
	DataInicio *time.Time
	DataFim    *time.Time
}

type ReservaListResult struct {
	Items      []ReservaComNomes
	NextCursor *ReservaCursor
	HasMore    bool
}

type ReservaServiceConfig struct {
	Location               *time.Location
	Now                    func() time.Time
	AntecedenciaFechamento time.Duration
}

type VinculoSnapshot struct {
	ClienteID     int64
	Turno         TurnoReserva
	DestinoID     int64
	RotaInternaID int64
}

type ReservaStore interface {
	Create(ctx context.Context, input ReservaInput) (*Reserva, error)
	GetByID(ctx context.Context, reservaID int64) (*Reserva, error)
	GetVinculoSnapshot(ctx context.Context, vinculoID int64) (VinculoSnapshot, error)
	GetHorarioPartida(ctx context.Context, destinoID int64, turno TurnoReserva, sentido SentidoReserva) (time.Duration, error)
	List(ctx context.Context, params ReservaListParams) (ReservaListResult, error)
	ListByCliente(ctx context.Context, clienteID int64) ([]Reserva, error)
	ListByVinculo(ctx context.Context, clienteID, vinculoID int64) ([]Reserva, error)
	Update(ctx context.Context, reservaID int64, updateFunc func(*Reserva) (bool, error)) (*Reserva, error)
	Delete(ctx context.Context, reservaID int64) error
}

type RotaDinamicaInvalidator interface {
	InvalidarPorReserva(ctx context.Context, reservaID int64) error
}

type ReservaService interface {
	Create(ctx context.Context, input ReservaInput) (*Reserva, error)
	ConsultarDisponibilidade(ctx context.Context, input DisponibilidadeReservaInput) (*DisponibilidadeReserva, error)
	GetByID(ctx context.Context, reservaID int64) (*Reserva, error)
	List(ctx context.Context, params ReservaListParams) (ReservaListResult, error)
	ListByCliente(ctx context.Context, clienteID int64) ([]Reserva, error)
	ListByVinculo(ctx context.Context, clienteID, vinculoID int64) ([]Reserva, error)
	Update(ctx context.Context, reservaID int64, updateFunc func(*Reserva) (bool, error)) (*Reserva, error)
	Cancel(ctx context.Context, reservaID int64) (*Reserva, error)
	Delete(ctx context.Context, reservaID int64) error
}
