package auth

import (
	"net/http"
	"strings"
)

// ProtectCookieRequests blocks cross-origin state changes and rejects cookie-
// authenticated mutations without an Origin header. Bearer-only API clients
// remain compatible.
func ProtectCookieRequests(allowedOrigins []string, cookieName string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimRight(strings.TrimSpace(origin), "/")] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "Origem não autorizada.", http.StatusForbidden)
				return
			}

			_, cookieErr := r.Cookie(cookieName)
			if cookieErr == nil && r.Header.Get("Authorization") == "" {
				http.Error(w, "Origem não autorizada.", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
