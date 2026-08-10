package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type contextKey string

const ClaimsKey contextKey = "claims"

const (
	RoleAdmin     = "admin"
	RoleCliente   = "cliente"
	RoleMotorista = "motorista"
)

func (s *AuthService) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := s.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *AuthService) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsKey).(*Claims)
			if !ok || !hasAllowedRole(claims.Role, roles) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireUserIDOrRole(idParam, ownerRole string, bypassRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsKey).(*Claims)
			if !ok || claims.UserID <= 0 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if hasAllowedRole(claims.Role, bypassRoles) {
				next.ServeHTTP(w, r)
				return
			}
			if claims.Role != ownerRole {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			resourceID, err := strconv.ParseInt(chi.URLParam(r, idParam), 10, 64)
			if err != nil || resourceID <= 0 {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}
			if resourceID != claims.UserID {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func hasAllowedRole(userRole string, allowedRoles []string) bool {
	for _, role := range allowedRoles {
		if userRole == role {
			return true
		}
	}
	return false
}
