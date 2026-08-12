package clientes

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type ClienteHandler struct {
	clienteSvc ClienteService
}

func NewClienteHandler(clienteSvc ClienteService) *ClienteHandler {
	return &ClienteHandler{
		clienteSvc: clienteSvc,
	}
}

type CreateClienteRequest struct {
	Nome     string `json:"nome"`
	CPF      string `json:"cpf"`
	Senha    string `json:"senha"`
	Telefone string `json:"telefone"`
	DataNasc string `json:"data_nasc"`
	Foto     string `json:"foto"`
}

type UpdateClienteRequest struct {
	Nome string `json:"nome"`
	// Telefone e Foto sao ponteiros porque sao opcionais e podem ser legitimamente
	// limpos: chave ausente/null preserva o valor atual, string vazia explicita
	// apaga o campo. Nome e DataNasc sao obrigatorios e nunca fazem sentido em
	// branco, entao continuam string simples.
	Telefone *string `json:"telefone"`
	DataNasc string  `json:"data_nasc"`
	Foto     *string `json:"foto"`
}

type ClienteResponse struct {
	ID       int64  `json:"id"`
	Nome     string `json:"nome"`
	CPF      string `json:"cpf"`
	Telefone string `json:"telefone"`
	DataNasc string `json:"data_nasc"`
	Foto     string `json:"foto"`
}

