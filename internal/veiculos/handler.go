package veiculos

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/fredsaggio/bondrota-api/internal/httputils"
	"github.com/go-chi/chi/v5"
)

type VeiculoHandler struct {
	store VeiculoStore
}

func NewVeiculoHandler(store VeiculoStore) *VeiculoHandler {
	return &VeiculoHandler{store: store}
}

type CreateVeiculoRequest struct {
	Placa          string        `json:"placa"`
	Modelo         string        `json:"modelo"`
	Capacidade     int16         `json:"capacidade"`
	CidadeBase     string        `json:"cidade_base"`
	Status         StatusVeiculo `json:"status"`
	ArCondicionado bool          `json:"ar_condicionado"`
	Banheiro       bool          `json:"banheiro"`
	Persiana       bool          `json:"persiana"`
	LuzLeitura     bool          `json:"luz_leitura"`
	Tomada         bool          `json:"tomada"`
}

type CreateVeiculoResponse struct {
	ID int64 `json:"id"`
}

type VeiculoResponse struct {
	ID             int64         `json:"id"`
	Placa          string        `json:"placa"`
	Modelo         string        `json:"modelo"`
	Capacidade     int16         `json:"capacidade"`
	CidadeBase     string        `json:"cidade_base"`
	Status         StatusVeiculo `json:"status"`
	ArCondicionado bool          `json:"ar_condicionado"`
	Banheiro       bool          `json:"banheiro"`
	Persiana       bool          `json:"persiana"`
	LuzLeitura     bool          `json:"luz_leitura"`
	Tomada         bool          `json:"tomada"`
}

type UpdateVeiculoRequest struct {
	Placa          string        `json:"placa"`
	Modelo         string        `json:"modelo"`
	Capacidade     int16         `json:"capacidade"`
	CidadeBase     string        `json:"cidade_base"`
	Status         StatusVeiculo `json:"status"`
	ArCondicionado bool          `json:"ar_condicionado"`
	Banheiro       bool          `json:"banheiro"`
	Persiana       bool          `json:"persiana"`
	LuzLeitura     bool          `json:"luz_leitura"`
	Tomada         bool          `json:"tomada"`
}

func (h *VeiculoHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateVeiculoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	veiculo, err := h.store.Create(ctx, VeiculoInput{
		Placa:          req.Placa,
		Modelo:         req.Modelo,
		Capacidade:     req.Capacidade,
		CidadeBase:     req.CidadeBase,
		Status:         req.Status,
		ArCondicionado: req.ArCondicionado,
		Banheiro:       req.Banheiro,
		Persiana:       req.Persiana,
		LuzLeitura:     req.LuzLeitura,
		Tomada:         req.Tomada,
	})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, CreateVeiculoResponse{ID: veiculo.ID})
}

func (h *VeiculoHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "veiculoID")
	vehicleID, err := strconv.ParseInt(idStr, 10, 64)
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
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toVeiculoResponse(veiculo))
}

func (h *VeiculoHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	veiculos, err := h.store.List(ctx)
	if err != nil {
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

	id, err := strconv.ParseInt(chi.URLParam(r, "veiculoID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid veiculo id", http.StatusBadRequest)
		return
	}

	var req UpdateVeiculoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	veiculo, err := h.store.Update(ctx, id, func(v *Veiculo) (bool, error) {
		changed := false
		if req.Placa != "" && req.Placa != v.Placa {
			v.Placa = req.Placa
			changed = true
		}
		if req.Modelo != "" && req.Modelo != v.Modelo {
			v.Modelo = req.Modelo
			changed = true
		}
		if req.Capacidade > 0 && req.Capacidade != v.Capacidade {
			v.Capacidade = req.Capacidade
			changed = true
		}
		if req.CidadeBase != "" && req.CidadeBase != v.CidadeBase {
			v.CidadeBase = req.CidadeBase
			changed = true
		}
		if req.Status != "" && req.Status != v.Status {
			v.Status = req.Status
			changed = true
		}
		if req.ArCondicionado != v.ArCondicionado {
			v.ArCondicionado = req.ArCondicionado
			changed = true
		}
		if req.Banheiro != v.Banheiro {
			v.Banheiro = req.Banheiro
			changed = true
		}
		if req.Persiana != v.Persiana {
			v.Persiana = req.Persiana
			changed = true
		}
		if req.LuzLeitura != v.LuzLeitura {
			v.LuzLeitura = req.LuzLeitura
			changed = true
		}
		if req.Tomada != v.Tomada {
			v.Tomada = req.Tomada
			changed = true
		}
		return changed, nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "veiculo not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toVeiculoResponse(veiculo))
}

func toVeiculoResponse(v *Veiculo) VeiculoResponse {
	return VeiculoResponse{
		ID:             v.ID,
		Placa:          v.Placa,
		Modelo:         v.Modelo,
		Capacidade:     v.Capacidade,
		CidadeBase:     v.CidadeBase,
		Status:         v.Status,
		ArCondicionado: v.ArCondicionado,
		Banheiro:       v.Banheiro,
		Persiana:       v.Persiana,
		LuzLeitura:     v.LuzLeitura,
		Tomada:         v.Tomada,
	}
}
