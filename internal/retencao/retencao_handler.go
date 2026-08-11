package retencao

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

type ResumoLimpezaResponse struct {
	Corte              string `json:"corte"`
	CiclosRemovidos    int64  `json:"ciclos_removidos"`
	ReservasRemovidas  int64  `json:"reservas_removidas"`
	ExecucoesRemovidas int64  `json:"execucoes_removidas"`
	LoteSaturado       bool   `json:"lote_saturado"`
}

func (h *Handler) Limpar(w http.ResponseWriter, r *http.Request) {
	inicio := time.Now()
	resumo, err := h.svc.Limpar(r.Context())
	duracao := time.Since(inicio)

	if err != nil {
		slog.Error("scheduled retention cleanup failed",
			"error", err,
			"corte", resumo.Corte.Format(time.RFC3339),
			"ciclos_removidos", resumo.CiclosRemovidos,
			"reservas_removidas", resumo.ReservasRemovidas,
			"execucoes_removidas", resumo.ExecucoesRemovidas,
			"duracao_ms", duracao.Milliseconds(),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Execucoes sem nada a remover sao a maioria; so registra quando houve trabalho
	// ou quando o lote saturou e sobrou coisa para a proxima rodada.
	if total := resumo.CiclosRemovidos + resumo.ReservasRemovidas + resumo.ExecucoesRemovidas; total > 0 || resumo.LoteSaturado {
		slog.Info("scheduled retention cleanup completed",
			"corte", resumo.Corte.Format(time.RFC3339),
			"ciclos_removidos", resumo.CiclosRemovidos,
			"reservas_removidas", resumo.ReservasRemovidas,
			"execucoes_removidas", resumo.ExecucoesRemovidas,
			"lote_saturado", resumo.LoteSaturado,
			"duracao_ms", duracao.Milliseconds(),
		)
	}

	httputils.Respond(w, http.StatusOK, ResumoLimpezaResponse{
		Corte:              resumo.Corte.Format(time.RFC3339),
		CiclosRemovidos:    resumo.CiclosRemovidos,
		ReservasRemovidas:  resumo.ReservasRemovidas,
		ExecucoesRemovidas: resumo.ExecucoesRemovidas,
		LoteSaturado:       resumo.LoteSaturado,
	})
}
