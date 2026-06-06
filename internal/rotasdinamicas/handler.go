package rotasdinamicas

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type RotaDinamicaHandler struct {
	svc           RotaDinamicaService
	calculadorSvc CalculadorRotaDinamicaService
}

func NewRotaDinamicaHandler(svc RotaDinamicaService, calculadorSvc ...CalculadorRotaDinamicaService) *RotaDinamicaHandler {
	h := &RotaDinamicaHandler{svc: svc}
	if len(calculadorSvc) > 0 {
		h.calculadorSvc = calculadorSvc[0]
	}
	return h
}

type PontoRotaRequest struct {
	Nome      string  `json:"nome"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type RotaDinamicaDestinoRequest struct {
	DestinoID int64 `json:"destino_id"`
}

type RotaDinamicaRequest struct {
	Provider        string                       `json:"provider"`
	Origem          PontoRotaRequest             `json:"origem"`
	DestinoFinal    PontoRotaRequest             `json:"destino_final"`
	DistanciaMetros int                          `json:"distancia_metros"`
	DuracaoSegundos int                          `json:"duracao_segundos"`
	Geometry        json.RawMessage              `json:"geometry"`
	ExpiresAt       string                       `json:"expires_at"`
	Destinos        []RotaDinamicaDestinoRequest `json:"destinos"`
}

type PontoRotaResponse struct {
	Nome      string  `json:"nome"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type RotaDinamicaResponse struct {
	ID              int64             `json:"id"`
	ViagemID        int64             `json:"viagem_id"`
	Provider        string            `json:"provider"`
	Origem          PontoRotaResponse `json:"origem"`
	DestinoFinal    PontoRotaResponse `json:"destino_final"`
	DistanciaMetros int               `json:"distancia_metros"`
	DuracaoSegundos int               `json:"duracao_segundos"`
	Geometry        json.RawMessage   `json:"geometry"`
	ExpiresAt       string            `json:"expires_at"`
	CreatedAt       string            `json:"created_at"`
	UpdatedAt       string            `json:"updated_at"`
}

type RotaDinamicaDestinoResponse struct {
	ID             int64  `json:"id"`
	RotaDinamicaID int64  `json:"rota_dinamica_id"`
	DestinoID      int64  `json:"destino_id"`
	Ordem          int    `json:"ordem"`
	CreatedAt      string `json:"created_at"`
}

type RotaDinamicaComDestinosResponse struct {
	Rota     RotaDinamicaResponse          `json:"rota"`
	Destinos []RotaDinamicaDestinoResponse `json:"destinos"`
}

func (h *RotaDinamicaHandler) Create(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	var req RotaDinamicaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input, err := toRotaDinamicaInput(viagemID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rota, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err, "failed to create rota dinamica")
		return
	}

	httputils.Respond(w, http.StatusCreated, toRotaDinamicaComDestinosResponse(rota))
}

func (h *RotaDinamicaHandler) GetByViagem(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	rota, err := h.svc.GetByViagem(r.Context(), viagemID)
	if err != nil {
		h.handleError(w, err, "failed to get rota dinamica")
		return
	}

	httputils.Respond(w, http.StatusOK, toRotaDinamicaComDestinosResponse(rota))
}

func (h *RotaDinamicaHandler) Calcular(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}
	if h.calculadorSvc == nil {
		http.Error(w, "calculation service unavailable", http.StatusInternalServerError)
		return
	}

	rota, err := h.calculadorSvc.Calcular(r.Context(), viagemID)
	if err != nil {
		h.handleError(w, err, "failed to calculate rota dinamica")
		return
	}

	httputils.Respond(w, http.StatusCreated, toRotaDinamicaComDestinosResponse(rota))
}

func (h *RotaDinamicaHandler) DeleteByViagem(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteByViagem(r.Context(), viagemID); err != nil {
		h.handleError(w, err, "failed to delete rota dinamica")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RotaDinamicaHandler) handleError(w http.ResponseWriter, err error, msg string) {
	if errors.Is(err, brerror.ErrNotFound) {
		http.Error(w, "resource not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, brerror.ErrAlreadyExists) {
		http.Error(w, "resource already exists", http.StatusConflict)
		return
	}
	if errors.Is(err, brerror.ErrInvalidInput) {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	slog.Error(msg, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func toRotaDinamicaInput(viagemID int64, req RotaDinamicaRequest) (RotaDinamicaInput, error) {
	expiresAt, err := parseTimestamp(req.ExpiresAt, "expires_at")
	if err != nil {
		return RotaDinamicaInput{}, err
	}

	destinos := make([]RotaDinamicaDestinoInput, 0, len(req.Destinos))
	for _, destino := range req.Destinos {
		destinos = append(destinos, RotaDinamicaDestinoInput{
			DestinoID: destino.DestinoID,
		})
	}

	return RotaDinamicaInput{
		ViagemID:        viagemID,
		Provider:        req.Provider,
		Origem:          toPontoRota(req.Origem),
		DestinoFinal:    toPontoRota(req.DestinoFinal),
		DistanciaMetros: req.DistanciaMetros,
		DuracaoSegundos: req.DuracaoSegundos,
		Geometry:        req.Geometry,
		ExpiresAt:       expiresAt,
		Destinos:        destinos,
	}, nil
}

func toPontoRota(req PontoRotaRequest) PontoRota {
	return PontoRota{
		Nome:      req.Nome,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}
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

func toRotaDinamicaComDestinosResponse(data *RotaDinamicaComDestinos) RotaDinamicaComDestinosResponse {
	return RotaDinamicaComDestinosResponse{
		Rota:     toRotaDinamicaResponse(&data.Rota),
		Destinos: toRotaDinamicaDestinoResponses(data.Destinos),
	}
}

func toRotaDinamicaResponse(rota *RotaDinamica) RotaDinamicaResponse {
	return RotaDinamicaResponse{
		ID:       rota.ID,
		ViagemID: rota.ViagemID,
		Provider: rota.Provider,
		Origem: PontoRotaResponse{
			Nome:      rota.OrigemNome,
			Latitude:  rota.OrigemLatitude,
			Longitude: rota.OrigemLongitude,
		},
		DestinoFinal: PontoRotaResponse{
			Nome:      rota.DestinoFinalNome,
			Latitude:  rota.DestinoFinalLatitude,
			Longitude: rota.DestinoFinalLongitude,
		},
		DistanciaMetros: rota.DistanciaMetros,
		DuracaoSegundos: rota.DuracaoSegundos,
		Geometry:        rota.Geometry,
		ExpiresAt:       rota.ExpiresAt.Format(time.RFC3339),
		CreatedAt:       rota.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       rota.UpdatedAt.Format(time.RFC3339),
	}
}

func toRotaDinamicaDestinoResponses(destinos []RotaDinamicaDestino) []RotaDinamicaDestinoResponse {
	resp := make([]RotaDinamicaDestinoResponse, 0, len(destinos))
	for _, destino := range destinos {
		resp = append(resp, RotaDinamicaDestinoResponse{
			ID:             destino.ID,
			RotaDinamicaID: destino.RotaDinamicaID,
			DestinoID:      destino.DestinoID,
			Ordem:          destino.Ordem,
			CreatedAt:      destino.CreatedAt.Format(time.RFC3339),
		})
	}
	return resp
}