type ClienteListResponse struct {
	Items      []ClienteResponse `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

type ClienteResumoResponse struct {
	Total int64 `json:"total"`
}

type ClienteComVinculosResponse struct {
	ID       int64             `json:"id"`
	Nome     string            `json:"nome"`
	CPF      string            `json:"cpf"`
	Telefone string            `json:"telefone"`
	DataNasc string            `json:"data_nasc"`
	Foto     string            `json:"foto"`
	Vinculos []VinculoResponse `json:"vinculos"`
}

type LoginRequest struct {
	CPF   string `json:"cpf"`
	Senha string `json:"senha"`
}

func (h *ClienteHandler) Login(w http.ResponseWriter, r *http.Request) {
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

	token, err := h.clienteSvc.Login(ctx, strings.TrimSpace(req.CPF), req.Senha)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, ErrNotFound) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		slog.Error("failed to login cliente", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, map[string]string{"token": token})
}

func (h *ClienteHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateClienteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input, err := toClienteInput(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cliente, err := h.clienteSvc.Create(ctx, input)
	if err != nil {
		if db.IsUniqueViolation(err, "clientes_cpf_key") {
			http.Error(w, "cpf already exists", http.StatusConflict)
			return
		}
		slog.Error("failed to create cliente", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, toClienteResponse(cliente))
}

func (h *ClienteHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	cliente, err := h.clienteSvc.GetByID(ctx, clienteID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "cliente not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get cliente", "error", err, "clienteID", clienteID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toClienteComVinculosResponse(cliente))
}

func (h *ClienteHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	params, err := parseClienteListParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.clienteSvc.List(ctx, params)
	if err != nil {
		slog.Error("failed to list clientes", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	items := make([]ClienteResponse, 0, len(result.Items))
	for _, c := range result.Items {
		items = append(items, toClienteResponse(&c))
	}

	resp := ClienteListResponse{Items: items, HasMore: result.HasMore}
	if result.NextCursorID > 0 {
		resp.NextCursor = encodeClienteCursor(result.NextCursorID)
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *ClienteHandler) Resumo(w http.ResponseWriter, r *http.Request) {
	resumo, err := h.clienteSvc.Resumo(r.Context())
	if err != nil {
		slog.Error("failed to summarize clientes", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, ClienteResumoResponse{Total: resumo.Total})
}

func parseClienteListParams(r *http.Request) (ClienteListParams, error) {
	query := r.URL.Query()
	params := ClienteListParams{Busca: query.Get("q")}

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit <= 0 {
			return ClienteListParams{}, errors.New("invalid limit")
		}
		params.Limit = limit
	}

	if raw := query.Get("cursor"); raw != "" {
		cursorID, err := decodeClienteCursor(raw)
		if err != nil {
			return ClienteListParams{}, errors.New("invalid cursor")
		}
		params.CursorID = cursorID
	}

	return params, nil
}

// O cursor e opaco para o consumidor, mesmo carregando so o id: o formato pode
// mudar (para incluir outra chave de ordenacao) sem quebrar contrato.
func encodeClienteCursor(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

func decodeClienteCursor(value string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return 0, fmt.Errorf("decode cursor: %w", err)
	}
	id, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("malformed cursor")
	}
	return id, nil
}

func (h *ClienteHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req UpdateClienteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	cliente, err := h.clienteSvc.Update(ctx, clienteID, func(c *Cliente) (bool, error) {
		updated := false
		if req.Nome != "" {
			nome := strings.TrimSpace(req.Nome)
			if nome == "" {
				return false, ErrNomeObrigatorio
			}
			if nome != c.Nome {
				c.Nome = nome
				updated = true
			}
		}
		if req.Telefone != nil {
			telefone := strings.TrimSpace(*req.Telefone)
			if telefone != c.Telefone {
				c.Telefone = telefone
				updated = true
			}
		}
		if req.DataNasc != "" {
			dataNasc, err := parseDate(req.DataNasc)
			if err != nil {
				return false, ErrDataInvalida
			}
			if !dataNasc.Equal(c.DataNasc) {
				c.DataNasc = dataNasc
				updated = true
			}
		}
		if req.Foto != nil {
			foto := strings.TrimSpace(*req.Foto)
			if foto != c.Foto {
				c.Foto = foto
				updated = true
			}
		}
		return updated, nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "cliente not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrNomeObrigatorio) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrDataInvalida) {
			http.Error(w, "data_nasc must be in format YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		slog.Error("failed to update cliente", "error", err, "clienteID", clienteID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toClienteResponse(cliente))
}

func (h *ClienteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.clienteSvc.Delete(ctx, clienteID); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "cliente not found", http.StatusNotFound)
			return
		}
		// Vinculos e reservas do cliente somem em cascata, mas uma reserva ja alocada
		// a uma viagem e protegida por RESTRICT em viagem_reservas.
		if db.IsAnyForeignKeyViolation(err) {
			http.Error(w, "cliente possui reservas alocadas a viagens e não pode ser excluído", http.StatusConflict)
			return
		}
		slog.Error("failed to delete cliente", "error", err, "clienteID", clienteID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toClienteInput(req CreateClienteRequest) (ClienteInput, error) {
	if strings.TrimSpace(req.Nome) == "" {
		return ClienteInput{}, errors.New("nome is required")
	}
	if strings.TrimSpace(req.CPF) == "" {
		return ClienteInput{}, errors.New("cpf is required")
	}
	if strings.TrimSpace(req.Senha) == "" {
		return ClienteInput{}, errors.New("senha is required")
	}
	if req.DataNasc == "" {
		return ClienteInput{}, errors.New("data_nasc is required")
	}

	dataNasc, err := parseDate(req.DataNasc)
	if err != nil {
		return ClienteInput{}, errors.New("data_nasc must be in format YYYY-MM-DD")
	}

	return ClienteInput{
		Nome:     strings.TrimSpace(req.Nome),
		CPF:      strings.TrimSpace(req.CPF),
		Senha:    req.Senha,
		Telefone: strings.TrimSpace(req.Telefone),
		DataNasc: dataNasc,
		Foto:     strings.TrimSpace(req.Foto),
	}, nil
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func toClienteResponse(c *Cliente) ClienteResponse {
	return ClienteResponse{
		ID:       c.ID,
		Nome:     c.Nome,
		CPF:      c.CPF,
		Telefone: c.Telefone,
		DataNasc: c.DataNasc.Format("2006-01-02"),
		Foto:     c.Foto,
	}
}

func toClienteComVinculosResponse(c *ClienteComVinculos) ClienteComVinculosResponse {
	vinculos := make([]VinculoResponse, 0, len(c.Vinculos))
	for _, v := range c.Vinculos {
		vinculos = append(vinculos, toVinculoResponse(&v))
	}

	return ClienteComVinculosResponse{
		ID:       c.ID,
		Nome:     c.Nome,
		CPF:      c.CPF,
		Telefone: c.Telefone,
		DataNasc: c.DataNasc.Format("2006-01-02"),
		Foto:     c.Foto,
		Vinculos: vinculos,
	}
}
