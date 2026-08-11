package clientes

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type VinculoHandler struct {
	svc VinculoService
}

func NewVinculoHandler(svc VinculoService) *VinculoHandler {
	return &VinculoHandler{svc: svc}
}

type VinculoRequest struct {
	Tipo          TipoConta    `json:"tipo"`
	Turno         TurnoCliente `json:"turno"`
	DestinoID     int64        `json:"destino_id"`
	RotaInternaID int64        `json:"rota_interna_id"`
	Curso         string       `json:"curso"`
	Comprovante   string       `json:"comprovante"`
	Validade      string       `json:"validade"`
	HorariosFixos []DiaSemana  `json:"horarios_fixos"`
}

type HorarioFixoResponse struct {
	ID        int64     `json:"id"`
	VinculoID int64     `json:"vinculo_id"`
	DiaSemana DiaSemana `json:"dia_semana"`
}

type VinculoResponse struct {
	ID            int64                 `json:"id"`
	ClienteID     int64                 `json:"cliente_id"`
	Tipo          TipoConta             `json:"tipo"`
	Turno         TurnoCliente          `json:"turno"`
	DestinoID     int64                 `json:"destino_id"`
	RotaInternaID int64                 `json:"rota_interna_id"`
	Curso         string                `json:"curso"`
	Comprovante   string                `json:"comprovante"`
	Validade      string                `json:"validade"`
	HorariosFixos []HorarioFixoResponse `json:"horarios_fixos"`
}

// VinculoComClienteResponse acrescenta o nome do cliente ao vinculo. Ela e usada
// apenas na listagem administrativa, para que o painel identifique o passageiro
// sem consultar cada cliente.
type VinculoComClienteResponse struct {
	VinculoResponse
	ClienteNome string `json:"cliente_nome"`
}

func (h *VinculoHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "invalid cliente id", http.StatusBadRequest)
		return
	}

	var req VinculoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input, err := toVinculoInput(clienteID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vinculo, err := h.svc.Create(ctx, input)
	if err != nil {
		h.handleError(w, err, "failed to create vinculo")
		return
	}

	httputils.Respond(w, http.StatusCreated, toVinculoResponse(vinculo))
}

