package viagens

import (
	"log/slog"
	"net/http"

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
	resumo, err := h.processador.Processar(r.Context())
	if err != nil {
		slog.Error("failed to process scheduled planejamentos", "error", err, "summary", resumo)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toResumoProcessamentoPlanejamentoResponse(resumo))
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
