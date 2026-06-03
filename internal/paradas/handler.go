package paradas

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
	Nome      string  `json:"nome"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Cidade    string  `json:"cidade"`
}

type ParadaResponse struct {
	ID        int64   `json:"id"`
	Nome      string  `json:"nome"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Cidade    string  `json:"cidade"`
}

type ParadaHandler struct {
	store ParadaStore
}

func NewParadaHandler(store ParadaStore) *ParadaHandler {
	return &ParadaHandler{store: store}
}

func (h *ParadaHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ParadaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Nome == "" {
		http.Error(w, "nome is required", http.StatusBadRequest)
		return
	}
	if req.Cidade == "" {
		http.Error(w, "cidade is required", http.StatusBadRequest)
		return
	}

	input := ParadaInput{
		Nome:      strings.TrimSpace(req.Nome),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Cidade:    strings.TrimSpace(strings.ToLower(req.Cidade)),
	}

	parada, err := h.store.Create(ctx, input)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, toParadaResponse(parada))
}

func (h *ParadaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paradaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	parada, err := h.store.GetByID(ctx, paradaID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "parada not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toParadaResponse(parada))
}

func (h *ParadaHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paradas, err := h.store.List(ctx)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]ParadaResponse, 0, len(paradas))
	for _, p := range paradas {
		resp = append(resp, toParadaResponse(&p))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *ParadaHandler) ListByCity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cidade := strings.TrimSpace(strings.ToLower(chi.URLParam(r, "cidade")))
	if cidade == "" {
		http.Error(w, "cidade is required", http.StatusBadRequest)
		return
	}

	paradas, err := h.store.ListByCity(ctx, cidade)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]ParadaResponse, 0, len(paradas))
	for _, p := range paradas {
		resp = append(resp, toParadaResponse(&p))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func toParadaResponse(p *Parada) ParadaResponse {
	return ParadaResponse{
		ID:        p.ID,
		Nome:      p.Nome,
		Latitude:  p.Latitude,
		Longitude: p.Longitude,
		Cidade:    p.Cidade,
	}
}
