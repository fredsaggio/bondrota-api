package clientes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type VinculoHandler struct {
	svc      VinculoService
	arquivos ArquivoMovedor
}

// NewVinculoHandler aceita o movedor de arquivos como variadico pelo mesmo
// motivo de NewClienteHandler: a maioria dos testes deste pacote nem chega a
// exercitar o campo comprovante, e sem o argumento ele so fica no caminho que
// veio na requisicao — igual ao comportamento de antes desta funcionalidade
// existir.
func NewVinculoHandler(svc VinculoService, arquivos ...ArquivoMovedor) *VinculoHandler {
	h := &VinculoHandler{svc: svc}
	if len(arquivos) > 0 {
		h.arquivos = arquivos[0]
	}
	return h
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
	DestinoNome string `json:"destino_nome"`
}

type VinculoListResponse struct {
	Items      []VinculoComClienteResponse `json:"items"`
	NextCursor string                      `json:"next_cursor,omitempty"`
	HasMore    bool                        `json:"has_more"`
}

func (h *VinculoHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "Cliente não encontrado.", http.StatusBadRequest)
		return
	}

	var req VinculoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
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

	if input.Comprovante != "" {
		vinculo = h.organizarComprovante(ctx, vinculo, input)
	}

	httputils.Respond(w, http.StatusCreated, toVinculoResponse(vinculo))
}

// organizarComprovante leva o comprovante da pasta de espera (onde o admin
// envia antes do vinculo ter ID) para o caminho definitivo
// "clientes/{cliente_id}/vinculos/{vinculo_id}/comprovante-{tipo}{ext}".
// Best-effort: se o Storage falhar aqui, o vinculo ja foi criado com sucesso —
// falhar a resposta agora deixaria o admin sem saber se o cadastro existe. O
// comprovante so fica no caminho de espera nesse caso raro, em vez de ficar
// organizado.
//
// Reaproveita VinculoService.Update em vez de um metodo novo: os dois campos
// obrigatorios de VinculoInput e VinculoUpdateInput sao os mesmos, entao dá
// pra montar o update com os valores que acabaram de criar o vinculo, so
// trocando o Comprovante — sem precisar de um metodo novo no Store nem mexer
// nos mocks gerados.
func (h *VinculoHandler) organizarComprovante(ctx context.Context, vinculo *Vinculo, input VinculoInput) *Vinculo {
	if h.arquivos == nil {
		return vinculo
	}
	destino := fmt.Sprintf(
		"clientes/%s/vinculos/%s/comprovante-%s%s",
		strconv.FormatInt(input.ClienteID, 10),
		strconv.FormatInt(vinculo.ID, 10),
		input.Tipo,
		path.Ext(input.Comprovante),
	)
	if err := h.arquivos.MoveObject(ctx, "documentos", input.Comprovante, destino); err != nil {
		slog.Error("failed to organize vinculo comprovante", "error", err, "vinculoID", vinculo.ID)
		return vinculo
	}
	atualizado, err := h.svc.Update(ctx, vinculo.ID, VinculoUpdateInput{
		Tipo:          input.Tipo,
		Turno:         input.Turno,
		DestinoID:     input.DestinoID,
		RotaInternaID: input.RotaInternaID,
		Curso:         input.Curso,
		Comprovante:   destino,
		Validade:      input.Validade,
		HorariosFixos: input.HorariosFixos,
	})
	if err != nil {
		slog.Error("failed to persist organized vinculo comprovante", "error", err, "vinculoID", vinculo.ID)
		return vinculo
	}
	return atualizado
}

