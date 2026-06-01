package admin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/auth"
)

type AdminHandler struct {
	s       AdminStore
	authSvc *auth.AuthService
}

type CreateAdminRequest struct {
	Email  string `json:"email"`
	Senha  string `json:"senha"`
	Cidade string `json:"cidade"`
}

type AdminResponse struct {
	ID     int64  `json:"id"`
	Email  string `json:"email"`
	Cidade string `json:"cidade"`
}

type UpdateAdminRequest struct {
	Email  string `json:"email"`
	Cidade string `json:"cidade"`
}

type LoginRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func NewAdminHandler(store AdminStore) *AdminHandler {
	return &AdminHandler{s: store}
}

func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	admin, err := h.s.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	ok, err := h.authSvc.CheckPassword(admin.Senha, req.Senha)
	if err != nil || !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.authSvc.GenerateToken(admin.ID, "admin")
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, LoginResponse{Token: token})
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}