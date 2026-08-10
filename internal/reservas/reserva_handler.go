package reservas

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type ReservaHandler struct {
	svc ReservaService
}

func NewReservaHandler(svc ReservaService) *ReservaHandler {
	return &ReservaHandler{svc: svc}
}

type CreateReservaRequest struct {
	DataViagem string         `json:"data_viagem"`
	Turno      TurnoReserva   `json:"turno"`
	Sentido    SentidoReserva `json:"sentido"`
}

type UpdateReservaRequest struct {
	DataViagem string         `json:"data_viagem"`
	Turno      TurnoReserva   `json:"turno"`
	Sentido    SentidoReserva `json:"sentido"`
	Status     StatusReserva  `json:"status"`
}

type ReservaResponse struct {
	ID            int64          `json:"id"`
	ClienteID     int64          `json:"cliente_id"`
	VinculoID     int64          `json:"vinculo_id"`
	DataViagem    string         `json:"data_viagem"`
	Turno         TurnoReserva   `json:"turno"`
	DestinoID     int64          `json:"destino_id"`
	RotaInternaID int64          `json:"rota_interna_id"`
	Sentido       SentidoReserva `json:"sentido"`
	Status        StatusReserva  `json:"status"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

func (h *ReservaHandler) CreateByVinculo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, vinculoID, err := parseNestedVinculoIDs(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req CreateReservaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input, err := toReservaInput(clienteID, vinculoID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reserva, err := h.svc.Create(ctx, input)
	if err != nil {
		h.handleError(w, err, "failed to create reserva")
		return
	}

	httputils.Respond(w, http.StatusCreated, toReservaResponse(reserva))
}

func (h *ReservaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reservaID, err := conv.ParseInt(r, "reservaID")
	if err != nil {
		http.Error(w, "invalid reserva id", http.StatusBadRequest)
		return
	}

	reserva, err := h.svc.GetByID(ctx, reservaID)
	if err != nil {
		h.handleError(w, err, "failed to get reserva")
		return
	}

	httputils.Respond(w, http.StatusOK, toReservaResponse(reserva))
}

func (h *ReservaHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reservas, err := h.svc.List(ctx)
	if err != nil {
		slog.Error("failed to list reservas", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toReservaResponses(reservas))
}

func (h *ReservaHandler) ListByCliente(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "invalid cliente id", http.StatusBadRequest)
		return
	}

	reservas, err := h.svc.ListByCliente(ctx, clienteID)
	if err != nil {
		slog.Error("failed to list reservas by cliente", "error", err, "clienteID", clienteID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toReservaResponses(reservas))
}

func (h *ReservaHandler) ListByVinculo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, vinculoID, err := parseNestedVinculoIDs(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	reservas, err := h.svc.ListByVinculo(ctx, clienteID, vinculoID)
	if err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "vinculo not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to list reservas by vinculo", "error", err, "clienteID", clienteID, "vinculoID", vinculoID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toReservaResponses(reservas))
}

func (h *ReservaHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reservaID, err := conv.ParseInt(r, "reservaID")
	if err != nil {
		http.Error(w, "invalid reserva id", http.StatusBadRequest)
		return
	}

	var req UpdateReservaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	reserva, err := h.svc.Update(ctx, reservaID, func(reserva *Reserva) (bool, error) {
		return applyReservaUpdate(reserva, req)
	})
	if err != nil {
		h.handleError(w, err, "failed to update reserva")
		return
	}

	httputils.Respond(w, http.StatusOK, toReservaResponse(reserva))
}

func (h *ReservaHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reservaID, err := conv.ParseInt(r, "reservaID")
	if err != nil {
		http.Error(w, "invalid reserva id", http.StatusBadRequest)
		return
	}

	if !h.canCancelReserva(ctx, reservaID, w) {
		return
	}

	reserva, err := h.svc.Cancel(ctx, reservaID)
	if err != nil {
		h.handleError(w, err, "failed to cancel reserva")
		return
	}

	httputils.Respond(w, http.StatusOK, toReservaResponse(reserva))
}

func (h *ReservaHandler) canCancelReserva(ctx context.Context, reservaID int64, w http.ResponseWriter) bool {
	claims, ok := ctx.Value(auth.ClaimsKey).(*auth.Claims)
	if !ok || claims.UserID <= 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	if claims.Role != auth.RoleCliente {
		return true
	}

	reserva, err := h.svc.GetByID(ctx, reservaID)
	if err != nil {
		h.handleError(w, err, "failed to get reserva")
		return false
	}
	if reserva.ClienteID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (h *ReservaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reservaID, err := conv.ParseInt(r, "reservaID")
	if err != nil {
		http.Error(w, "invalid reserva id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(ctx, reservaID); err != nil {
		h.handleError(w, err, "failed to delete reserva")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ReservaHandler) handleError(w http.ResponseWriter, err error, msg string) {
	if errors.Is(err, ErrReservaNotFound) {
		http.Error(w, "reserva not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrVinculoNotFound) {
		http.Error(w, "vinculo not found", http.StatusNotFound)
		return
	}
	if db.IsUniqueViolation(err, "uq_reservas_ativas_vinculo_data_turno_sentido") {
		http.Error(w, "active reserva already exists for this vinculo, date, turno and sentido", http.StatusConflict)
		return
	}
	if errors.Is(err, ErrDataObrigatoria) ||
		errors.Is(err, ErrDataInvalida) ||
		errors.Is(err, ErrSentidoInvalido) ||
		errors.Is(err, ErrStatusInvalido) ||
		errors.Is(err, ErrTurnoInvalido) ||
		errors.Is(err, ErrTurnoObrigatorio) ||
		errors.Is(err, ErrTurnoIncompativel) ||
		errors.Is(err, ErrVinculoIDObrigatorio) {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	slog.Error(msg, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
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

func toReservaInput(clienteID, vinculoID int64, req CreateReservaRequest) (ReservaInput, error) {
	dataViagem, err := parseReservaDate(req.DataViagem)
	if err != nil {
		return ReservaInput{}, err
	}

	return ReservaInput{
		ClienteID:  clienteID,
		VinculoID:  vinculoID,
		DataViagem: dataViagem,
		Turno:      req.Turno,
		Sentido:    req.Sentido,
	}, nil
}

func applyReservaUpdate(reserva *Reserva, req UpdateReservaRequest) (bool, error) {
	updated := false

	if req.DataViagem != "" {
		dataViagem, err := parseReservaDate(req.DataViagem)
		if err != nil {
			return false, err
		}
		if !sameDate(dataViagem, reserva.DataViagem) {
			reserva.DataViagem = dataViagem
			updated = true
		}
	}
	if req.Turno != "" {
		if !isOperationalTurno(req.Turno) {
			return false, ErrTurnoInvalido
		}
		if req.Turno != reserva.Turno {
			reserva.Turno = req.Turno
			updated = true
		}
	}
	if req.Sentido != "" {
		if !isValidSentido(req.Sentido) {
			return false, ErrSentidoInvalido
		}
		if req.Sentido != reserva.Sentido {
			reserva.Sentido = req.Sentido
			updated = true
		}
	}
	if req.Status != "" {
		if !isValidStatus(req.Status) {
			return false, ErrStatusInvalido
		}
		if req.Status != reserva.Status {
			reserva.Status = req.Status
			updated = true
		}
	}
	return updated, nil
}

func parseReservaDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, ErrDataObrigatoria
	}
	data, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, ErrDataInvalida
	}
	return data, nil
}

func toReservaResponses(reservas []Reserva) []ReservaResponse {
	resp := make([]ReservaResponse, 0, len(reservas))
	for _, reserva := range reservas {
		resp = append(resp, toReservaResponse(&reserva))
	}
	return resp
}

func toReservaResponse(r *Reserva) ReservaResponse {
	return ReservaResponse{
		ID:            r.ID,
		ClienteID:     r.ClienteID,
		VinculoID:     r.VinculoID,
		DataViagem:    r.DataViagem.Format("2006-01-02"),
		Turno:         r.Turno,
		DestinoID:     r.DestinoID,
		RotaInternaID: r.RotaInternaID,
		Sentido:       r.Sentido,
		Status:        r.Status,
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     r.UpdatedAt.Format(time.RFC3339),
	}
}
