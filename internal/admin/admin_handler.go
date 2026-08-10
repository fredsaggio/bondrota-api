package admin

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type AdminHandler struct {
	svc          AdminService
	cookieConfig SessionCookieConfig
}

const DefaultSessionCookieName = "bondrota_admin_session"
const SessionModeHeader = "X-Admin-Session-Mode"

type SessionCookieConfig struct {
	Name     string
	Path     string
	Domain   string
	Secure   bool
	SameSite http.SameSite
	TTL      time.Duration
}

func NewAdminHandler(svc AdminService, configs ...SessionCookieConfig) *AdminHandler {
	config := SessionCookieConfig{}
	if len(configs) > 0 {
		config = configs[0]
	}
	if config.Name == "" {
		config.Name = DefaultSessionCookieName
	}
	if config.Path == "" {
		config.Path = "/api/v1"
	}
	if config.SameSite == 0 {
		config.SameSite = http.SameSiteLaxMode
	}
	if config.TTL <= 0 {
		config.TTL = auth.TokenTTL
	}
	return &AdminHandler{svc: svc, cookieConfig: config}
}

type CreateAdminRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type CreateAdminResponse struct {
	ID int64 `json:"id"`
}

type AdminResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type UpdateAdminRequest struct {
	Email string `json:"email"`
}

type LoginRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type SessionResponse struct {
	UserID    int64  `json:"user_id"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expires_at"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.svc.Login(ctx, req.Email, req.Senha)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		slog.Error("failed to login admin", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.setSessionCookie(w, token)
	if strings.EqualFold(strings.TrimSpace(r.Header.Get(SessionModeHeader)), "cookie") {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	httputils.Respond(w, http.StatusOK, LoginResponse{Token: token})
}

func (h *AdminHandler) Session(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsKey).(*auth.Claims)
	if !ok || claims.ExpiresAt == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	httputils.Respond(w, http.StatusOK, SessionResponse{
		UserID:    claims.UserID,
		Role:      claims.Role,
		ExpiresAt: claims.ExpiresAt.Time.UnixMilli(),
	})
}

func (h *AdminHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	cookie := h.sessionCookie("")
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(1, 0)
	http.SetCookie(w, cookie)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) setSessionCookie(w http.ResponseWriter, token string) {
	cookie := h.sessionCookie(token)
	cookie.MaxAge = int(h.cookieConfig.TTL.Seconds())
	cookie.Expires = time.Now().Add(h.cookieConfig.TTL)
	http.SetCookie(w, cookie)
}

func (h *AdminHandler) sessionCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     h.cookieConfig.Name,
		Value:    value,
		Path:     h.cookieConfig.Path,
		Domain:   h.cookieConfig.Domain,
		HttpOnly: true,
		Secure:   h.cookieConfig.Secure,
		SameSite: h.cookieConfig.SameSite,
	}
}

func (h *AdminHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	admin, err := h.svc.Create(ctx, AdminInput{
		Email: req.Email,
		Senha: req.Senha,
	})
	if err != nil {
		slog.Error("failed to create admin", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, CreateAdminResponse{ID: admin.ID})
}

func (h *AdminHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adminID, err := conv.ParseInt(r, "adminID")
	if err != nil {
		http.Error(w, "invalid admin id", http.StatusBadRequest)
		return
	}

	var req UpdateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	admin, err := h.svc.Update(ctx, adminID, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			slog.Error("failed to update admin", "error", err)
			http.Error(w, "admin not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to update admin", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, AdminResponse{
		ID:    admin.ID,
		Email: admin.Email,
	})
}

func (h *AdminHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adminID, err := conv.ParseInt(r, "adminID")
	if err != nil {
		http.Error(w, "invalid admin id", http.StatusBadRequest)
		return
	}

	admin, err := h.svc.GetByID(ctx, adminID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "admin not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get admin", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, AdminResponse{
		ID:    admin.ID,
		Email: admin.Email,
	})
}

func (h *AdminHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adminID, err := conv.ParseInt(r, "adminID")
	if err != nil {
		http.Error(w, "invalid admin id", http.StatusBadRequest)
		return
	}

	err = h.svc.Delete(ctx, adminID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "admin not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to delete admin", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	admins, err := h.svc.List(ctx)
	if err != nil {
		slog.Error("failed to list admins", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]AdminResponse, 0, len(admins))
	for _, a := range admins {
		resp = append(resp, AdminResponse{
			ID:    a.ID,
			Email: a.Email,
		})
	}

	httputils.Respond(w, http.StatusOK, resp)
}
