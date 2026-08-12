package motoristas

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
	"github.com/fredsaggio/bondrota-api/internal/validation"
)

type CreateMotoristaRequest struct {
	Nome                string `json:"nome"`
	CPF                 string `json:"cpf"`
	Senha               string `json:"senha"`
	Telefone            string `json:"telefone"`
	DataNasc            string `json:"data_nasc"`
	Turno               Turno  `json:"turno"`
	MunicipioTrabalhoID int64  `json:"municipio_trabalho_id"`
	Residencia          string `json:"residencia"`
	Foto                string `json:"foto"`
}

type UpdateMotoristaRequest struct {
	Nome string `json:"nome"`
	// Telefone, Residencia e Foto sao ponteiros porque sao opcionais e podem ser
	// legitimamente limpos: chave ausente/null preserva o valor atual, string vazia
	// explicita apaga o campo. Os demais campos sao obrigatorios e nunca fazem
	// sentido em branco, entao continuam string simples.
	Telefone            *string `json:"telefone"`
	DataNasc            string  `json:"data_nasc"`
	Turno               Turno   `json:"turno"`
	MunicipioTrabalhoID int64   `json:"municipio_trabalho_id"`
	Residencia          *string `json:"residencia"`
	Foto                *string `json:"foto"`
}

type LoginRequest struct {
	CPF   string `json:"cpf"`
	Senha string `json:"senha"`
}

type MotoristaResponse struct {
	ID                  int64  `json:"id"`
	Nome                string `json:"nome"`
	CPF                 string `json:"cpf"`
	Telefone            string `json:"telefone"`
	DataNasc            string `json:"data_nasc"`
	Turno               Turno  `json:"turno"`
	MunicipioTrabalhoID int64  `json:"municipio_trabalho_id"`
	Residencia          string `json:"residencia"`
	Foto                string `json:"foto"`
}

type MotoristaHandler struct {
	svc MotoristaService
}

func NewMotoristaHandler(svc MotoristaService) *MotoristaHandler {
	return &MotoristaHandler{svc: svc}
}

func (h *MotoristaHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.CPF) == "" {
		http.Error(w, "cpf is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Senha) == "" {
		http.Error(w, "senha is required", http.StatusBadRequest)
		return
	}

	token, err := h.svc.Login(ctx, strings.TrimSpace(req.CPF), req.Senha)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		slog.Error("failed to login motorista", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, map[string]string{"token": token})
}

