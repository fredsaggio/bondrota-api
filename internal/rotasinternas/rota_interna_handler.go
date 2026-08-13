package rotasinternas

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type ParadaRequest struct {
	ParadaID int64 `json:"parada_id"`
	Ordem    int   `json:"ordem"`
}

type CreateRotaInternaRequest struct {
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
	Ordem     int     `json:"ordem"`
}

type RotaInternaResponse struct {
	ID      int64            `json:"id"`
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
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	input := CreateRotaInternaInput{
		Paradas: toParadaInputs(req.Paradas),
	}

	rota, err := h.svc.Create(ctx, input)
	if err != nil {
		if errors.Is(err, ErrOrdemDuplicada) || errors.Is(err, ErrSemParadas) || errors.Is(err, ErrParadaInvalida) {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, toRotaInternaResponse(rota))
}

func (h *RotaInternaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rotaInternaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	rota, err := h.svc.GetByID(ctx, rotaInternaID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Rota interna não encontrada.", http.StatusNotFound)
			return
		}
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toRotaInternaResponse(rota))
}

func (h *RotaInternaHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rotas, err := h.svc.List(ctx)
	if err != nil {
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
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
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	var req UpdateParadasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	input := UpdateParadasInput{
		Paradas: toParadaInputs(req.Paradas),
	}

	rota, err := h.svc.UpdateParadas(ctx, rotaInternaID, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Rota interna não encontrada.", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrOrdemDuplicada) || errors.Is(err, ErrSemParadas) || errors.Is(err, ErrParadaInvalida) {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toRotaInternaResponse(rota))
}

func (h *RotaInternaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rotaInternaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(ctx, rotaInternaID); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Rota interna não encontrada.", http.StatusNotFound)
			return
		}
		if db.IsAnyForeignKeyViolation(err) {
			http.Error(w, "Esta rota está em uso e não pode ser removida.", http.StatusConflict)
			return
		}
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
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
			Ordem:     p.Ordem,
		})
	}
	return RotaInternaResponse{
		ID:      r.ID,
		Paradas: paradas,
	}
}
