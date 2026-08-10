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
	DataViagem         string      `json:"data_viagem"`
	Turno              TurnoViagem `json:"turno"`
	MunicipioDestinoID int64       `json:"municipio_destino_id"`
	RotaInternaID      int64       `json:"rota_interna_id"`
}

type PlanejamentoViagensResponse struct {
	Ciclos                  []CicloComViagensResponse `json:"ciclos"`
	QuantidadeReservasIda   int                       `json:"quantidade_reservas_ida"`
	QuantidadeReservasVolta int                       `json:"quantidade_reservas_volta"`
	CapacidadeTotal         int                       `json:"capacidade_total"`
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

	input, err := toPlanejamentoInput(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validatePlanejamentoInput(input); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	ciclo, err := h.svc.Planejar(r.Context(), input)
	if err != nil {
		h.handleError(w, err, "failed to plan viagens")
		return
	}

	httputils.Respond(w, http.StatusCreated, toPlanejamentoViagensResponse(ciclo))
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
	if errors.Is(err, brerror.ErrInvalidInput) {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	slog.Error(msg, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func toPlanejamentoInput(req PlanejarViagensRequest) (PlanejamentoViagensInput, error) {
	dataViagem, err := parseDate(req.DataViagem, "data_viagem")
	if err != nil {
		return PlanejamentoViagensInput{}, err
	}

	return PlanejamentoViagensInput{
		DataViagem:         dataViagem,
		Turno:              req.Turno,
		MunicipioDestinoID: req.MunicipioDestinoID,
		RotaInternaID:      req.RotaInternaID,
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

func toPlanejamentoViagensResponse(p *PlanejamentoViagens) PlanejamentoViagensResponse {
	resp := PlanejamentoViagensResponse{
		Ciclos:                  make([]CicloComViagensResponse, 0, len(p.Ciclos)),
		QuantidadeReservasIda:   p.QuantidadeReservasIda,
		QuantidadeReservasVolta: p.QuantidadeReservasVolta,
		CapacidadeTotal:         p.CapacidadeTotal,
	}

	for _, ciclo := range p.Ciclos {
		resp.Ciclos = append(resp.Ciclos, toCicloComViagensResponse(&ciclo))
	}

	return resp
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
