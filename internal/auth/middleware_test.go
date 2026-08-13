package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/publicid"
)

type staticIdentityResolver struct{}

func (staticIdentityResolver) Resolve(_ context.Context, _ publicid.Prefix, _ string) (int64, error) {
	return 42, nil
}

func TestAuthenticateWithCookie(t *testing.T) {
	svc := NewAuthService(nil, "test-secret")
	svc.SetIdentityResolver(staticIdentityResolver{})
	token, err := svc.GenerateToken("adm_012345678901234567890", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		configure  func(*http.Request)
		wantStatus int
	}{
		{
			name: "accepts bearer token",
			configure: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+token)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "accepts admin cookie",
			configure: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "admin_session", Value: token})
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "rejects missing credentials",
			configure:  func(*http.Request) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "does not fall back to cookie for malformed authorization",
			configure: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic invalid")
				r.AddCookie(&http.Cookie{Name: "admin_session", Value: token})
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims, ok := r.Context().Value(ClaimsKey).(*Claims)
				if !ok || claims.UserID != 42 || claims.Role != RoleAdmin {
					t.Fatal("expected admin claims in request context")
				}
				w.WriteHeader(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/admin/session", nil)
			tc.configure(req)
			rr := httptest.NewRecorder()
			svc.AuthenticateWithCookie("admin_session")(next).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
