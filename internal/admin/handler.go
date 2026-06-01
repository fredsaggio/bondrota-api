package admin

import "github.com/fredsaggio/bondrota-api/internal/auth"

type AdminHandler struct {
	s           AdminStore
	authSvc *auth.AuthService
}

type CreateAdminRequest struct {
	Email  string `json:"email"`
	Senha  string `json:"senha"`
	Cidade string `json:"cidade"`
}

type AdminResponse struct {
	ID     int64    `json:"id"`
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
