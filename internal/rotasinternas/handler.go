package rotasinternas

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
	"github.com/go-chi/chi/v5"
)

type ParadaRequest struct {
	ParadaID int64 `json:"parada_id"`
	Ordem    int   `json:"ordem"`
}

type CreateRotaInternaRequest struct {
	Cidade  string          `json:"cidade"`
	Paradas []ParadaRequest `json:"paradas"`
}

type UpdateParadasRequest struct {
	Paradas []ParadaRequest `json:"paradas"`
}

type ParadaResponse struct {
	ID        int64   `json:"id"`
	Nome      string  `json:"nome"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Cidade    string  `json:"cidade"`
	Ordem     int     `json:"ordem"`
}

type RotaInternaResponse struct {
	ID      int64            `json:"id"`
	Cidade  string           `json:"cidade"`
	Paradas []ParadaResponse `json:"paradas"`
}

type RotaInternaHandler struct {
	svc RotaInternaService
}

func NewRotaInternaHandler(svc RotaInternaService) *RotaInternaHandler {
	return &RotaInternaHandler{svc: svc}
}

func (h *RotaInternaHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRotaInternaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Cidade == "" {
		http.Error(w, "cidade is required", http.StatusBadRequest)
		return
	}

	input := CreateRotaInternaInput{
		Cidade:  strings.TrimSpace(strings.ToLower(req.Cidade)),
		Paradas: toParadaInputs(req.Paradas),
	}

	rota, err := h.svc.Create(ctx, input)
	if err != nil {
		if errors.Is(err, ErrOrdemDuplicada) || errors.Is(err, ErrSemParadas) || errors.Is(err, ErrParadaInvalida) {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, toRotaInternaResponse(rota))
}

func (h *RotaInternaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rotaInternaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	rota, err := h.svc.GetByID(ctx, rotaInternaID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "rota interna not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toRotaInternaResponse(rota))
}

func (h *RotaInternaHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rotas, err := h.svc.List(ctx)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]RotaInternaResponse, 0, len(rotas))
	for _, rota := range rotas {
		resp = append(resp, toRotaInternaResponse(&rota))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *RotaInternaHandler) ListByCity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cidade := strings.TrimSpace(strings.ToLower(chi.URLParam(r, "cidade")))
	if cidade == "" {
		http.Error(w, "cidade is required", http.StatusBadRequest)
		return
	}

	rotas, err := h.svc.ListByCity(ctx, cidade)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]RotaInternaResponse, 0, len(rotas))
	for _, rota := range rotas {
		resp = append(resp, toRotaInternaResponse(&rota))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *RotaInternaHandler) UpdateParadas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rotaInternaID, err := conv.ParseInt(r, "id")

	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req UpdateParadasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input := UpdateParadasInput{
		Paradas: toParadaInputs(req.Paradas),
	}

	rota, err := h.svc.UpdateParadas(ctx, rotaInternaID, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "rota interna not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrOrdemDuplicada) || errors.Is(err, ErrSemParadas) || errors.Is(err, ErrParadaInvalida) {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toRotaInternaResponse(rota))
}

func (h *RotaInternaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rotaInternaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(ctx, rotaInternaID); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "rota interna not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toParadaInputs(paradas []ParadaRequest) []ParadaInput {
	inputs := make([]ParadaInput, 0, len(paradas))
	for _, p := range paradas {
		inputs = append(inputs, ParadaInput{
			ParadaID: p.ParadaID,
			Ordem:    p.Ordem,
		})
	}
	return inputs
}

func toRotaInternaResponse(r *RotaInterna) RotaInternaResponse {
	paradas := make([]ParadaResponse, 0, len(r.Paradas))
	for _, p := range r.Paradas {
		paradas = append(paradas, ParadaResponse{
			ID:        p.ID,
			Nome:      p.Nome,
			Latitude:  p.Latitude,
			Longitude: p.Longitude,
			Cidade:    p.Cidade,
			Ordem:     p.Ordem,
		})
	}
	return RotaInternaResponse{
		ID:      r.ID,
		Cidade:  r.Cidade,
		Paradas: paradas,
	}
}
