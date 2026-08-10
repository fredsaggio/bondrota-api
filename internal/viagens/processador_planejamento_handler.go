package viagens

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type ProcessadorPlanejamentoHandler struct {
	processador ProcessadorPlanejamentoService
}

func NewProcessadorPlanejamentoHandler(processador ProcessadorPlanejamentoService) *ProcessadorPlanejamentoHandler {
	return &ProcessadorPlanejamentoHandler{processador: processador}
}

type ResumoProcessamentoPlanejamentoResponse struct {
	Candidatos int `json:"candidatos"`
	Devidos    int `json:"devidos"`
	Adquiridos int `json:"adquiridos"`
	Concluidos int `json:"concluidos"`
	SemDemanda int `json:"sem_demanda"`
	Falhos     int `json:"falhos"`
}

func (h *ProcessadorPlanejamentoHandler) Processar(w http.ResponseWriter, r *http.Request) {
	inicio := time.Now()
	resumo, err := h.processador.Processar(r.Context())
	if err != nil {
		logResumoProcessamentoPlanejamento(r.Context(), slog.LevelError, resumo, time.Since(inicio), err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if resumo.Devidos > 0 {
		logResumoProcessamentoPlanejamento(r.Context(), slog.LevelInfo, resumo, time.Since(inicio), nil)
	}

	httputils.Respond(w, http.StatusOK, toResumoProcessamentoPlanejamentoResponse(resumo))
}

func logResumoProcessamentoPlanejamento(ctx context.Context, level slog.Level, resumo ResumoProcessamentoPlanejamento, duracao time.Duration, err error) {
	attrs := []any{
		"candidatos", resumo.Candidatos,
		"devidos", resumo.Devidos,
		"adquiridos", resumo.Adquiridos,
		"concluidos", resumo.Concluidos,
		"sem_demanda", resumo.SemDemanda,
		"falhos", resumo.Falhos,
		"duracao_ms", duracao.Milliseconds(),
	}
	if err != nil {
		attrs = append(attrs, "error", err)
	}
	slog.Log(ctx, level, "scheduled planning processing completed", attrs...)
}

func toResumoProcessamentoPlanejamentoResponse(resumo ResumoProcessamentoPlanejamento) ResumoProcessamentoPlanejamentoResponse {
	return ResumoProcessamentoPlanejamentoResponse{
		Candidatos: resumo.Candidatos,
		Devidos:    resumo.Devidos,
		Adquiridos: resumo.Adquiridos,
		Concluidos: resumo.Concluidos,
		SemDemanda: resumo.SemDemanda,
		Falhos:     resumo.Falhos,
	}
}
