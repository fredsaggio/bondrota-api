package viagens

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type PlanejamentoHandler struct {
	svc PlanejamentoService
}

func NewPlanejamentoHandler(svc PlanejamentoService) *PlanejamentoHandler {
	return &PlanejamentoHandler{svc: svc}
}

type PlanejarViagensRequest struct {
	DataViagem    string      `json:"data_viagem"`
	Turno         TurnoViagem `json:"turno"`
	Cidade        string      `json:"cidade"`
	RotaInternaID int64       `json:"rota_interna_id"`
	VeiculoID     int64       `json:"veiculo_id"`
	MotoristaID   int64       `json:"motorista_id"`
	ExpiresAt     string      `json:"expires_at"`
	PartidaIda    string      `json:"partida_ida"`
	PartidaVolta  string      `json:"partida_volta"`
}

type CicloComViagensResponse struct {
	Ciclo   CicloViagemResponse `json:"ciclo"`
	Viagens []ViagemResponse    `json:"viagens"`
}

func (h *PlanejamentoHandler) PlanejarViagens(w http.ResponseWriter, r *http.Request) {
	var req PlanejarViagensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input, partidas, err := toPlanejamentoInput(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validatePlanejamentoInput(input, partidas); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	ciclo, err := h.svc.Planejar(r.Context(), input, partidas)
	if err != nil {
		h.handleError(w, err, "failed to plan viagens")
		return
	}

	httputils.Respond(w, http.StatusCreated, toCicloComViagensResponse(ciclo))
}

func (h *PlanejamentoHandler) handleError(w http.ResponseWriter, err error, msg string) {
	if errors.Is(err, brerror.ErrAlreadyExists) {
		http.Error(w, "resource already exists", http.StatusConflict)
		return
	}
	if errors.Is(err, brerror.ErrNotFound) {
		http.Error(w, "resource not found", http.StatusNotFound)
		return
	}

	slog.Error(msg, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func toPlanejamentoInput(req PlanejarViagensRequest) (CicloViagemInput, map[SentidoViagem]time.Time, error) {
	dataViagem, err := parseDate(req.DataViagem, "data_viagem")
	if err != nil {
		return CicloViagemInput{}, nil, err
	}

	expiresAt, err := parseTimestamp(req.ExpiresAt, "expires_at")
	if err != nil {
		return CicloViagemInput{}, nil, err
	}

	partidaIda, err := parseTimestamp(req.PartidaIda, "partida_ida")
	if err != nil {
		return CicloViagemInput{}, nil, err
	}

	partidaVolta, err := parseTimestamp(req.PartidaVolta, "partida_volta")
	if err != nil {
		return CicloViagemInput{}, nil, err
	}

	return CicloViagemInput{
			DataViagem:    dataViagem,
			Turno:         req.Turno,
			Cidade:        req.Cidade,
			RotaInternaID: req.RotaInternaID,
			VeiculoID:     req.VeiculoID,
			MotoristaID:   req.MotoristaID,
			ExpiresAt:     expiresAt,
		}, map[SentidoViagem]time.Time{
			SentidoIda:   partidaIda,
			SentidoVolta: partidaVolta,
		}, nil
}

func parseDate(value, field string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New(field + " is required")
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, errors.New(field + " must be in format YYYY-MM-DD")
	}
	return parsed, nil
}

func parseTimestamp(value, field string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New(field + " is required")
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New(field + " must be in RFC3339 format")
	}
	return parsed, nil
}

func toCicloComViagensResponse(c *CicloComViagens) CicloComViagensResponse {
	return CicloComViagensResponse{
		Ciclo:   toCicloViagemResponse(&c.Ciclo),
		Viagens: toViagemResponses(c.Viagens),
	}
}

func toViagemResponses(viagens []Viagem) []ViagemResponse {
	resp := make([]ViagemResponse, 0, len(viagens))
	for _, viagem := range viagens {
		resp = append(resp, toViagemResponse(&viagem))
	}
	return resp
}
