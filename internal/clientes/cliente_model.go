package clientes

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("cliente not found")
	ErrVinculoNotFound  = errors.New("vinculo not found")
	ErrNomeObrigatorio  = errors.New("nome is required")
	ErrCursoObrigatorio = errors.New("curso is required for estudante")
	ErrTipoInvalido     = errors.New("tipo must be estudante or estagio")
	ErrTurnoInvalido    = errors.New("turno must be MT, VT, NT or IN")
	ErrDiaInvalido      = errors.New("dia_semana must be between 1 and 5")
	ErrDiaDuplicado     = errors.New("dia_semana must not be duplicated")
	ErrDataInvalida     = errors.New("date must be in format YYYY-MM-DD")
)

type TipoConta string
type TurnoCliente string
type DiaSemana int8

const (
	TipoEstudante TipoConta = "estudante"
	TipoEstagio   TipoConta = "estagio"

	TurnoMatutino   TurnoCliente = "MT"
	TurnoVespertino TurnoCliente = "VT"
	TurnoNoturno    TurnoCliente = "NT"
	TurnoIntegral   TurnoCliente = "IN"
)

const (
	Segunda DiaSemana = iota + 1
	Terca
	Quarta
	Quinta
	Sexta
)

type Cliente struct {
	ID       int64
	Nome     string
	CPF      string
	Senha    string
	Telefone string
	DataNasc time.Time
	Foto     string
}

type ClienteComVinculos struct {
	Cliente
	Vinculos []Vinculo
}

// ClienteListParams pagina por id (a listagem ordena por id DESC). Diferente de
// reservas e viagens, cliente nao tem data de referencia para recortar.
type ClienteListParams struct {
	// CursorID > 0 retoma a partir da ultima linha vista.
	CursorID int64
	Limit    int
	// Busca casa por nome, CPF ou telefone.
	Busca string
}

type ClienteListResult struct {
	Items        []Cliente
	NextCursorID int64
	HasMore      bool
}

// ClienteResumo evita que o painel baixe a tabela so para contar cadastros.
type ClienteResumo struct {
	Total int64
}

type ClienteInput struct {
	Nome     string
	CPF      string
	Senha    string
	Telefone string
	DataNasc time.Time
	Foto     string
}

type Vinculo struct {
	ID            int64
	ClienteID     int64
	Tipo          TipoConta
	Turno         TurnoCliente
	DestinoID     int64
	RotaInternaID int64
	Curso         string
	Comprovante   string
	Validade      time.Time
	HorariosFixos []HorarioFixo
}

// VinculoComCliente carrega o nome do cliente dono do vinculo. Ele existe para a
// listagem administrativa, que precisa identificar o passageiro sem obrigar o
// consumidor a buscar cada cliente separadamente.
type VinculoComCliente struct {
	Vinculo
	ClienteNome string
	DestinoNome string
}

// VinculoCursor guarda o par que ordena a listagem (nome do cliente, id). O nome
// sozinho nao serve de cursor por nao ser unico; o id desempata.
type VinculoCursor struct {
	ClienteNome string
	ID          int64
}

type VinculoListParams struct {
	Cursor *VinculoCursor
	Limit  int
	// Busca casa por nome do cliente, nome do destino, curso, tipo e turno.
	Busca string
}

type VinculoListResult struct {
	Items      []VinculoComCliente
	NextCursor *VinculoCursor
	HasMore    bool
}

type VinculoInput struct {
	ClienteID     int64
	Tipo          TipoConta
	Turno         TurnoCliente
	DestinoID     int64
	RotaInternaID int64
	Curso         string
	Comprovante   string
	Validade      time.Time
	HorariosFixos []DiaSemana
}

type VinculoUpdateInput struct {
	Tipo          TipoConta
	Turno         TurnoCliente
	DestinoID     int64
	RotaInternaID int64
	Curso         string
	Comprovante   string
	Validade      time.Time
	HorariosFixos []DiaSemana
}

type HorarioFixo struct {
	ID        int64
	VinculoID int64
	DiaSemana DiaSemana
}

type ClienteStore interface {
	Create(ctx context.Context, input ClienteInput) (*Cliente, error)
	GetByID(ctx context.Context, clienteID int64) (*ClienteComVinculos, error)
	GetByCPF(ctx context.Context, cpf string) (*Cliente, error)
	List(ctx context.Context, params ClienteListParams) (ClienteListResult, error)
	Resumo(ctx context.Context) (ClienteResumo, error)
	Update(ctx context.Context, clienteID int64, updateFunc func(*Cliente) (bool, error)) (*Cliente, error)
	Delete(ctx context.Context, clienteID int64) error
}

type VinculoStore interface {
	Create(ctx context.Context, input VinculoInput) (*Vinculo, error)
	GetByID(ctx context.Context, vinculoID int64) (*Vinculo, error)
	List(ctx context.Context, params VinculoListParams) (VinculoListResult, error)
	ListByCliente(ctx context.Context, clienteID int64) ([]Vinculo, error)
	Update(ctx context.Context, vinculoID int64, input VinculoUpdateInput) (*Vinculo, error)
	Delete(ctx context.Context, vinculoID int64) error
}

type ClienteService interface {
	Login(ctx context.Context, cpf, senha string) (string, error)
	Create(ctx context.Context, input ClienteInput) (*Cliente, error)
	GetByID(ctx context.Context, clienteID int64) (*ClienteComVinculos, error)
	List(ctx context.Context, params ClienteListParams) (ClienteListResult, error)
	Resumo(ctx context.Context) (ClienteResumo, error)
	Update(ctx context.Context, clienteID int64, updateFunc func(*Cliente) (bool, error)) (*Cliente, error)
	Delete(ctx context.Context, clienteID int64) error
}

type VinculoService interface {
	Create(ctx context.Context, input VinculoInput) (*Vinculo, error)
	GetByID(ctx context.Context, vinculoID int64) (*Vinculo, error)
	List(ctx context.Context, params VinculoListParams) (VinculoListResult, error)
	ListByCliente(ctx context.Context, clienteID int64) ([]Vinculo, error)
	Update(ctx context.Context, vinculoID int64, input VinculoUpdateInput) (*Vinculo, error)
	Delete(ctx context.Context, vinculoID int64) error
}
