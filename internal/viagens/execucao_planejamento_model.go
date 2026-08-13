package viagens

import (
	"context"
	"errors"
	"time"
)

var (
	ErrExecucaoNaoProcessando   = errors.New("Este planejamento não está em processamento.")
	ErrResultadoInvalido        = errors.New("Resultado de planejamento inválido.")
	ErrUltimoErroObrigatorio    = errors.New("Informe o motivo da falha.")
	ErrProximaTentativaInvalida = errors.New("A próxima tentativa deve ser posterior à falha.")
)

type StatusExecucaoPlanejamento string

const (
	StatusExecucaoProcessando StatusExecucaoPlanejamento = "processando"
	StatusExecucaoConcluido   StatusExecucaoPlanejamento = "concluido"
	StatusExecucaoSemDemanda  StatusExecucaoPlanejamento = "sem_demanda"
	StatusExecucaoFalhou      StatusExecucaoPlanejamento = "falhou"
)

type ChaveExecucaoPlanejamento struct {
	DataViagem         time.Time
	Turno              TurnoViagem
	MunicipioDestinoID int64
	RotaInternaID      int64
	Sentido            SentidoViagem
}

type ExecucaoPlanejamento struct {
	ID                 int64
	DataViagem         time.Time
	Turno              TurnoViagem
	MunicipioDestinoID int64
	RotaInternaID      int64
	Sentido            SentidoViagem
	PartidaEm          time.Time
	FechamentoEm       time.Time
	Status             StatusExecucaoPlanejamento
	Tentativas         int
	UltimoErro         *string
	BloqueioExpiraEm   *time.Time
	ProximaTentativaEm *time.Time
	IniciadoEm         time.Time
	FinalizadoEm       *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type FalharExecucaoPlanejamentoInput struct {
	ExecucaoID         int64
	Mensagem           string
	FalhouEm           time.Time
	ProximaTentativaEm time.Time
}

type IniciarExecucaoPlanejamentoInput struct {
	Chave            ChaveExecucaoPlanejamento
	PartidaEm        time.Time
	FechamentoEm     time.Time
	Agora            time.Time
	BloqueioExpiraEm time.Time
}

type ExecucaoPlanejamentoStore interface {
	TentarIniciar(ctx context.Context, input IniciarExecucaoPlanejamentoInput) (*ExecucaoPlanejamento, bool, error)
	GetByChave(ctx context.Context, chave ChaveExecucaoPlanejamento) (*ExecucaoPlanejamento, error)
	Finalizar(ctx context.Context, execucaoID int64, resultado StatusExecucaoPlanejamento) (*ExecucaoPlanejamento, error)
	Falhar(ctx context.Context, input FalharExecucaoPlanejamentoInput) (*ExecucaoPlanejamento, error)
}

type ExecucaoPlanejamentoFalhaStore interface {
	ListFalhas(ctx context.Context, limit int) ([]ExecucaoPlanejamento, error)
}

type ExecucaoPlanejamentoRepository interface {
	ExecucaoPlanejamentoStore
	ExecucaoPlanejamentoFalhaStore
}
