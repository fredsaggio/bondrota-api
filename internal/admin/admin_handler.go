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
	ID string `json:"id"`
}

type AdminResponse struct {
	ID    string `json:"id"`
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
	UserID    string `json:"user_id"`
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
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	token, err := h.svc.Login(ctx, req.Email, req.Senha)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "E-mail ou senha inválidos.", http.StatusUnauthorized)
			return
		}
		slog.Error("failed to login admin", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	h.setSessionCookie(w, token)
	if strings.EqualFold(strings.TrimSpace(r.Header.Get(SessionModeHeader)), "cookie") {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	httputils.Respond(w, http.StatusOK, LoginResponse{Token: token})
}

type ChangePasswordRequest struct {
	SenhaAtual string `json:"senha_atual"`
	NovaSenha  string `json:"nova_senha"`
}

func (h *AdminHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := ctx.Value(auth.ClaimsKey).(*auth.Claims)
	if !ok || claims.UserID <= 0 {
		http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	token, err := h.svc.ChangePassword(ctx, claims.UserID, req.SenhaAtual, req.NovaSenha)
	if err != nil {
		switch {
		case errors.Is(err, ErrSenhaFraca):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, auth.ErrInvalidCredentials):
			// 403 e nao 401 de proposito: o painel derruba a sessao em qualquer 401,
			// entao errar a senha atual deslogaria quem so digitou errado.
			http.Error(w, "A senha atual está incorreta.", http.StatusForbidden)
		case errors.Is(err, ErrNotFound):
			http.Error(w, "Administrador não encontrado.", http.StatusNotFound)
		default:
			slog.Error("failed to change admin password", "error", err)
			http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		}
		return
	}

	// A sessao de quem trocou a senha continua de pe com um token novo. As demais
	// sessoes seguem valendo ate expirar: nao ha revogacao de JWT.
	h.setSessionCookie(w, token)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) Session(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsKey).(*auth.Claims)
	if !ok || claims.ExpiresAt == nil {
		http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
		return
	}

	httputils.Respond(w, http.StatusOK, SessionResponse{
		UserID:    claims.Subject,
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
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	admin, err := h.svc.Create(ctx, AdminInput{
		Email: req.Email,
		Senha: req.Senha,
	})
	if err != nil {
		slog.Error("failed to create admin", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusCreated, CreateAdminResponse{ID: admin.PublicID})
}

func (h *AdminHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adminID, err := conv.ParseInt(r, "adminID")
	if err != nil {
		http.Error(w, "Administrador não encontrado.", http.StatusBadRequest)
		return
	}

	var req UpdateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	admin, err := h.svc.Update(ctx, adminID, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			slog.Error("failed to update admin", "error", err)
			http.Error(w, "Administrador não encontrado.", http.StatusNotFound)
			return
		}
		slog.Error("failed to update admin", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, AdminResponse{
		ID:    admin.PublicID,
		Email: admin.Email,
	})
}

func (h *AdminHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adminID, err := conv.ParseInt(r, "adminID")
	if err != nil {
		http.Error(w, "Administrador não encontrado.", http.StatusBadRequest)
		return
	}

	admin, err := h.svc.GetByID(ctx, adminID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Administrador não encontrado.", http.StatusNotFound)
			return
		}
		slog.Error("failed to get admin", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, AdminResponse{
		ID:    admin.PublicID,
		Email: admin.Email,
	})
}

func (h *AdminHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adminID, err := conv.ParseInt(r, "adminID")
	if err != nil {
		http.Error(w, "Administrador não encontrado.", http.StatusBadRequest)
		return
	}

	err = h.svc.Delete(ctx, adminID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Administrador não encontrado.", http.StatusNotFound)
			return
		}
		slog.Error("failed to delete admin", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	admins, err := h.svc.List(ctx)
	if err != nil {
		slog.Error("failed to list admins", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	resp := make([]AdminResponse, 0, len(admins))
	for _, a := range admins {
		resp = append(resp, AdminResponse{
			ID:    a.PublicID,
			Email: a.Email,
		})
	}

	httputils.Respond(w, http.StatusOK, resp)
}
