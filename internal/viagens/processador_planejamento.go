package viagens

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

const (
	DefaultAntecedenciaFechamentoPlanejamento = 30 * time.Minute
	DefaultDuracaoBloqueioPlanejamento        = 5 * time.Minute
)

type ProcessadorPlanejamento struct {
	agendador              AgendadorPlanejamentoStore
	execucoes              ExecucaoPlanejamentoStore
	planejador             PlanejamentoService
	location               *time.Location
	now                    func() time.Time
	antecedenciaFechamento time.Duration
	duracaoBloqueio        time.Duration
}

func NewProcessadorPlanejamento(
	agendador AgendadorPlanejamentoStore,
	execucoes ExecucaoPlanejamentoStore,
	planejador PlanejamentoService,
	config ProcessadorPlanejamentoConfig,
) *ProcessadorPlanejamento {
	if config.Location == nil {
		config.Location = time.UTC
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.AntecedenciaFechamento <= 0 {
		config.AntecedenciaFechamento = DefaultAntecedenciaFechamentoPlanejamento
	}
	if config.DuracaoBloqueio <= 0 {
		config.DuracaoBloqueio = DefaultDuracaoBloqueioPlanejamento
	}

	return &ProcessadorPlanejamento{
		agendador:              agendador,
		execucoes:              execucoes,
		planejador:             planejador,
		location:               config.Location,
		now:                    config.Now,
		antecedenciaFechamento: config.AntecedenciaFechamento,
		duracaoBloqueio:        config.DuracaoBloqueio,
	}
}

func (p *ProcessadorPlanejamento) Processar(ctx context.Context) (ResumoProcessamentoPlanejamento, error) {
	agora := p.now().In(p.location)
	dataInicio := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, p.location)
	dataFim := dataInicio.AddDate(0, 0, 1)

	candidatos, err := p.agendador.ListCandidatos(ctx, dataInicio, dataFim)
	if err != nil {
		return ResumoProcessamentoPlanejamento{}, err
	}

	resumo := ResumoProcessamentoPlanejamento{Candidatos: len(candidatos)}
	errosProcessamento := make([]error, 0)

	for _, candidato := range candidatos {
		if err := ctx.Err(); err != nil {
			errosProcessamento = append(errosProcessamento, err)
			break
		}

		partidaEm := combinarDataHorario(candidato.Chave.DataViagem, candidato.HorarioPartida, p.location)
		fechamentoEm := partidaEm.Add(-p.antecedenciaFechamento)
		if agora.Before(fechamentoEm) || !agora.Before(partidaEm) {
			continue
		}

		resumo.Devidos++
		if err := p.processarCandidato(ctx, candidato, partidaEm, fechamentoEm, agora, &resumo); err != nil {
			errosProcessamento = append(errosProcessamento, err)
		}
	}

	return resumo, errors.Join(errosProcessamento...)
}

func (p *ProcessadorPlanejamento) processarCandidato(
	ctx context.Context,
	candidato CandidatoPlanejamento,
	partidaEm time.Time,
	fechamentoEm time.Time,
	agora time.Time,
	resumo *ResumoProcessamentoPlanejamento,
) error {
	execucao, adquirida, err := p.execucoes.TentarIniciar(ctx, IniciarExecucaoPlanejamentoInput{
		Chave:            candidato.Chave,
		PartidaEm:        partidaEm,
		FechamentoEm:     fechamentoEm,
		Agora:            agora,
		BloqueioExpiraEm: agora.Add(p.duracaoBloqueio),
	})
	if err != nil {
		resumo.Falhos++
		return fmt.Errorf("claim planejamento %s: %w", formatarChavePlanejamento(candidato.Chave), err)
	}
	if !adquirida {
		return nil
	}
	resumo.Adquiridos++

	_, err = p.planejador.Planejar(ctx, PlanejamentoViagensInput{
		DataViagem:         candidato.Chave.DataViagem,
		Turno:              candidato.Chave.Turno,
		MunicipioDestinoID: candidato.Chave.MunicipioDestinoID,
		RotaInternaID:      candidato.Chave.RotaInternaID,
		Sentido:            candidato.Chave.Sentido,
	})
	switch {
	case err == nil, errors.Is(err, brerror.ErrAlreadyExists):
		if _, finalizarErr := p.execucoes.Finalizar(ctx, execucao.ID, StatusExecucaoConcluido); finalizarErr != nil {
			resumo.Falhos++
			return fmt.Errorf("finish planejamento %s: %w", formatarChavePlanejamento(candidato.Chave), finalizarErr)
		}
		resumo.Concluidos++
		return nil

	case errors.Is(err, ErrSemDemandaPlanejamento):
		if _, finalizarErr := p.execucoes.Finalizar(ctx, execucao.ID, StatusExecucaoSemDemanda); finalizarErr != nil {
			resumo.Falhos++
			return fmt.Errorf("finish planejamento without demand %s: %w", formatarChavePlanejamento(candidato.Chave), finalizarErr)
		}
		resumo.SemDemanda++
		return nil

	default:
		resumo.Falhos++
		_, falharErr := p.execucoes.Falhar(ctx, execucao.ID, err.Error())
		if falharErr != nil {
			return errors.Join(
				fmt.Errorf("process planejamento %s: %w", formatarChavePlanejamento(candidato.Chave), err),
				fmt.Errorf("mark planejamento %s as failed: %w", formatarChavePlanejamento(candidato.Chave), falharErr),
			)
		}
		return fmt.Errorf("process planejamento %s: %w", formatarChavePlanejamento(candidato.Chave), err)
	}
}

func formatarChavePlanejamento(chave ChaveExecucaoPlanejamento) string {
	return fmt.Sprintf(
		"%s/%s/%d/%d/%s",
		chave.DataViagem.Format("2006-01-02"),
		chave.Turno,
		chave.MunicipioDestinoID,
		chave.RotaInternaID,
		chave.Sentido,
	)
}