func (h *MotoristaHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateMotoristaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	nome := strings.TrimSpace(req.Nome)
	if nome == "" {
		http.Error(w, "nome is required", http.StatusBadRequest)
		return
	}
	if err := validation.Nome(nome); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.CPF) == "" {
		http.Error(w, "cpf is required", http.StatusBadRequest)
		return
	}
	cpf, err := validation.CPF(req.CPF)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Senha) == "" {
		http.Error(w, "senha is required", http.StatusBadRequest)
		return
	}
	if req.Turno == "" {
		http.Error(w, "turno is required", http.StatusBadRequest)
		return
	}
	if req.DataNasc == "" {
		http.Error(w, "data_nasc is required", http.StatusBadRequest)
		return
	}
	if req.MunicipioTrabalhoID <= 0 {
		http.Error(w, "municipio_trabalho_id is required", http.StatusBadRequest)
		return
	}

	switch req.Turno {
	case TurnoMatutino, TurnoVespertino, TurnoNoturno, TurnoIntegral:

	default:
		http.Error(w, "turno must be MT, VT, NT or IN", http.StatusBadRequest)
		return
	}

	dataNasc, err := time.Parse("2006-01-02", req.DataNasc)
	if err != nil {
		http.Error(w, "data_nasc must be in format YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	telefone, err := validation.Telefone(req.Telefone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input := MotoristaInput{
		Nome:                nome,
		CPF:                 cpf,
		Senha:               req.Senha,
		Telefone:            telefone,
		DataNasc:            dataNasc,
		Turno:               req.Turno,
		MunicipioTrabalhoID: req.MunicipioTrabalhoID,
		Residencia:          strings.TrimSpace(req.Residencia),
		Foto:                strings.TrimSpace(req.Foto),
	}

	motorista, err := h.svc.Create(ctx, input)
	if err != nil {
		if db.IsUniqueViolation(err, "motoristas_cpf_key") {
			http.Error(w, "cpf already exists", http.StatusConflict)
			return
		}
		slog.Error("failed to create motorista", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, toMotoristaResponse(motorista))
}

func (h *MotoristaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	motoristaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	motorista, err := h.svc.GetByID(ctx, motoristaID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "motorista not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get motorista", "error", err, "motoristaID", motoristaID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toMotoristaResponse(motorista))
}

func (h *MotoristaHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	motoristas, err := h.svc.List(ctx)
	if err != nil {
		slog.Error("failed to list motoristas", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]MotoristaResponse, 0, len(motoristas))
	for _, m := range motoristas {
		resp = append(resp, toMotoristaResponse(&m))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *MotoristaHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	motoristaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req UpdateMotoristaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	motorista, err := h.svc.Update(ctx, motoristaID, func(m *Motorista) (bool, error) {
		updated := false
		if req.Nome != "" {
			nome := strings.TrimSpace(req.Nome)
			if nome == "" {
				return false, ErrNomeObrigatorio
			}
			if err := validation.Nome(nome); err != nil {
				return false, err
			}
			if nome != m.Nome {
				m.Nome = nome
				updated = true
			}
		}
		if req.Telefone != nil {
			telefone, err := validation.Telefone(*req.Telefone)
			if err != nil {
				return false, err
			}
			if telefone != m.Telefone {
				m.Telefone = telefone
				updated = true
			}
		}
		if req.DataNasc != "" {
			dataNasc, err := time.Parse("2006-01-02", req.DataNasc)
			if err != nil {
				return false, ErrDataNascInvalida
			}
			if !dataNasc.Equal(m.DataNasc) {
				m.DataNasc = dataNasc
				updated = true
			}
		}
		if req.Turno != "" {
			switch req.Turno {
			case TurnoMatutino, TurnoVespertino, TurnoNoturno, TurnoIntegral:
			default:
				return false, ErrTurnoInvalido
			}
			if req.Turno != m.Turno {
				m.Turno = req.Turno
				updated = true
			}
		}
		if req.MunicipioTrabalhoID > 0 {
			if req.MunicipioTrabalhoID != m.MunicipioTrabalhoID {
				m.MunicipioTrabalhoID = req.MunicipioTrabalhoID
				updated = true
			}
		}
		if req.Residencia != nil {
			residencia := strings.TrimSpace(*req.Residencia)
			if residencia != m.Residencia {
				m.Residencia = residencia
				updated = true
			}
		}
		if req.Foto != nil {
			foto := strings.TrimSpace(*req.Foto)
			if foto != m.Foto {
				m.Foto = foto
				updated = true
			}
		}
		return updated, nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "motorista not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrNomeObrigatorio) ||
			errors.Is(err, ErrTurnoInvalido) ||
			errors.Is(err, ErrDataNascInvalida) ||
			errors.Is(err, validation.ErrNomeInvalido) ||
			errors.Is(err, validation.ErrTelefoneInvalido) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.Error("failed to update motorista", "error", err, "motoristaID", motoristaID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toMotoristaResponse(motorista))
}

func (h *MotoristaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	motoristaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(ctx, motoristaID); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "motorista not found", http.StatusNotFound)
			return
		}
		if db.IsAnyForeignKeyViolation(err) {
			http.Error(w, "motorista alocado em ciclos de viagem e não pode ser excluído", http.StatusConflict)
			return
		}
		slog.Error("failed to delete motorista", "error", err, "motoristaID", motoristaID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toMotoristaResponse(m *Motorista) MotoristaResponse {
	return MotoristaResponse{
		ID:                  m.ID,
		Nome:                m.Nome,
		CPF:                 m.CPF,
		Telefone:            m.Telefone,
		DataNasc:            m.DataNasc.Format("2006-01-02"),
		Turno:               m.Turno,
		MunicipioTrabalhoID: m.MunicipioTrabalhoID,
		Residencia:          m.Residencia,
		Foto:                m.Foto,
	}
}
