package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

func requireBearerSecret(secret string) func(http.Handler) http.Handler {
	expectedHash := sha256.Sum256([]byte(secret))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			providedHash := sha256.Sum256([]byte(provided))
			if secret == "" || !found || provided == "" || subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
