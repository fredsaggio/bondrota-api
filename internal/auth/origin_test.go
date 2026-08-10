package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProtectCookieRequests(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		origin     string
		cookie     bool
		bearer     bool
		wantStatus int
	}{
		{name: "allows configured browser origin", method: http.MethodPost, origin: "https://admin.bondrota.com", cookie: true, wantStatus: http.StatusNoContent},
		{name: "blocks foreign browser origin", method: http.MethodPost, origin: "https://evil.example", cookie: true, wantStatus: http.StatusForbidden},
		{name: "blocks cookie mutation without origin", method: http.MethodDelete, cookie: true, wantStatus: http.StatusForbidden},
		{name: "keeps bearer clients compatible", method: http.MethodPost, bearer: true, wantStatus: http.StatusNoContent},
		{name: "allows safe cookie request", method: http.MethodGet, cookie: true, wantStatus: http.StatusNoContent},
		{name: "blocks login csrf even before cookie exists", method: http.MethodPost, origin: "https://evil.example", wantStatus: http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/admin", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.cookie {
				req.AddCookie(&http.Cookie{Name: "admin_session", Value: "token"})
			}
			if tc.bearer {
				req.Header.Set("Authorization", "Bearer token")
			}
			rr := httptest.NewRecorder()
			middleware := ProtectCookieRequests([]string{"https://admin.bondrota.com"}, "admin_session")
			middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