func (h *VinculoHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vinculos, err := h.svc.List(ctx)
	if err != nil {
		slog.Error("failed to list vinculos", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]VinculoComClienteResponse, 0, len(vinculos))
	for _, v := range vinculos {
		resp = append(resp, VinculoComClienteResponse{
			VinculoResponse: toVinculoResponse(&v.Vinculo),
			ClienteNome:     v.ClienteNome,
		})
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *VinculoHandler) ListByCliente(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "invalid cliente id", http.StatusBadRequest)
		return
	}

	vinculos, err := h.svc.ListByCliente(ctx, clienteID)
	if err != nil {
		slog.Error("failed to list vinculos", "error", err, "clienteID", clienteID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]VinculoResponse, 0, len(vinculos))
	for _, v := range vinculos {
		resp = append(resp, toVinculoResponse(&v))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *VinculoHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, vinculoID, err := parseNestedVinculoIDs(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	vinculo, err := h.svc.GetByID(ctx, vinculoID)
	if err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "vinculo not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get vinculo", "error", err, "vinculoID", vinculoID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if vinculo.ClienteID != clienteID {
		http.Error(w, "vinculo not found", http.StatusNotFound)
		return
	}

	httputils.Respond(w, http.StatusOK, toVinculoResponse(vinculo))
}

func (h *VinculoHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, vinculoID, err := parseNestedVinculoIDs(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	vinculo, err := h.svc.GetByID(ctx, vinculoID)
	if err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "vinculo not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get vinculo", "error", err, "vinculoID", vinculoID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if vinculo.ClienteID != clienteID {
		http.Error(w, "vinculo not found", http.StatusNotFound)
		return
	}

	var req VinculoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input, err := toVinculoUpdateInput(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vinculo, err = h.svc.Update(ctx, vinculoID, input)
	if err != nil {
		h.handleError(w, err, "failed to update vinculo")
		return
	}

	httputils.Respond(w, http.StatusOK, toVinculoResponse(vinculo))
}

func (h *VinculoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, vinculoID, err := parseNestedVinculoIDs(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	vinculo, err := h.svc.GetByID(ctx, vinculoID)
	if err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "vinculo not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get vinculo", "error", err, "vinculoID", vinculoID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if vinculo.ClienteID != clienteID {
		http.Error(w, "vinculo not found", http.StatusNotFound)
		return
	}

	if err := h.svc.Delete(ctx, vinculoID); err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "vinculo not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to delete vinculo", "error", err, "vinculoID", vinculoID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseNestedVinculoIDs(r *http.Request) (int64, int64, error) {
	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		return 0, 0, err
	}

	vinculoID, err := conv.ParseInt(r, "vinculoID")
	if err != nil {
		return 0, 0, err
	}

	return clienteID, vinculoID, nil
}

func (h *VinculoHandler) handleError(w http.ResponseWriter, err error, msg string) {
	if errors.Is(err, ErrVinculoNotFound) {
		http.Error(w, "vinculo not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrTipoInvalido) ||
		errors.Is(err, ErrTurnoInvalido) ||
		errors.Is(err, ErrDiaInvalido) ||
		errors.Is(err, ErrDiaDuplicado) ||
		errors.Is(err, ErrCursoObrigatorio) {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if db.IsForeignKeyViolation(err, "cliente_vinculos_cliente_id_fkey") {
		http.Error(w, "cliente not found", http.StatusNotFound)
		return
	}
	if db.IsForeignKeyViolation(err, "cliente_vinculos_destino_id_fkey") {
		http.Error(w, "destino not found", http.StatusUnprocessableEntity)
		return
	}
	if db.IsForeignKeyViolation(err, "cliente_vinculos_rota_interna_id_fkey") {
		http.Error(w, "rota interna not found", http.StatusUnprocessableEntity)
		return
	}
	slog.Error(msg, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func toVinculoInput(clienteID int64, req VinculoRequest) (VinculoInput, error) {
	validade, err := validateVinculoRequest(req)
	if err != nil {
		return VinculoInput{}, err
	}

	return VinculoInput{
		ClienteID:     clienteID,
		Tipo:          req.Tipo,
		Turno:         req.Turno,
		DestinoID:     req.DestinoID,
		RotaInternaID: req.RotaInternaID,
		Curso:         strings.TrimSpace(req.Curso),
		Comprovante:   strings.TrimSpace(req.Comprovante),
		Validade:      validade,
		HorariosFixos: req.HorariosFixos,
	}, nil
}

func toVinculoUpdateInput(req VinculoRequest) (VinculoUpdateInput, error) {
	validade, err := validateVinculoRequest(req)
	if err != nil {
		return VinculoUpdateInput{}, err
	}

	return VinculoUpdateInput{
		Tipo:          req.Tipo,
		Turno:         req.Turno,
		DestinoID:     req.DestinoID,
		RotaInternaID: req.RotaInternaID,
		Curso:         strings.TrimSpace(req.Curso),
		Comprovante:   strings.TrimSpace(req.Comprovante),
		Validade:      validade,
		HorariosFixos: req.HorariosFixos,
	}, nil
}

func validateVinculoRequest(req VinculoRequest) (time.Time, error) {
	if req.DestinoID <= 0 {
		return time.Time{}, errors.New("destino_id is required")
	}
	if req.RotaInternaID <= 0 {
		return time.Time{}, errors.New("rota_interna_id is required")
	}
	if req.Validade == "" {
		return time.Time{}, errors.New("validade is required")
	}

	validade, err := parseDate(req.Validade)
	if err != nil {
		return time.Time{}, errors.New("validade must be in format YYYY-MM-DD")
	}

	return validade, nil
}

func toVinculoResponse(v *Vinculo) VinculoResponse {
	horarios := make([]HorarioFixoResponse, 0, len(v.HorariosFixos))
	for _, h := range v.HorariosFixos {
		horarios = append(horarios, HorarioFixoResponse{
			ID:        h.ID,
			VinculoID: h.VinculoID,
			DiaSemana: h.DiaSemana,
		})
	}

	return VinculoResponse{
		ID:            v.ID,
		ClienteID:     v.ClienteID,
		Tipo:          v.Tipo,
		Turno:         v.Turno,
		DestinoID:     v.DestinoID,
		RotaInternaID: v.RotaInternaID,
		Curso:         v.Curso,
		Comprovante:   v.Comprovante,
		Validade:      v.Validade.Format("2006-01-02"),
		HorariosFixos: horarios,
	}
}
