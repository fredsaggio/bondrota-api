package viagens

import (
	"context"
	"time"
)

type TurnoViagem string
type SentidoViagem string
type StatusCicloViagem string
type StatusViagem string
type StatusPresencaViagem string

const (
	TurnoMatutino   TurnoViagem = "MT"
	TurnoVespertino TurnoViagem = "VT"
	TurnoNoturno    TurnoViagem = "NT"

	SentidoIda   SentidoViagem = "ida"
	SentidoVolta SentidoViagem = "volta"

	StatusCicloPlanejado   StatusCicloViagem = "planejado"
	StatusCicloEmAndamento StatusCicloViagem = "em_andamento"
	StatusCicloConcluido   StatusCicloViagem = "concluido"
	StatusCicloCancelado   StatusCicloViagem = "cancelado"

	StatusViagemProgramada  StatusViagem = "programada"
	StatusViagemEmAndamento StatusViagem = "em_andamento"
	StatusViagemConcluida   StatusViagem = "concluida"
	StatusViagemCancelada   StatusViagem = "cancelada"

	StatusPresencaAguardando StatusPresencaViagem = "aguardando"
	StatusPresencaEmbarcou   StatusPresencaViagem = "embarcou"
	StatusPresencaFaltou     StatusPresencaViagem = "faltou"
	StatusPresencaCancelado  StatusPresencaViagem = "cancelado"
)

type CicloViagem struct {
	ID            int64
	DataViagem    time.Time
	Turno         TurnoViagem
	Cidade        string
	RotaInternaID int64
	VeiculoID     int64
	MotoristaID   int64
	Status        StatusCicloViagem
	ExpiresAt     time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Viagem struct {
	ID              int64
	CicloViagemID   int64
	Sentido         SentidoViagem
	Status          StatusViagem
	PartidaPrevista time.Time
	InicioReal      time.Time
	FimReal         time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ViagemReserva struct {
	ID                 int64
	ViagemID           int64
	ReservaID          int64
	StatusPresenca     StatusPresencaViagem
	HorarioConfirmacao time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CicloViagemInput struct {
	DataViagem    time.Time
	Turno         TurnoViagem
	Cidade        string
	RotaInternaID int64
	VeiculoID     int64
	MotoristaID   int64
	ExpiresAt     time.Time
}

type ViagemInput struct {
	CicloViagemID   int64
	Sentido         SentidoViagem
	PartidaPrevista time.Time
}

type ViagemReservaInput struct {
	ViagemID  int64
	ReservaID int64
}

type CicloComViagens struct {
	Ciclo   CicloViagem
	Viagens []Viagem
}

type ViagemComCiclo struct {
	Viagem Viagem
	Ciclo  CicloViagem
}

type ViagemReservaComReserva struct {
	ViagemReserva
	ClienteID     int64
	VinculoID     int64
	DataViagem    time.Time
	Turno         TurnoViagem
	DestinoID     int64
	RotaInternaID int64
	Cidade        string
	Sentido       SentidoViagem
}

type CicloViagemStore interface {
	CreateCiclo(ctx context.Context, input CicloViagemInput) (*CicloViagem, error)
	GetCicloByID(ctx context.Context, cicloID int64) (*CicloViagem, error)
	ListCiclos(ctx context.Context) ([]CicloViagem, error)
	UpdateCiclo(ctx context.Context, cicloID int64, updateFunc func(*CicloViagem) (bool, error)) (*CicloViagem, error)
}

type ViagemStore interface {
	CreateViagem(ctx context.Context, input ViagemInput) (*Viagem, error)
	GetViagemByID(ctx context.Context, viagemID int64) (*ViagemComCiclo, error)
	ListViagens(ctx context.Context) ([]ViagemComCiclo, error)
	ListViagensByCiclo(ctx context.Context, cicloID int64) ([]Viagem, error)
	UpdateViagem(ctx context.Context, viagemID int64, updateFunc func(*Viagem) (bool, error)) (*Viagem, error)
}

type ViagemReservaStore interface {
	CreateViagemReserva(ctx context.Context, input ViagemReservaInput) (*ViagemReserva, error)
	ListReservasByViagem(ctx context.Context, viagemID int64) ([]ViagemReservaComReserva, error)
	UpdatePresenca(ctx context.Context, viagemID, reservaID int64, updateFunc func(*ViagemReserva) (bool, error)) (*ViagemReserva, error)
}

type PlanejamentoService interface {
	Planejar(ctx context.Context, input CicloViagemInput, partidas map[SentidoViagem]time.Time) (*CicloComViagens, error)
}

type ViagemService interface {
	GetByID(ctx context.Context, viagemID int64) (*ViagemComCiclo, error)
	List(ctx context.Context) ([]ViagemComCiclo, error)
	Iniciar(ctx context.Context, viagemID int64) (*Viagem, error)
	Concluir(ctx context.Context, viagemID int64) (*Viagem, error)
	Cancelar(ctx context.Context, viagemID int64) (*Viagem, error)
}

type PresencaService interface {
	ListReservasByViagem(ctx context.Context, viagemID int64) ([]ViagemReservaComReserva, error)
	AtualizarPresenca(ctx context.Context, viagemID, reservaID int64, status StatusPresencaViagem) (*ViagemReserva, error)
}
