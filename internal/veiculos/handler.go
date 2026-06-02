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
