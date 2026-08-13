package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fredsaggio/bondrota-api/internal/publicid"
)

type contextKey string

const ClaimsKey contextKey = "claims"

const (
	RoleAdmin     = "admin"
	RoleCliente   = "cliente"
	RoleMotorista = "motorista"
)

func (s *AuthService) Authenticate(next http.Handler) http.Handler {
	return s.AuthenticateWithCookie("")(next)
}

// AuthenticateWithCookie accepts the regular Bearer token and, when configured,
// the HttpOnly cookie used exclusively by the administrative web application.
func (s *AuthService) AuthenticateWithCookie(cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			var tokenStr string
			if authHeader != "" {
				if !strings.HasPrefix(authHeader, "Bearer ") {
					http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
					return
				}
				tokenStr = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			} else if cookieName != "" {
				cookie, err := r.Cookie(cookieName)
				if err == nil {
					tokenStr = cookie.Value
				}
			}

			if tokenStr == "" {
				http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
				return
			}

			claims, err := s.ValidateToken(tokenStr)
			if err != nil {
				http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
				return
			}
			if err := s.resolveIdentity(r.Context(), claims); err != nil {
				http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (s *AuthService) resolveIdentity(ctx context.Context, claims *Claims) error {
	if s.identityResolver == nil || claims == nil {
		return errors.New("identity resolver is not configured")
	}

	var prefix publicid.Prefix
	switch claims.Role {
	case RoleAdmin:
		prefix = publicid.Admin
	case RoleCliente:
		prefix = publicid.Cliente
	case RoleMotorista:
		prefix = publicid.Motorista
	default:
		return errors.New("unsupported identity role")
	}

	id, err := s.identityResolver.Resolve(ctx, prefix, claims.Subject)
	if err != nil {
		return err
	}
	claims.UserID = id
	return nil
}

func (s *AuthService) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsKey).(*Claims)
			if !ok || !hasAllowedRole(claims.Role, roles) {
				http.Error(w, "Você não tem permissão para executar esta ação.", http.StatusForbidden)
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
				http.Error(w, "Sua sessão expirou. Entre novamente.", http.StatusUnauthorized)
				return
			}

			if hasAllowedRole(claims.Role, bypassRoles) {
				next.ServeHTTP(w, r)
				return
			}
			if claims.Role != ownerRole {
				http.Error(w, "Você não tem permissão para executar esta ação.", http.StatusForbidden)
				return
			}

			resourceID, ok := publicid.ResolvedParam(r.Context(), idParam)
			if !ok {
				http.Error(w, "Registro não encontrado.", http.StatusNotFound)
				return
			}
			if resourceID != claims.UserID {
				http.Error(w, "Você não tem permissão para executar esta ação.", http.StatusForbidden)
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
