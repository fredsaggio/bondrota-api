package viagens

import (
	"context"
	"errors"
	"time"
)

var (
	ErrExecucaoNaoProcessando = errors.New("execucao planejamento is not processing")
	ErrResultadoInvalido      = errors.New("resultado execucao planejamento must be concluido or sem_demanda")
	ErrUltimoErroObrigatorio  = errors.New("ultimo erro is required")
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
	IniciadoEm         time.Time
	FinalizadoEm       *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
	Falhar(ctx context.Context, execucaoID int64, mensagem string) (*ExecucaoPlanejamento, error)
}
