package admin

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type AdminHandler struct {
	svc AdminService
}

func NewAdminHandler(svc AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
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

	httputils.Respond(w, http.StatusOK, LoginResponse{Token: token})
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
