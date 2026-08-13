package storage

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

type SignedUploadURLRequest struct {
	Bucket      string `json:"bucket"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	Upsert      bool   `json:"upsert"`
}

type SignedDownloadURLRequest struct {
	Bucket           string `json:"bucket"`
	Path             string `json:"path"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

func (h *Handler) CreateSignedUploadURL(w http.ResponseWriter, r *http.Request) {
	actor, err := actorFromRequest(r)
	if err != nil {
		http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
		return
	}

	var req SignedUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	signedURL, err := h.svc.CreateSignedUploadURL(r.Context(), actor, SignedUploadURLInput{
		Bucket:      req.Bucket,
		Path:        req.Path,
		ContentType: req.ContentType,
		Upsert:      req.Upsert,
	})
	if err != nil {
		h.handleError(w, err, "failed to create signed upload url")
		return
	}

	httputils.Respond(w, http.StatusCreated, signedURL)
}

func (h *Handler) CreateSignedDownloadURL(w http.ResponseWriter, r *http.Request) {
	actor, err := actorFromRequest(r)
	if err != nil {
		http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
		return
	}

	var req SignedDownloadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	signedURL, err := h.svc.CreateSignedDownloadURL(r.Context(), actor, SignedDownloadURLInput{
		Bucket:           req.Bucket,
		Path:             req.Path,
		ExpiresInSeconds: req.ExpiresInSeconds,
	})
	if err != nil {
		h.handleError(w, err, "failed to create signed download url")
		return
	}

	httputils.Respond(w, http.StatusOK, signedURL)
}

func (h *Handler) handleError(w http.ResponseWriter, err error, msg string) {
	if errors.Is(err, brerror.ErrUnauthenticated) {
		http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, brerror.ErrForbidden) {
		http.Error(w, "Você não tem permissão para executar esta ação.", http.StatusForbidden)
		return
	}
	if errors.Is(err, brerror.ErrInvalidInput) {
		http.Error(w, brerror.MensagemUsuario(err), http.StatusUnprocessableEntity)
		return
	}

	slog.Error(msg, "error", err)
	http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
}

func actorFromRequest(r *http.Request) (Actor, error) {
	claims, ok := r.Context().Value(auth.ClaimsKey).(*auth.Claims)
	if !ok || claims.UserID <= 0 {
		return Actor{}, brerror.ErrUnauthenticated
	}
	return Actor{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, nil
}
