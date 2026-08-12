package veiculos

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
	"github.com/fredsaggio/bondrota-api/internal/validation"
)

type VeiculoHandler struct {
	store VeiculoStore
}

func NewVeiculoHandler(store VeiculoStore) *VeiculoHandler {
	return &VeiculoHandler{store: store}
}

type CreateVeiculoRequest struct {
	Placa          string           `json:"placa"`
	Modelo         string           `json:"modelo"`
	Categoria      CategoriaVeiculo `json:"categoria"`
	Capacidade     int16            `json:"capacidade"`
	Status         StatusVeiculo    `json:"status"`
	ArCondicionado bool             `json:"ar_condicionado"`
	Banheiro       bool             `json:"banheiro"`
	Persiana       bool             `json:"persiana"`
	LuzLeitura     bool             `json:"luz_leitura"`
	Tomada         bool             `json:"tomada"`
}

type CreateVeiculoResponse struct {
	ID int64 `json:"id"`
}

type VeiculoResponse struct {
	ID             int64            `json:"id"`
	Placa          string           `json:"placa"`
	Modelo         string           `json:"modelo"`
	Categoria      CategoriaVeiculo `json:"categoria"`
	Capacidade     int16            `json:"capacidade"`
	Status         StatusVeiculo    `json:"status"`
	ArCondicionado bool             `json:"ar_condicionado"`
	Banheiro       bool             `json:"banheiro"`
	Persiana       bool             `json:"persiana"`
	LuzLeitura     bool             `json:"luz_leitura"`
	Tomada         bool             `json:"tomada"`
}

type UpdateVeiculoRequest struct {
	Placa          string           `json:"placa"`
	Modelo         string           `json:"modelo"`
	Categoria      CategoriaVeiculo `json:"categoria"`
	Capacidade     int16            `json:"capacidade"`
	Status         StatusVeiculo    `json:"status"`
	ArCondicionado *bool            `json:"ar_condicionado"`
	Banheiro       *bool            `json:"banheiro"`
	Persiana       *bool            `json:"persiana"`
	LuzLeitura     *bool            `json:"luz_leitura"`
	Tomada         *bool            `json:"tomada"`
}

func (h *VeiculoHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateVeiculoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := ValidateCategoriaCapacidade(req.Categoria, req.Capacidade); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	placa, err := validation.Placa(req.Placa)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	veiculo, err := h.store.Create(ctx, VeiculoInput{
		Placa:          placa,
		Modelo:         req.Modelo,
		Categoria:      req.Categoria,
		Capacidade:     req.Capacidade,
		Status:         req.Status,
		ArCondicionado: req.ArCondicionado,
		Banheiro:       req.Banheiro,
		Persiana:       req.Persiana,
		LuzLeitura:     req.LuzLeitura,
		Tomada:         req.Tomada,
	})
	if err != nil {
		slog.Error("failed to create veiculo", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, CreateVeiculoResponse{ID: veiculo.ID})
}

func (h *VeiculoHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vehicleID, err := conv.ParseInt(r, "veiculoID")
	if err != nil {
		http.Error(w, "invalid veiculo id", http.StatusBadRequest)
		return
	}

	veiculo, err := h.store.GetByID(ctx, vehicleID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "veiculo not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get veiculo", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toVeiculoResponse(veiculo))
}

func (h *VeiculoHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	veiculos, err := h.store.List(ctx)
	if err != nil {
		slog.Error("failed to list veiculos", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]VeiculoResponse, 0, len(veiculos))
	for _, v := range veiculos {
		resp = append(resp, toVeiculoResponse(&v))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *VeiculoHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vehicleID, err := conv.ParseInt(r, "veiculoID")
	if err != nil {
		http.Error(w, "invalid veiculo id", http.StatusBadRequest)
		return
	}

	var req UpdateVeiculoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	veiculo, err := h.store.Update(ctx, vehicleID, func(v *Veiculo) (bool, error) {
		changed := false
		if req.Placa != "" {
			placa, err := validation.Placa(req.Placa)
			if err != nil {
				return false, err
			}
			if placa != v.Placa {
				v.Placa = placa
				changed = true
			}
		}
		if req.Modelo != "" && req.Modelo != v.Modelo {
			v.Modelo = req.Modelo
			changed = true
		}
		if req.Categoria != "" && req.Categoria != v.Categoria {
			v.Categoria = req.Categoria
			changed = true
		}
		if req.Capacidade > 0 && req.Capacidade != v.Capacidade {
			v.Capacidade = req.Capacidade
			changed = true
		}
		if req.Status != "" && req.Status != v.Status {
			v.Status = req.Status
			changed = true
		}
		if req.ArCondicionado != nil && *req.ArCondicionado != v.ArCondicionado {
			v.ArCondicionado = *req.ArCondicionado
			changed = true
		}
		if req.Banheiro != nil && *req.Banheiro != v.Banheiro {
			v.Banheiro = *req.Banheiro
			changed = true
		}
		if req.Persiana != nil && *req.Persiana != v.Persiana {
			v.Persiana = *req.Persiana
			changed = true
		}
		if req.LuzLeitura != nil && *req.LuzLeitura != v.LuzLeitura {
			v.LuzLeitura = *req.LuzLeitura
			changed = true
		}
		if req.Tomada != nil && *req.Tomada != v.Tomada {
			v.Tomada = *req.Tomada
			changed = true
		}
		if err := ValidateCategoriaCapacidade(v.Categoria, v.Capacidade); err != nil {
			return false, err
		}
		return changed, nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "veiculo not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, validation.ErrPlacaInvalida) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, brerror.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		slog.Error("failed to update veiculo", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toVeiculoResponse(veiculo))
}

func (h *VeiculoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vehicleID, err := conv.ParseInt(r, "veiculoID")

	if err != nil {
		http.Error(w, "invalid veiculo id", http.StatusBadRequest)
		return
	}

	err = h.store.Delete(ctx, vehicleID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "veiculo not found", http.StatusNotFound)
			return
		}
		if db.IsAnyForeignKeyViolation(err) {
			http.Error(w, "veículo alocado em ciclos de viagem e não pode ser excluído", http.StatusConflict)
			return
		}
		slog.Error("failed to delete veiculo", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toVeiculoResponse(v *Veiculo) VeiculoResponse {
	return VeiculoResponse{
		ID:             v.ID,
		Placa:          v.Placa,
		Modelo:         v.Modelo,
		Categoria:      v.Categoria,
		Capacidade:     v.Capacidade,
		Status:         v.Status,
		ArCondicionado: v.ArCondicionado,
		Banheiro:       v.Banheiro,
		Persiana:       v.Persiana,
		LuzLeitura:     v.LuzLeitura,
		Tomada:         v.Tomada,
	}
}
