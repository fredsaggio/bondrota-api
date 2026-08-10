package viagens

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type ViagemHandler struct {
	viagemSvc      ViagemService
	presencaSvc    PresencaService
	localizacaoSvc ViagemLocalizacaoService
}

func NewViagemHandler(viagemSvc ViagemService, presencaSvc PresencaService, localizacaoSvc ...ViagemLocalizacaoService) *ViagemHandler {
	var locSvc ViagemLocalizacaoService
	if len(localizacaoSvc) > 0 {
		locSvc = localizacaoSvc[0]
	}

	return &ViagemHandler{
		viagemSvc:      viagemSvc,
		presencaSvc:    presencaSvc,
		localizacaoSvc: locSvc,
	}
}

func (h *ViagemHandler) RequireAssignedMotoristaOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, err := actorFromRequest(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if actor.Role == auth.RoleAdmin {
			next.ServeHTTP(w, r)
			return
		}
		if actor.Role != auth.RoleMotorista {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		viagemID, err := conv.ParseInt(r, "viagemID")
		if err != nil {
			http.Error(w, "invalid viagem id", http.StatusBadRequest)
			return
		}
		viagem, err := h.viagemSvc.GetByID(r.Context(), viagemID)
		if err != nil {
			h.handleError(w, err, "failed to authorize viagem access")
			return
		}
		if viagem.Ciclo.MotoristaID != actor.UserID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type AtualizarPresencaRequest struct {
	StatusPresenca StatusPresencaViagem `json:"status_presenca"`
}

type AtualizarLocalizacaoRequest struct {
	MotoristaID    int64   `json:"motorista_id"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	VelocidadeKmh  float64 `json:"velocidade_kmh"`
	DirecaoGraus   float64 `json:"direcao_graus"`
	PrecisaoMetros float64 `json:"precisao_metros"`
}

type ViagemResponse struct {
	ID            int64         `json:"id"`
	CicloViagemID int64         `json:"ciclo_viagem_id"`
	Sentido       SentidoViagem `json:"sentido"`
	Status        StatusViagem  `json:"status"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
}

type CicloViagemResponse struct {
	ID                 int64             `json:"id"`
	DataViagem         string            `json:"data_viagem"`
	Turno              TurnoViagem       `json:"turno"`
	MunicipioDestinoID int64             `json:"municipio_destino_id"`
	RotaInternaID      int64             `json:"rota_interna_id"`
	VeiculoID          int64             `json:"veiculo_id"`
	MotoristaID        int64             `json:"motorista_id"`
	Status             StatusCicloViagem `json:"status"`
	ExpiresAt          string            `json:"expires_at"`
	CreatedAt          string            `json:"created_at"`
	UpdatedAt          string            `json:"updated_at"`
}

type ViagemComCicloResponse struct {
	Viagem ViagemResponse      `json:"viagem"`
	Ciclo  CicloViagemResponse `json:"ciclo"`
}

type ViagemHorarioResponse struct {
	ID        int64             `json:"id"`
	ViagemID  int64             `json:"viagem_id"`
	Tipo      TipoHorarioViagem `json:"tipo"`
	Horario   string            `json:"horario"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

type ViagemReservaResponse struct {
	ID             int64                `json:"id"`
	ViagemID       int64                `json:"viagem_id"`
	ReservaID      int64                `json:"reserva_id"`
	StatusPresenca StatusPresencaViagem `json:"status_presenca"`
	CreatedAt      string               `json:"created_at"`
	UpdatedAt      string               `json:"updated_at"`
}

type ViagemReservaComReservaResponse struct {
	ViagemReservaResponse
	ClienteID     int64         `json:"cliente_id"`
	VinculoID     int64         `json:"vinculo_id"`
	DataViagem    string        `json:"data_viagem"`
	Turno         TurnoViagem   `json:"turno"`
	DestinoID     int64         `json:"destino_id"`
	RotaInternaID int64         `json:"rota_interna_id"`
	Sentido       SentidoViagem `json:"sentido"`
}

type ViagemLocalizacaoResponse struct {
	ViagemID       int64   `json:"viagem_id"`
	MotoristaID    int64   `json:"motorista_id"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	VelocidadeKmh  float64 `json:"velocidade_kmh"`
	DirecaoGraus   float64 `json:"direcao_graus"`
	PrecisaoMetros float64 `json:"precisao_metros"`
	RegistradaEm   string  `json:"registrada_em"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func (h *ViagemHandler) List(w http.ResponseWriter, r *http.Request) {
	viagens, err := h.viagemSvc.List(r.Context())
	if err != nil {
		slog.Error("failed to list viagens", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if claims, ok := r.Context().Value(auth.ClaimsKey).(*auth.Claims); ok && claims.Role == auth.RoleMotorista {
		assigned := make([]ViagemComCiclo, 0, len(viagens))
		for _, viagem := range viagens {
			if viagem.Ciclo.MotoristaID == claims.UserID {
				assigned = append(assigned, viagem)
			}
		}
		viagens = assigned
	}

	httputils.Respond(w, http.StatusOK, toViagemComCicloResponses(viagens))
}

func (h *ViagemHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	viagem, err := h.viagemSvc.GetByID(r.Context(), viagemID)
	if err != nil {
		h.handleError(w, err, "failed to get viagem")
		return
	}

	httputils.Respond(w, http.StatusOK, toViagemComCicloResponse(viagem))
}

func (h *ViagemHandler) Iniciar(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	viagem, err := h.viagemSvc.Iniciar(r.Context(), viagemID)
	if err != nil {
		h.handleError(w, err, "failed to start viagem")
		return
	}

	httputils.Respond(w, http.StatusOK, toViagemResponse(viagem))
}

func (h *ViagemHandler) Concluir(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	viagem, err := h.viagemSvc.Concluir(r.Context(), viagemID)
	if err != nil {
		h.handleError(w, err, "failed to finish viagem")
		return
	}

	httputils.Respond(w, http.StatusOK, toViagemResponse(viagem))
}

func (h *ViagemHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	viagem, err := h.viagemSvc.Cancelar(r.Context(), viagemID)
	if err != nil {
		h.handleError(w, err, "failed to cancel viagem")
		return
	}

	httputils.Respond(w, http.StatusOK, toViagemResponse(viagem))
}

func (h *ViagemHandler) ListHorarios(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	horarios, err := h.viagemSvc.ListHorariosByViagem(r.Context(), viagemID)
	if err != nil {
		h.handleError(w, err, "failed to list viagem horarios")
		return
	}

	httputils.Respond(w, http.StatusOK, toViagemHorarioResponses(horarios))
}

func (h *ViagemHandler) ListReservas(w http.ResponseWriter, r *http.Request) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	reservas, err := h.presencaSvc.ListReservasByViagem(r.Context(), viagemID)
	if err != nil {
		h.handleError(w, err, "failed to list reservas by viagem")
		return
	}

	httputils.Respond(w, http.StatusOK, toViagemReservaComReservaResponses(reservas))
}

func (h *ViagemHandler) AtualizarPresenca(w http.ResponseWriter, r *http.Request) {
	viagemID, reservaID, err := parseViagemReservaIDs(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req AtualizarPresencaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	viagemReserva, err := h.presencaSvc.AtualizarPresenca(r.Context(), viagemID, reservaID, req.StatusPresenca)
	if err != nil {
		h.handleError(w, err, "failed to update presenca")
		return
	}

	httputils.Respond(w, http.StatusOK, toViagemReservaResponse(viagemReserva))
}

func (h *ViagemHandler) AtualizarLocalizacao(w http.ResponseWriter, r *http.Request) {
	if h.localizacaoSvc == nil {
		http.Error(w, "localizacao service not configured", http.StatusInternalServerError)
		return
	}

	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	actor, err := actorFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req AtualizarLocalizacaoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	localizacao, err := h.localizacaoSvc.Atualizar(r.Context(), actor, ViagemLocalizacaoInput{
		ViagemID:       viagemID,
		MotoristaID:    req.MotoristaID,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		VelocidadeKmh:  req.VelocidadeKmh,
		DirecaoGraus:   req.DirecaoGraus,
		PrecisaoMetros: req.PrecisaoMetros,
	})
	if err != nil {
		h.handleError(w, err, "failed to update viagem localizacao")
		return
	}

	httputils.Respond(w, http.StatusOK, toViagemLocalizacaoResponse(localizacao))
}

func (h *ViagemHandler) GetLocalizacao(w http.ResponseWriter, r *http.Request) {
	if h.localizacaoSvc == nil {
		http.Error(w, "localizacao service not configured", http.StatusInternalServerError)
		return
	}

	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		http.Error(w, "invalid viagem id", http.StatusBadRequest)
		return
	}

	actor, err := actorFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	localizacao, err := h.localizacaoSvc.GetByViagem(r.Context(), actor, viagemID)
	if err != nil {
		h.handleError(w, err, "failed to get viagem localizacao")
		return
	}

	httputils.Respond(w, http.StatusOK, toViagemLocalizacaoResponse(localizacao))
}

func (h *ViagemHandler) handleError(w http.ResponseWriter, err error, msg string) {
	if errors.Is(err, brerror.ErrUnauthenticated) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, brerror.ErrForbidden) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if errors.Is(err, brerror.ErrInvalidInput) {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if errors.Is(err, brerror.ErrNotFound) {
		http.Error(w, "resource not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, brerror.ErrAlreadyExists) {
		http.Error(w, "resource already exists", http.StatusConflict)
		return
	}

	slog.Error(msg, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func parseViagemReservaIDs(r *http.Request) (int64, int64, error) {
	viagemID, err := conv.ParseInt(r, "viagemID")
	if err != nil {
		return 0, 0, err
	}

	reservaID, err := conv.ParseInt(r, "reservaID")
	if err != nil {
		return 0, 0, err
	}

	return viagemID, reservaID, nil
}

func actorFromRequest(r *http.Request) (ViagemLocalizacaoActor, error) {
	claims, ok := r.Context().Value(auth.ClaimsKey).(*auth.Claims)
	if !ok || claims.UserID <= 0 {
		return ViagemLocalizacaoActor{}, brerror.ErrUnauthenticated
	}

	return ViagemLocalizacaoActor{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, nil
}

func toViagemComCicloResponses(viagens []ViagemComCiclo) []ViagemComCicloResponse {
	resp := make([]ViagemComCicloResponse, 0, len(viagens))
	for _, viagem := range viagens {
		resp = append(resp, toViagemComCicloResponse(&viagem))
	}
	return resp
}

func toViagemComCicloResponse(v *ViagemComCiclo) ViagemComCicloResponse {
	return ViagemComCicloResponse{
		Viagem: toViagemResponse(&v.Viagem),
		Ciclo:  toCicloViagemResponse(&v.Ciclo),
	}
}

func toViagemResponse(v *Viagem) ViagemResponse {
	return ViagemResponse{
		ID:            v.ID,
		CicloViagemID: v.CicloViagemID,
		Sentido:       v.Sentido,
		Status:        v.Status,
		CreatedAt:     v.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     v.UpdatedAt.Format(time.RFC3339),
	}
}

func toCicloViagemResponse(c *CicloViagem) CicloViagemResponse {
	return CicloViagemResponse{
		ID:                 c.ID,
		DataViagem:         c.DataViagem.Format("2006-01-02"),
		Turno:              c.Turno,
		MunicipioDestinoID: c.MunicipioDestinoID,
		RotaInternaID:      c.RotaInternaID,
		VeiculoID:          c.VeiculoID,
		MotoristaID:        c.MotoristaID,
		Status:             c.Status,
		ExpiresAt:          c.ExpiresAt.Format(time.RFC3339),
		CreatedAt:          c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          c.UpdatedAt.Format(time.RFC3339),
	}
}

func toViagemHorarioResponses(horarios []ViagemHorario) []ViagemHorarioResponse {
	resp := make([]ViagemHorarioResponse, 0, len(horarios))
	for _, horario := range horarios {
		resp = append(resp, toViagemHorarioResponse(&horario))
	}
	return resp
}

func toViagemHorarioResponse(h *ViagemHorario) ViagemHorarioResponse {
	return ViagemHorarioResponse{
		ID:        h.ID,
		ViagemID:  h.ViagemID,
		Tipo:      h.Tipo,
		Horario:   h.Horario.Format(time.RFC3339),
		CreatedAt: h.CreatedAt.Format(time.RFC3339),
		UpdatedAt: h.UpdatedAt.Format(time.RFC3339),
	}
}

func toViagemReservaComReservaResponses(reservas []ViagemReservaComReserva) []ViagemReservaComReservaResponse {
	resp := make([]ViagemReservaComReservaResponse, 0, len(reservas))
	for _, reserva := range reservas {
		resp = append(resp, toViagemReservaComReservaResponse(&reserva))
	}
	return resp
}

func toViagemReservaComReservaResponse(vr *ViagemReservaComReserva) ViagemReservaComReservaResponse {
	return ViagemReservaComReservaResponse{
		ViagemReservaResponse: toViagemReservaResponse(&vr.ViagemReserva),
		ClienteID:             vr.ClienteID,
		VinculoID:             vr.VinculoID,
		DataViagem:            vr.DataViagem.Format("2006-01-02"),
		Turno:                 vr.Turno,
		DestinoID:             vr.DestinoID,
		RotaInternaID:         vr.RotaInternaID,
		Sentido:               vr.Sentido,
	}
}

func toViagemReservaResponse(vr *ViagemReserva) ViagemReservaResponse {
	return ViagemReservaResponse{
		ID:             vr.ID,
		ViagemID:       vr.ViagemID,
		ReservaID:      vr.ReservaID,
		StatusPresenca: vr.StatusPresenca,
		CreatedAt:      vr.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      vr.UpdatedAt.Format(time.RFC3339),
	}
}

func toViagemLocalizacaoResponse(l *ViagemLocalizacao) ViagemLocalizacaoResponse {
	return ViagemLocalizacaoResponse{
		ViagemID:       l.ViagemID,
		MotoristaID:    l.MotoristaID,
		Latitude:       l.Latitude,
		Longitude:      l.Longitude,
		VelocidadeKmh:  l.VelocidadeKmh,
		DirecaoGraus:   l.DirecaoGraus,
		PrecisaoMetros: l.PrecisaoMetros,
		RegistradaEm:   l.RegistradaEm.Format(time.RFC3339),
		CreatedAt:      l.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      l.UpdatedAt.Format(time.RFC3339),
	}
}
