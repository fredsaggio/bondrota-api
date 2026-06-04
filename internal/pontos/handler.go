package pontos

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

type PontoRequest struct {
	Nome      string  `json:"nome"`
	Rua       string  `json:"rua"`
	Cidade    string  `json:"cidade"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type CreatePontoResponse struct {
	ID int64 `json:"id"`
}
type PontoResponse struct {
	ID        int64   `json:"id"`
	Nome      string  `json:"nome"`
	Rua       string  `json:"rua"`
	Cidade    string  `json:"cidade"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type PontoHandler struct {
	store PontoStore
}

func NewPontoHandler(store PontoStore) *PontoHandler {
	return &PontoHandler{store: store}
}

func (h *PontoHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req PontoRequest
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

	input := PontoInput{
		Nome:      req.Nome,
		Rua:       req.Rua,
		Cidade:    cidade,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}

	ponto, err := h.store.Create(ctx, input)
	if err != nil {
		slog.Error("failed to create ponto", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, CreatePontoResponse{ID: ponto.ID})
}

func (h *PontoHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pontoID, err := conv.ParseInt(r, "id")

	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ponto, err := h.store.GetByID(ctx, pontoID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "ponto not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get ponto", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toPontoResponse(ponto))
}

func (h *PontoHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pontos, err := h.store.List(ctx)
	if err != nil {
		slog.Error("failed to list pontos", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]PontoResponse, 0, len(pontos))
	for _, p := range pontos {
		resp = append(resp, toPontoResponse(&p))
	}
	httputils.Respond(w, http.StatusOK, resp)
}

func (h *PontoHandler) ListByCity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cidade := strings.TrimSpace(strings.ToLower(chi.URLParam(r, "cidade")))

	if cidade == "" {
		http.Error(w, "cidade is required", http.StatusBadRequest)
		return
	}

	pontos, err := h.store.ListByCity(ctx, cidade)
	if err != nil {
		slog.Error("failed to list pontos by city", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]PontoResponse, 0, len(pontos))
	for _, p := range pontos {
		resp = append(resp, toPontoResponse(&p))
	}
	httputils.Respond(w, http.StatusOK, resp)
}

func (h *PontoHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pontoID, err := conv.ParseInt(r, "id")

	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req PontoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ponto, err := h.store.Update(ctx, pontoID, func(p *Ponto) (bool, error) {
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
			http.Error(w, "ponto not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to update ponto", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toPontoResponse(ponto))
}

func (h *PontoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	err = h.store.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "ponto not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to delete ponto", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toPontoResponse(p *Ponto) PontoResponse {
	return PontoResponse{
		ID:        p.ID,
		Nome:      p.Nome,
		Rua:       p.Rua,
		Cidade:    p.Cidade,
		Latitude:  p.Latitude,
		Longitude: p.Longitude,
	}
}
