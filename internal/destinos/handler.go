package destinos

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
	"github.com/go-chi/chi/v5"
)

type DestinoRequest struct {
	Nome      string  `json:"nome"`
	Rua       string  `json:"rua"`
	Cidade    string  `json:"cidade"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type CreateDestinoResponse struct {
	ID int64 `json:"id"`
}
type DestinoResponse struct {
	ID        int64   `json:"id"`
	Nome      string  `json:"nome"`
	Rua       string  `json:"rua"`
	Cidade    string  `json:"cidade"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type DestinoHandler struct {
	store DestinoStore
}

func NewDestinoHandler(store DestinoStore) *DestinoHandler {
	return &DestinoHandler{store: store}
}

func (h *DestinoHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req DestinoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Nome == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Rua == "" {
		http.Error(w, "street is required", http.StatusBadRequest)
		return
	}
	if req.Cidade == "" {
		http.Error(w, "city is required", http.StatusBadRequest)
		return
	}
	if req.Latitude == 0 || req.Longitude == 0 {
		http.Error(w, "latitude and longitude are required", http.StatusBadRequest)
		return
	}

	cidade := strings.TrimSpace(strings.ToLower(req.Cidade))

	input := DestinoInput{
		Nome:      req.Nome,
		Rua:       req.Rua,
		Cidade:    cidade,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}

	destino, err := h.store.Create(ctx, input)
	if err != nil {
		slog.Error("failed to create destino", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, CreateDestinoResponse{ID: destino.ID})
}

func (h *DestinoHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	destinoID, err := conv.ParseInt(r, "id")

	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	destino, err := h.store.GetByID(ctx, destinoID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "destino not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get destino", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toDestinoResponse(destino))
}

func (h *DestinoHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	destinos, err := h.store.List(ctx)
	if err != nil {
		slog.Error("failed to list destinos", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]DestinoResponse, 0, len(destinos))
	for _, p := range destinos {
		resp = append(resp, toDestinoResponse(&p))
	}
	httputils.Respond(w, http.StatusOK, resp)
}

func (h *DestinoHandler) ListByCity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cidade := strings.TrimSpace(strings.ToLower(chi.URLParam(r, "cidade")))

	if cidade == "" {
		http.Error(w, "cidade is required", http.StatusBadRequest)
		return
	}

	destinos, err := h.store.ListByCity(ctx, cidade)
	if err != nil {
		slog.Error("failed to list destinos by city", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]DestinoResponse, 0, len(destinos))
	for _, p := range destinos {
		resp = append(resp, toDestinoResponse(&p))
	}
	httputils.Respond(w, http.StatusOK, resp)
}

func (h *DestinoHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	destinoID, err := conv.ParseInt(r, "id")

	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req DestinoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	destino, err := h.store.Update(ctx, destinoID, func(p *Destino) (bool, error) {
		updated := false
		if req.Nome != "" && req.Nome != p.Nome {
			p.Nome = req.Nome
			updated = true
		}
		if req.Rua != "" && req.Rua != p.Rua {
			p.Rua = req.Rua
			updated = true
		}
		if req.Cidade != "" && req.Cidade != p.Cidade {
			p.Cidade = req.Cidade
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
			http.Error(w, "destino not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to update destino", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toDestinoResponse(destino))
}

func (h *DestinoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	err = h.store.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "destino not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to delete destino", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toDestinoResponse(p *Destino) DestinoResponse {
	return DestinoResponse{
		ID:        p.ID,
		Nome:      p.Nome,
		Rua:       p.Rua,
		Cidade:    p.Cidade,
		Latitude:  p.Latitude,
		Longitude: p.Longitude,
	}
}
