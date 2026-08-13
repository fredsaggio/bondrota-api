package reservas

import (
	"context"
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

type ReservaHandler struct {
	svc ReservaService
}

func NewReservaHandler(svc ReservaService) *ReservaHandler {
	return &ReservaHandler{svc: svc}
}

func (h *ReservaHandler) RequireOwnerOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsKey).(*auth.Claims)
		if !ok || claims.UserID <= 0 {
			http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
			return
		}
		if claims.Role == auth.RoleAdmin {
			next.ServeHTTP(w, r)
			return
		}
		if claims.Role != auth.RoleCliente {
			http.Error(w, "Você não tem permissão para executar esta ação.", http.StatusForbidden)
			return
		}

		reservaID, err := conv.ParseInt(r, "reservaID")
		if err != nil {
			http.Error(w, "Reserva não encontrada.", http.StatusBadRequest)
			return
		}
		reserva, err := h.svc.GetByID(r.Context(), reservaID)
		if err != nil {
			h.handleError(w, err, "failed to authorize reserva access")
			return
		}
		if reserva.ClienteID != claims.UserID {
			http.Error(w, "Você não tem permissão para executar esta ação.", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
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

type ReservaComNomesResponse struct {
	ReservaResponse
	ClienteNome string `json:"cliente_nome"`
	DestinoNome string `json:"destino_nome"`
}

type ReservaListResponse struct {
	Items      []ReservaComNomesResponse `json:"items"`
	NextCursor string                    `json:"next_cursor,omitempty"`
	HasMore    bool                      `json:"has_more"`
}

type ReservaResumoResponse struct {
	ConfirmadasTotal    int64            `json:"confirmadas_total"`
	ConfirmadasPorTurno map[string]int64 `json:"confirmadas_por_turno"`
}

type DisponibilidadeReservaResponse struct {
	DataViagem   string         `json:"data_viagem"`
	Turno        TurnoReserva   `json:"turno"`
	Sentido      SentidoReserva `json:"sentido"`
	PartidaEm    string         `json:"partida_em"`
	FechamentoEm string         `json:"fechamento_em"`
	ConsultadoEm string         `json:"consultado_em"`
	Disponivel   bool           `json:"disponivel"`
}

func (h *ReservaHandler) CreateByVinculo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, vinculoID, err := parseNestedVinculoIDs(r)
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	var req CreateReservaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
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

func (h *ReservaHandler) ConsultarDisponibilidade(w http.ResponseWriter, r *http.Request) {
	clienteID, vinculoID, err := parseNestedVinculoIDs(r)
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	dataViagem, err := parseReservaDate(r.URL.Query().Get("data_viagem"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	disponibilidade, err := h.svc.ConsultarDisponibilidade(r.Context(), DisponibilidadeReservaInput{
		ClienteID:  clienteID,
		VinculoID:  vinculoID,
		DataViagem: dataViagem,
		Turno:      TurnoReserva(r.URL.Query().Get("turno")),
		Sentido:    SentidoReserva(r.URL.Query().Get("sentido")),
	})
	if err != nil {
		h.handleError(w, err, "failed to check reservation availability")
		return
	}

	httputils.Respond(w, http.StatusOK, toDisponibilidadeReservaResponse(disponibilidade))
}

func (h *ReservaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reservaID, err := conv.ParseInt(r, "reservaID")
	if err != nil {
		http.Error(w, "Reserva não encontrada.", http.StatusBadRequest)
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

	params, err := parseReservaListParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.svc.List(ctx, params)
	if err != nil {
		slog.Error("failed to list reservas", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toReservaListResponse(result))
}

func (h *ReservaHandler) Resumo(w http.ResponseWriter, r *http.Request) {
	resumo, err := h.svc.Resumo(r.Context())
	if err != nil {
		slog.Error("failed to summarize reservas", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	porTurno := make(map[string]int64, len(resumo.ConfirmadasPorTurno))
	for turno, total := range resumo.ConfirmadasPorTurno {
		porTurno[string(turno)] = total
	}

	httputils.Respond(w, http.StatusOK, ReservaResumoResponse{
		ConfirmadasTotal:    resumo.ConfirmadasTotal,
		ConfirmadasPorTurno: porTurno,
	})
}

func parseReservaListParams(r *http.Request) (ReservaListParams, error) {
	query := r.URL.Query()
	params := ReservaListParams{Busca: query.Get("q")}

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit <= 0 {
			return ReservaListParams{}, errors.New("Parâmetro de listagem inválido.")
		}
		params.Limit = limit
	}

	if raw := query.Get("cursor"); raw != "" {
		cursor, err := decodeReservaCursor(raw)
		if err != nil {
			return ReservaListParams{}, errors.New("Parâmetro de listagem inválido.")
		}
		params.Cursor = cursor
	}

	if raw := query.Get("data_inicio"); raw != "" {
		data, err := parseReservaDate(raw)
		if err != nil {
			return ReservaListParams{}, errors.New("Data inicial inválida.")
		}
		params.DataInicio = &data
	}

	if raw := query.Get("data_fim"); raw != "" {
		data, err := parseReservaDate(raw)
		if err != nil {
			return ReservaListParams{}, errors.New("Data final inválida.")
		}
		params.DataFim = &data
	}

	return params, nil
}

// encodeReservaCursor/decodeReservaCursor tornam o cursor opaco para o consumidor:
// ele so precisa devolver o valor recebido, nunca monta um na mao. O formato
// interno (data|id) pode mudar sem quebrar contrato.
func encodeReservaCursor(cursor ReservaCursor) string {
	raw := cursor.DataViagem.Format("2006-01-02") + "|" + strconv.FormatInt(cursor.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeReservaCursor(value string) (*ReservaCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, errors.New("Parâmetro de listagem inválido.")
	}
	data, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		return nil, fmt.Errorf("cursor date: %w", err)
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cursor id: %w", err)
	}
	return &ReservaCursor{DataViagem: data, ID: id}, nil
}

func (h *ReservaHandler) ListByCliente(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "Cliente não encontrado.", http.StatusBadRequest)
		return
	}

	reservas, err := h.svc.ListByCliente(ctx, clienteID)
	if err != nil {
		slog.Error("failed to list reservas by cliente", "error", err, "clienteID", clienteID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toReservaResponses(reservas))
}

func (h *ReservaHandler) ListByVinculo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, vinculoID, err := parseNestedVinculoIDs(r)
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	reservas, err := h.svc.ListByVinculo(ctx, clienteID, vinculoID)
	if err != nil {
		if errors.Is(err, ErrVinculoNotFound) {
			http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
			return
		}
		slog.Error("failed to list reservas by vinculo", "error", err, "clienteID", clienteID, "vinculoID", vinculoID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toReservaResponses(reservas))
}

func (h *ReservaHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reservaID, err := conv.ParseInt(r, "reservaID")
	if err != nil {
		http.Error(w, "Reserva não encontrada.", http.StatusBadRequest)
		return
	}

	var req UpdateReservaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
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
		http.Error(w, "Reserva não encontrada.", http.StatusBadRequest)
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
		http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
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
		http.Error(w, "Você não tem permissão para executar esta ação.", http.StatusForbidden)
		return false
	}
	return true
}

func (h *ReservaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reservaID, err := conv.ParseInt(r, "reservaID")
	if err != nil {
		http.Error(w, "Reserva não encontrada.", http.StatusBadRequest)
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
		http.Error(w, "Reserva não encontrada.", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrVinculoNotFound) {
		http.Error(w, "Vínculo não encontrado.", http.StatusNotFound)
		return
	}
	if db.IsUniqueViolation(err, "uq_reservas_ativas_vinculo_data_turno_sentido") {
		http.Error(w, "Já existe uma reserva ativa para este vínculo nesta data, turno e sentido.", http.StatusConflict)
		return
	}
	if errors.Is(err, ErrPrazoReservaEncerrado) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if errors.Is(err, ErrDataObrigatoria) ||
		errors.Is(err, ErrDataInvalida) ||
		errors.Is(err, ErrSentidoInvalido) ||
		errors.Is(err, ErrStatusInvalido) ||
		errors.Is(err, ErrTurnoInvalido) ||
		errors.Is(err, ErrTurnoObrigatorio) ||
		errors.Is(err, ErrTurnoIncompativel) ||
		errors.Is(err, ErrVinculoIDObrigatorio) ||
		errors.Is(err, ErrHorarioNaoConfigurado) {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	slog.Error(msg, "error", err)
	http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
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

func toReservaListResponse(result ReservaListResult) ReservaListResponse {
	items := make([]ReservaComNomesResponse, 0, len(result.Items))
	for _, item := range result.Items {
		reserva := item.Reserva
		items = append(items, ReservaComNomesResponse{
			ReservaResponse: toReservaResponse(&reserva),
			ClienteNome:     item.ClienteNome,
			DestinoNome:     item.DestinoNome,
		})
	}

	resp := ReservaListResponse{Items: items, HasMore: result.HasMore}
	if result.NextCursor != nil {
		resp.NextCursor = encodeReservaCursor(*result.NextCursor)
	}
	return resp
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

func toDisponibilidadeReservaResponse(d *DisponibilidadeReserva) DisponibilidadeReservaResponse {
	return DisponibilidadeReservaResponse{
		DataViagem:   d.DataViagem.Format("2006-01-02"),
		Turno:        d.Turno,
		Sentido:      d.Sentido,
		PartidaEm:    d.PartidaEm.Format(time.RFC3339),
		FechamentoEm: d.FechamentoEm.Format(time.RFC3339),
		ConsultadoEm: d.ConsultadoEm.Format(time.RFC3339),
		Disponivel:   d.Disponivel,
	}
}
