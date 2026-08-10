package viagens

import (
	"context"
	"errors"
	"time"
)

var ErrSemDemandaPlanejamento = errors.New("no demand for planejamento")

type CandidatoPlanejamento struct {
	Chave          ChaveExecucaoPlanejamento
	HorarioPartida time.Duration
}

type AgendadorPlanejamentoStore interface {
	ListCandidatos(ctx context.Context, dataInicio, dataFim time.Time) ([]CandidatoPlanejamento, error)
}

type ProcessadorPlanejamentoConfig struct {
	Location               *time.Location
	Now                    func() time.Time
	AntecedenciaFechamento time.Duration
	DuracaoBloqueio        time.Duration
	IntervaloRetryInicial  time.Duration
	IntervaloRetryMaximo   time.Duration
}

type ResumoProcessamentoPlanejamento struct {
	Candidatos int
	Devidos    int
	Adquiridos int
	Concluidos int
	SemDemanda int
	Falhos     int
}

type ProcessadorPlanejamentoService interface {
	Processar(ctx context.Context) (ResumoProcessamentoPlanejamento, error)
}
