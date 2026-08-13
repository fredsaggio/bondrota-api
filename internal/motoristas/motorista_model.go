package motoristas

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("Motorista não encontrado.")
	ErrNomeObrigatorio  = errors.New("Informe o nome.")
	ErrTurnoInvalido    = errors.New("turno must be MT, VT, NT or IN")
	ErrDataNascInvalida = errors.New("data_nasc must be in format YYYY-MM-DD")
)

type Turno string

const (
	TurnoMatutino   Turno = "MT"
	TurnoVespertino Turno = "VT"
	TurnoNoturno    Turno = "NT"
	TurnoIntegral   Turno = "IN"
)

type Motorista struct {
	ID                  int64
	Nome                string
	CPF                 string
	Senha               string
	Telefone            string
	DataNasc            time.Time
	Turno               Turno
	MunicipioTrabalhoID int64
	Residencia          string
	Foto                string
}

type MotoristaInput struct {
	Nome                string
	CPF                 string
	Senha               string
	Telefone            string
	DataNasc            time.Time
	Turno               Turno
	MunicipioTrabalhoID int64
	Residencia          string
	Foto                string
}

type MotoristasDisponiveisFiltro struct {
	MunicipioTrabalhoID int64
	DataViagem          time.Time
	Turno               Turno
	Limit               int
}

type AlocarMotoristasInput struct {
	MunicipioTrabalhoID int64
	DataViagem          time.Time
	Turno               Turno
	Quantidade          int
}

type MotoristaStore interface {
	Create(ctx context.Context, input MotoristaInput) (*Motorista, error)
	GetByID(ctx context.Context, motoristaID int64) (*Motorista, error)
	GetByCPF(ctx context.Context, cpf string) (*Motorista, error)
	List(ctx context.Context) ([]Motorista, error)
	Update(ctx context.Context, motoristaID int64, updateFunc func(*Motorista) (bool, error)) (*Motorista, error)
	Delete(ctx context.Context, motoristaID int64) error
}

type AlocacaoMotoristaStore interface {
	ListDisponiveisParaAlocacao(ctx context.Context, filtro MotoristasDisponiveisFiltro) ([]Motorista, error)
}

type MotoristaService interface {
	Login(ctx context.Context, cpf, senha string) (string, error)
	Create(ctx context.Context, input MotoristaInput) (*Motorista, error)
	GetByID(ctx context.Context, motoristaID int64) (*Motorista, error)
	List(ctx context.Context) ([]Motorista, error)
	Update(ctx context.Context, motoristaID int64, updateFunc func(*Motorista) (bool, error)) (*Motorista, error)
	Delete(ctx context.Context, motoristaID int64) error
}
