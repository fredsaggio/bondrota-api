package paradas

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type ParadaRequest struct {
	Nome      string  `json:"nome"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type ParadaResponse struct {
	ID        int64   `json:"id"`
	Nome      string  `json:"nome"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
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
	input := ParadaInput{
		Nome:      strings.TrimSpace(req.Nome),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}

	parada, err := h.store.Create(ctx, input)
	if err != nil {
		slog.Error("failed to create parada", "error", err)
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
		slog.Error("failed to get parada", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toParadaResponse(parada))
}

func (h *ParadaHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paradas, err := h.store.List(ctx)
	if err != nil {
		slog.Error("failed to list paradas", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]ParadaResponse, 0, len(paradas))
	for _, p := range paradas {
		resp = append(resp, toParadaResponse(&p))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *ParadaHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paradaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req ParadaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	parada, err := h.store.Update(ctx, paradaID, func(p *Parada) (bool, error) {
		updated := false
		if req.Nome != "" && req.Nome != p.Nome {
			p.Nome = strings.TrimSpace(req.Nome)
			updated = true
		}
		if req.Latitude != 0 && req.Latitude != p.Latitude {
			p.Latitude = req.Latitude
			updated = true
		}
		if req.Longitude != 0 && req.Longitude != p.Longitude {
			p.Longitude = req.Longitude
			updated = true
		}
		return updated, nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "parada not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to update parada", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toParadaResponse(parada))
}

func (h *ParadaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paradaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.store.Delete(ctx, paradaID); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "parada not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to delete parada", "error", err)
		http.Error(w, "parada em uso por uma rota interna", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toParadaResponse(p *Parada) ParadaResponse {
	return ParadaResponse{
		ID:        p.ID,
		Nome:      p.Nome,
		Latitude:  p.Latitude,
		Longitude: p.Longitude,
	}
}
