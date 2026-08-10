package viagens

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

const (
	defaultLimiteFalhasPlanejamento = 50
	maxLimiteFalhasPlanejamento     = 100
)

type ExecucaoPlanejamentoHandler struct {
	store ExecucaoPlanejamentoFalhaStore
}

func NewExecucaoPlanejamentoHandler(store ExecucaoPlanejamentoFalhaStore) *ExecucaoPlanejamentoHandler {
	return &ExecucaoPlanejamentoHandler{store: store}
}

type ExecucaoPlanejamentoFalhaResponse struct {
	ID                 int64                      `json:"id"`
	DataViagem         string                     `json:"data_viagem"`
	Turno              TurnoViagem                `json:"turno"`
	MunicipioDestinoID int64                      `json:"municipio_destino_id"`
	RotaInternaID      int64                      `json:"rota_interna_id"`
	Sentido            SentidoViagem              `json:"sentido"`
	PartidaEm          string                     `json:"partida_em"`
	FechamentoEm       string                     `json:"fechamento_em"`
	Status             StatusExecucaoPlanejamento `json:"status"`
	Tentativas         int                        `json:"tentativas"`
	UltimoErro         *string                    `json:"ultimo_erro"`
	ProximaTentativaEm *string                    `json:"proxima_tentativa_em"`
	FinalizadoEm       *string                    `json:"finalizado_em"`
}

func (h *ExecucaoPlanejamentoHandler) ListFalhas(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimiteFalhasPlanejamento(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, "limit must be an integer between 1 and 100", http.StatusBadRequest)
		return
	}

	execucoes, err := h.store.ListFalhas(r.Context(), limit)
	if err != nil {
		slog.Error("failed to list planejamento failures", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]ExecucaoPlanejamentoFalhaResponse, 0, len(execucoes))
	for _, execucao := range execucoes {
		response = append(response, toExecucaoPlanejamentoFalhaResponse(execucao))
	}
	httputils.Respond(w, http.StatusOK, response)
}

func parseLimiteFalhasPlanejamento(value string) (int, error) {
	if value == "" {
		return defaultLimiteFalhasPlanejamento, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > maxLimiteFalhasPlanejamento {
		return 0, strconv.ErrSyntax
	}
	return limit, nil
}

func toExecucaoPlanejamentoFalhaResponse(execucao ExecucaoPlanejamento) ExecucaoPlanejamentoFalhaResponse {
	return ExecucaoPlanejamentoFalhaResponse{
		ID:                 execucao.ID,
		DataViagem:         execucao.DataViagem.Format("2006-01-02"),
		Turno:              execucao.Turno,
		MunicipioDestinoID: execucao.MunicipioDestinoID,
		RotaInternaID:      execucao.RotaInternaID,
		Sentido:            execucao.Sentido,
		PartidaEm:          execucao.PartidaEm.Format(time.RFC3339),
		FechamentoEm:       execucao.FechamentoEm.Format(time.RFC3339),
		Status:             execucao.Status,
		Tentativas:         execucao.Tentativas,
		UltimoErro:         execucao.UltimoErro,
		ProximaTentativaEm: formatOptionalTime(execucao.ProximaTentativaEm),
		FinalizadoEm:       formatOptionalTime(execucao.FinalizadoEm),
	}
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}
