package auth

type LoginRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type AuthHandler struct {
	svc *AuthService
}

func NewAuthHandler(svc *AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}
