package viagens

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type HorarioTurnoViagemHandler struct {
	svc HorarioTurnoViagemService
}

func NewHorarioTurnoViagemHandler(svc HorarioTurnoViagemService) *HorarioTurnoViagemHandler {
	return &HorarioTurnoViagemHandler{svc: svc}
}

type HorarioTurnoViagemRequest struct {
	Cidade       string      `json:"cidade"`
	Turno        TurnoViagem `json:"turno"`
	HorarioIda   string      `json:"horario_ida"`
	HorarioVolta string      `json:"horario_volta"`
}

type HorarioTurnoViagemResponse struct {
	ID           int64       `json:"id"`
	Cidade       string      `json:"cidade"`
	Turno        TurnoViagem `json:"turno"`
	HorarioIda   string      `json:"horario_ida"`
	HorarioVolta string      `json:"horario_volta"`
	CreatedAt    string      `json:"created_at"`
	UpdatedAt    string      `json:"updated_at"`
}

func (h *HorarioTurnoViagemHandler) Create(w http.ResponseWriter, r *http.Request) {
	input, err := decodeHorarioTurnoInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	horario, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err, "failed to create horario turno viagem")
		return
	}

	httputils.Respond(w, http.StatusCreated, toHorarioTurnoViagemResponse(horario))
}

func (h *HorarioTurnoViagemHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := conv.ParseInt(r, "horarioTurnoID")
	if err != nil {
		http.Error(w, "invalid horario turno id", http.StatusBadRequest)
		return
	}

	horario, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err, "failed to get horario turno viagem")
		return
	}

	httputils.Respond(w, http.StatusOK, toHorarioTurnoViagemResponse(horario))
}

func (h *HorarioTurnoViagemHandler) List(w http.ResponseWriter, r *http.Request) {
	horarios, err := h.svc.List(r.Context())
	if err != nil {
		h.handleError(w, err, "failed to list horarios turno viagem")
		return
	}

	resp := make([]HorarioTurnoViagemResponse, 0, len(horarios))
	for _, horario := range horarios {
		resp = append(resp, toHorarioTurnoViagemResponse(&horario))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *HorarioTurnoViagemHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := conv.ParseInt(r, "horarioTurnoID")
	if err != nil {
		http.Error(w, "invalid horario turno id", http.StatusBadRequest)
		return
	}

	var req HorarioTurnoViagemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	horario, err := h.svc.Update(r.Context(), id, func(horario *HorarioTurnoViagem) (bool, error) {
		changed := false

		if req.Cidade != "" && req.Cidade != horario.Cidade {
			horario.Cidade = req.Cidade
			changed = true
		}
		if req.Turno != "" && req.Turno != horario.Turno {
			horario.Turno = req.Turno
			changed = true
		}
		if req.HorarioIda != "" {
			parsed, err := parseHorarioTurno(req.HorarioIda)
			if err != nil {
				return false, err
			}
			if parsed != horario.HorarioIda {
				horario.HorarioIda = parsed
				changed = true
			}
		}
		if req.HorarioVolta != "" {
			parsed, err := parseHorarioTurno(req.HorarioVolta)
			if err != nil {
				return false, err
			}
			if parsed != horario.HorarioVolta {
				horario.HorarioVolta = parsed
				changed = true
			}
		}

		return changed, nil
	})
	if err != nil {
		h.handleError(w, err, "failed to update horario turno viagem")
		return
	}

	httputils.Respond(w, http.StatusOK, toHorarioTurnoViagemResponse(horario))
}

func (h *HorarioTurnoViagemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := conv.ParseInt(r, "horarioTurnoID")
	if err != nil {
		http.Error(w, "invalid horario turno id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.handleError(w, err, "failed to delete horario turno viagem")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HorarioTurnoViagemHandler) handleError(w http.ResponseWriter, err error, msg string) {
	if errors.Is(err, brerror.ErrInvalidInput) {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if errors.Is(err, brerror.ErrAlreadyExists) {
		http.Error(w, "resource already exists", http.StatusConflict)
		return
	}
	if errors.Is(err, brerror.ErrNotFound) {
		http.Error(w, "resource not found", http.StatusNotFound)
		return
	}

	slog.Error(msg, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func decodeHorarioTurnoInput(r *http.Request) (HorarioTurnoViagemInput, error) {
	var req HorarioTurnoViagemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return HorarioTurnoViagemInput{}, errors.New("invalid request body")
	}

	horarioIda, err := parseHorarioTurno(req.HorarioIda)
	if err != nil {
		return HorarioTurnoViagemInput{}, errors.New("horario_ida must be in format HH:MM or HH:MM:SS")
	}

	horarioVolta, err := parseHorarioTurno(req.HorarioVolta)
	if err != nil {
		return HorarioTurnoViagemInput{}, errors.New("horario_volta must be in format HH:MM or HH:MM:SS")
	}

	return HorarioTurnoViagemInput{
		Cidade:       req.Cidade,
		Turno:        req.Turno,
		HorarioIda:   horarioIda,
		HorarioVolta: horarioVolta,
	}, nil
}

func toHorarioTurnoViagemResponse(h *HorarioTurnoViagem) HorarioTurnoViagemResponse {
	return HorarioTurnoViagemResponse{
		ID:           h.ID,
		Cidade:       h.Cidade,
		Turno:        h.Turno,
		HorarioIda:   formatHorarioTurno(h.HorarioIda),
		HorarioVolta: formatHorarioTurno(h.HorarioVolta),
		CreatedAt:    h.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    h.UpdatedAt.Format(time.RFC3339),
	}
}
