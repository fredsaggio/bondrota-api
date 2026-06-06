package viagens

import (
	"context"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
)

type TurnoViagem string
type SentidoViagem string
type StatusCicloViagem string
type StatusViagem string
type StatusPresencaViagem string
type TipoHorarioViagem string

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

	TipoHorarioPartidaPrevista TipoHorarioViagem = "partida_prevista"
	TipoHorarioInicioReal      TipoHorarioViagem = "inicio_real"
	TipoHorarioFimReal         TipoHorarioViagem = "fim_real"
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
	ID            int64
	CicloViagemID int64
	Sentido       SentidoViagem
	Status        StatusViagem
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ViagemHorario struct {
	ID        int64
	ViagemID  int64
	Tipo      TipoHorarioViagem
	Horario   time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ViagemReserva struct {
	ID             int64
	ViagemID       int64
	ReservaID      int64
	StatusPresenca StatusPresencaViagem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ViagemReservaConfirmacao struct {
	ViagemReservaID  int64
	RegistroPresenca time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ViagemLocalizacao struct {
	ViagemID       int64
	MotoristaID    int64
	Latitude       float64
	Longitude      float64
	VelocidadeKmh  float64
	DirecaoGraus   float64
	PrecisaoMetros float64
	RegistradaEm   time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ViagemLocalizacaoInput struct {
	ViagemID       int64
	MotoristaID    int64
	Latitude       float64
	Longitude      float64
	VelocidadeKmh  float64
	DirecaoGraus   float64
	PrecisaoMetros float64
	RegistradaEm   time.Time
}

type ViagemLocalizacaoActor struct {
	UserID int64
	Role   string
}

type HorarioTurnoViagem struct {
	ID           int64
	Cidade       string
	Turno        TurnoViagem
	HorarioIda   time.Duration
	HorarioVolta time.Duration
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type HorarioTurnoViagemInput struct {
	Cidade       string
	Turno        TurnoViagem
	HorarioIda   time.Duration
	HorarioVolta time.Duration
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

type PlanejamentoViagensInput struct {
	DataViagem    time.Time
	Turno         TurnoViagem
	Cidade        string
	RotaInternaID int64
	ExpiresAt     time.Time
}

type PlanejamentoReservasFiltro struct {
	DataViagem    time.Time
	Turno         TurnoViagem
	Cidade        string
	RotaInternaID int64
	Sentido       SentidoViagem
}

type PlanejamentoReserva struct {
	ID        int64
	DestinoID int64
}

type CicloViagemComReservasInput struct {
	Ciclo           CicloViagemInput
	ReservaIDsIda   []int64
	ReservaIDsVolta []int64
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

type PlanejamentoViagens struct {
	Ciclos                  []CicloComViagens
	QuantidadeReservasIda   int
	QuantidadeReservasVolta int
	CapacidadeTotal         int
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
	CreateCicloComViagens(ctx context.Context, input CicloViagemInput, partidas map[SentidoViagem]time.Time) (*CicloComViagens, error)
	CreateCiclosComViagens(ctx context.Context, inputs []CicloViagemComReservasInput, partidas map[SentidoViagem]time.Time) (*PlanejamentoViagens, error)
	ListReservasConfirmadasParaPlanejamento(ctx context.Context, filtro PlanejamentoReservasFiltro) ([]PlanejamentoReserva, error)
	GetCicloByID(ctx context.Context, cicloID int64) (*CicloViagem, error)
	ListCiclos(ctx context.Context) ([]CicloViagem, error)
	UpdateCiclo(ctx context.Context, cicloID int64, updateFunc func(*CicloViagem) (bool, error)) (*CicloViagem, error)
}

type HorarioTurnoViagemStore interface {
	Create(ctx context.Context, input HorarioTurnoViagemInput) (*HorarioTurnoViagem, error)
	GetByID(ctx context.Context, id int64) (*HorarioTurnoViagem, error)
	GetByCidadeTurno(ctx context.Context, cidade string, turno TurnoViagem) (*HorarioTurnoViagem, error)
	List(ctx context.Context) ([]HorarioTurnoViagem, error)
	Update(ctx context.Context, id int64, updateFunc func(*HorarioTurnoViagem) (bool, error)) (*HorarioTurnoViagem, error)
	Delete(ctx context.Context, id int64) error
}

type ViagemStore interface {
	CreateViagem(ctx context.Context, input ViagemInput) (*Viagem, error)
	GetViagemByID(ctx context.Context, viagemID int64) (*ViagemComCiclo, error)
	ListViagens(ctx context.Context) ([]ViagemComCiclo, error)
	ListViagensByCiclo(ctx context.Context, cicloID int64) ([]Viagem, error)
	ListHorariosByViagem(ctx context.Context, viagemID int64) ([]ViagemHorario, error)
	RegistrarHorarioViagem(ctx context.Context, viagemID int64, tipo TipoHorarioViagem, horario time.Time) (*ViagemHorario, error)
	AtualizarStatusERegistrarHorarioViagem(ctx context.Context, viagemID int64, from StatusViagem, to StatusViagem, tipo TipoHorarioViagem, horario time.Time) (*Viagem, error)
	UpdateViagem(ctx context.Context, viagemID int64, updateFunc func(*Viagem) (bool, error)) (*Viagem, error)
}

type ViagemReservaStore interface {
	CreateViagemReserva(ctx context.Context, input ViagemReservaInput) (*ViagemReserva, error)
	ListReservasByViagem(ctx context.Context, viagemID int64) ([]ViagemReservaComReserva, error)
	UpdatePresenca(ctx context.Context, viagemID, reservaID int64, updateFunc func(*ViagemReserva) (bool, error)) (*ViagemReserva, error)
}

type ViagemLocalizacaoStore interface {
	Upsert(ctx context.Context, input ViagemLocalizacaoInput) (*ViagemLocalizacao, error)
	GetByViagem(ctx context.Context, viagemID int64) (*ViagemLocalizacao, error)
	CanMotoristaAccessViagem(ctx context.Context, viagemID, motoristaID int64, requireEmAndamento bool) (bool, error)
	CanClienteAccessViagem(ctx context.Context, viagemID, clienteID int64) (bool, error)
}

type PlanejamentoService interface {
	Planejar(ctx context.Context, input PlanejamentoViagensInput) (*PlanejamentoViagens, error)
}

type HorarioTurnoViagemService interface {
	Create(ctx context.Context, input HorarioTurnoViagemInput) (*HorarioTurnoViagem, error)
	GetByID(ctx context.Context, id int64) (*HorarioTurnoViagem, error)
	List(ctx context.Context) ([]HorarioTurnoViagem, error)
	Update(ctx context.Context, id int64, updateFunc func(*HorarioTurnoViagem) (bool, error)) (*HorarioTurnoViagem, error)
	Delete(ctx context.Context, id int64) error
}

type VeiculoAlocador interface {
	Alocar(ctx context.Context, input veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error)
}

type MotoristaAlocador interface {
	Alocar(ctx context.Context, input motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error)
}

type ViagemService interface {
	GetByID(ctx context.Context, viagemID int64) (*ViagemComCiclo, error)
	List(ctx context.Context) ([]ViagemComCiclo, error)
	ListHorariosByViagem(ctx context.Context, viagemID int64) ([]ViagemHorario, error)
	Iniciar(ctx context.Context, viagemID int64) (*Viagem, error)
	Concluir(ctx context.Context, viagemID int64) (*Viagem, error)
	Cancelar(ctx context.Context, viagemID int64) (*Viagem, error)
}

type PresencaService interface {
	ListReservasByViagem(ctx context.Context, viagemID int64) ([]ViagemReservaComReserva, error)
	AtualizarPresenca(ctx context.Context, viagemID, reservaID int64, status StatusPresencaViagem) (*ViagemReserva, error)
}

type ViagemLocalizacaoService interface {
	Atualizar(ctx context.Context, actor ViagemLocalizacaoActor, input ViagemLocalizacaoInput) (*ViagemLocalizacao, error)
	GetByViagem(ctx context.Context, actor ViagemLocalizacaoActor, viagemID int64) (*ViagemLocalizacao, error)
}