func (h *VinculoHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	params, err := parseVinculoListParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.svc.List(ctx, params)
	if err != nil {
		slog.Error("failed to list vinculos", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	items := make([]VinculoComClienteResponse, 0, len(result.Items))
	for _, v := range result.Items {
		items = append(items, VinculoComClienteResponse{
			VinculoResponse: toVinculoResponse(&v.Vinculo),
			ClienteNome:     v.ClienteNome,
			DestinoNome:     v.DestinoNome,
		})
	}

	resp := VinculoListResponse{Items: items, HasMore: result.HasMore}
	if result.NextCursor != nil {
		resp.NextCursor = encodeVinculoCursor(*result.NextCursor)
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func parseVinculoListParams(r *http.Request) (VinculoListParams, error) {
	query := r.URL.Query()
	params := VinculoListParams{Busca: query.Get("q")}

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit <= 0 {
			return VinculoListParams{}, errors.New("Parâmetro de listagem inválido.")
		}
		params.Limit = limit
	}

	if raw := query.Get("cursor"); raw != "" {
		cursor, err := decodeVinculoCursor(raw)
		if err != nil {
			return VinculoListParams{}, errors.New("Parâmetro de listagem inválido.")
		}
		params.Cursor = cursor
	}

	return params, nil
}

// O cursor carrega nome e id. O nome pode conter qualquer caractere, inclusive o
// separador, entao a decodificacao corta no ultimo "|" — o id nunca tem um.
func encodeVinculoCursor(cursor VinculoCursor) string {
	raw := cursor.ClienteNome + "|" + strconv.FormatInt(cursor.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeVinculoCursor(value string) (*VinculoCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}
	decoded := string(raw)
	sep := strings.LastIndex(decoded, "|")
	if sep < 0 {
		return nil, errors.New("Parâmetro de listagem inválido.")
	}
	id, err := strconv.ParseInt(decoded[sep+1:], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cursor id: %w", err)
	}
	return &VinculoCursor{ClienteNome: decoded[:sep], ID: id}, nil
}

func (h *VinculoHandler) ListByCliente(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "Cliente não encontrado.", http.StatusBadRequest)
		return
	}

	vinculos, err := h.svc.ListByCliente(ctx, clienteID)
	if err != nil {
		slog.Error("failed to list vinculos", "error", err, "clienteID", clienteID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
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
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	vinculo, err := h.svc.GetByID(ctx, vinculoID)
	if err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
			return
		}
		slog.Error("failed to get vinculo", "error", err, "vinculoID", vinculoID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}
	if vinculo.ClienteID != clienteID {
		http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
		return
	}

	httputils.Respond(w, http.StatusOK, toVinculoResponse(vinculo))
}

func (h *VinculoHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, vinculoID, err := parseNestedVinculoIDs(r)
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	vinculo, err := h.svc.GetByID(ctx, vinculoID)
	if err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
			return
		}
		slog.Error("failed to get vinculo", "error", err, "vinculoID", vinculoID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}
	if vinculo.ClienteID != clienteID {
		http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
		return
	}

	var req VinculoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
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
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	vinculo, err := h.svc.GetByID(ctx, vinculoID)
	if err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
			return
		}
		slog.Error("failed to get vinculo", "error", err, "vinculoID", vinculoID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}
	if vinculo.ClienteID != clienteID {
		http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
		return
	}

	if err := h.svc.Delete(ctx, vinculoID); err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
			return
		}
		if db.IsAnyForeignKeyViolation(err) {
			http.Error(w, "Este vínculo tem reservas registradas e não pode ser removido.", http.StatusConflict)
			return
		}
		slog.Error("failed to delete vinculo", "error", err, "vinculoID", vinculoID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
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
		http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
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
		http.Error(w, "Cliente não encontrado.", http.StatusNotFound)
		return
	}
	if db.IsForeignKeyViolation(err, "cliente_vinculos_destino_id_fkey") {
		http.Error(w, "Destino não encontrado.", http.StatusUnprocessableEntity)
		return
	}
	if db.IsForeignKeyViolation(err, "cliente_vinculos_rota_interna_id_fkey") {
		http.Error(w, "Rota interna não encontrada.", http.StatusUnprocessableEntity)
		return
	}
	slog.Error(msg, "error", err)
	http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
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
		// Maiuscula por consistencia, como o nome — evita que "Ciência da
		// Computação" e "ciência da computação" virem dois valores diferentes
		// na coluna, o que atrapalharia agrupar/contar por curso depois.
		Curso:         strings.ToUpper(strings.TrimSpace(req.Curso)),
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
		// Maiuscula por consistencia, como o nome — evita que "Ciência da
		// Computação" e "ciência da computação" virem dois valores diferentes
		// na coluna, o que atrapalharia agrupar/contar por curso depois.
		Curso:         strings.ToUpper(strings.TrimSpace(req.Curso)),
		Comprovante:   strings.TrimSpace(req.Comprovante),
		Validade:      validade,
		HorariosFixos: req.HorariosFixos,
	}, nil
}

func validateVinculoRequest(req VinculoRequest) (time.Time, error) {
	if req.DestinoID <= 0 {
		return time.Time{}, errors.New("Selecione o destino.")
	}
	if req.RotaInternaID <= 0 {
		return time.Time{}, errors.New("Selecione a rota interna.")
	}
	if req.Validade == "" {
		return time.Time{}, errors.New("Informe a data de validade.")
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
