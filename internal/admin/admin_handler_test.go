package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/mocks"
)

// --- router helper ---

func newAdminRouter(h *admin.AdminHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/admin/login", h.Login)
	r.Post("/admin/logout", h.Logout)
	r.Get("/admin/session", h.Session)
	r.Put("/admin/senha", h.ChangePassword)
	r.Post("/admin", h.Create)
	r.Get("/admin", h.List)
	r.Get("/admin/{adminID}", h.GetByID)
	r.Patch("/admin/{adminID}", h.Update)
	r.Delete("/admin/{adminID}", h.Delete)
	return r
}

func TestAdminHandler_LoginSetsHttpOnlyCookie(t *testing.T) {
	svc := mocks.NewMockAdminService(t)
	svc.EXPECT().Login(mock.Anything, "admin@bondrota.com", "secret").Return("signed-token", nil)
	h := admin.NewAdminHandler(svc, admin.SessionCookieConfig{
		Name:     "admin_session",
		Path:     "/api/v1",
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		TTL:      time.Hour,
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/login", jsonBuf(map[string]any{
		"email": "admin@bondrota.com",
		"senha": "secret",
	}))
	req.Header.Set(admin.SessionModeHeader, "cookie")
	rr := httptest.NewRecorder()
	newAdminRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("want %d, got %d: %s", http.StatusNoContent, rr.Code, rr.Body.String())
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one session cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "admin_session" || cookie.Value != "signed-token" || cookie.Path != "/api/v1" {
		t.Fatalf("unexpected cookie: %#v", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteNoneMode {
		t.Fatalf("cookie security attributes missing: %#v", cookie)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("cookie-mode login must not expose the JWT: %q", rr.Body.String())
	}
}

func TestAdminHandler_Session(t *testing.T) {
	svc := mocks.NewMockAdminService(t)
	h := admin.NewAdminHandler(svc)
	expiresAt := time.Now().Add(time.Hour).Truncate(time.Second)
	claims := &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiresAt)},
		UserID:           7,
		Role:             auth.RoleAdmin,
	}
	req := httptest.NewRequest(http.MethodGet, "/admin/session", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsKey, claims))
	rr := httptest.NewRecorder()
	newAdminRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var response admin.SessionResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.UserID != 7 || response.Role != auth.RoleAdmin || response.ExpiresAt != expiresAt.UnixMilli() {
		t.Fatalf("unexpected session: %#v", response)
	}
}

func TestAdminHandler_LogoutClearsCookie(t *testing.T) {
	svc := mocks.NewMockAdminService(t)
	h := admin.NewAdminHandler(svc)
	rr := httptest.NewRecorder()
	newAdminRouter(h).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/logout", nil))

	if rr.Code != http.StatusNoContent {
		t.Fatalf("want %d, got %d", http.StatusNoContent, rr.Code)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 || !cookies[0].HttpOnly {
		t.Fatalf("expected an expired HttpOnly cookie, got %#v", cookies)
	}
}

func jsonBuf(v any) *bytes.Buffer {
	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(v)
	return &b
}

func sampleAdmin() *admin.Admin {
	return &admin.Admin{ID: 1, Email: "admin@bondrota.com", Senha: "hash"}
}

// --- Login ---

func TestAdminHandler_Login(t *testing.T) {
	tests := []struct {
		name       string
		body       *bytes.Buffer
		setup      func(*mocks.MockAdminService)
		wantStatus int
	}{
		{
			name: "sucesso",
			body: jsonBuf(map[string]any{"email": "admin@bondrota.com", "senha": "secret"}),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Login(mock.Anything, "admin@bondrota.com", "secret").Return("tok123", nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "body malformado → 400",
			body:       bytes.NewBufferString("not-json"),
			setup:      func(_ *mocks.MockAdminService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "credenciais inválidas → 401",
			body: jsonBuf(map[string]any{"email": "x@x.com", "senha": "wrong"}),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Login(mock.Anything, "x@x.com", "wrong").Return("", auth.ErrInvalidCredentials)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "admin não encontrado → 401",
			body: jsonBuf(map[string]any{"email": "ghost@x.com", "senha": "pw"}),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Login(mock.Anything, "ghost@x.com", "pw").Return("", admin.ErrNotFound)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "erro interno → 500",
			body: jsonBuf(map[string]any{"email": "a@a.com", "senha": "pw"}),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Login(mock.Anything, "a@a.com", "pw").Return("", errors.New("db err"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockAdminService(t)
			tc.setup(svc)
			h := admin.NewAdminHandler(svc)
			req := httptest.NewRequest(http.MethodPost, "/admin/login", tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newAdminRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// --- Create ---

func TestAdminHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       *bytes.Buffer
		setup      func(*mocks.MockAdminService)
		wantStatus int
	}{
		{
			name: "sucesso → 201",
			body: jsonBuf(map[string]any{"email": "new@bondrota.com", "senha": "pw"}),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Create(mock.Anything, admin.AdminInput{Email: "new@bondrota.com", Senha: "pw"}).
					Return(sampleAdmin(), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "body malformado → 400",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockAdminService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "erro interno → 500",
			body: jsonBuf(map[string]any{"email": "a@a.com", "senha": "pw"}),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db err"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockAdminService(t)
			tc.setup(svc)
			h := admin.NewAdminHandler(svc)
			req := httptest.NewRequest(http.MethodPost, "/admin", tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newAdminRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// --- GetByID ---

func TestAdminHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		adminID    string
		setup      func(*mocks.MockAdminService)
		wantStatus int
	}{
		{
			name:    "sucesso",
			adminID: "1",
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleAdmin(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			adminID:    "abc",
			setup:      func(_ *mocks.MockAdminService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "não encontrado → 404",
			adminID: "99",
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, admin.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:    "erro interno → 500",
			adminID: "1",
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockAdminService(t)
			tc.setup(svc)
			h := admin.NewAdminHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/admin/"+tc.adminID, nil)
			rr := httptest.NewRecorder()
			newAdminRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- List ---

func TestAdminHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*mocks.MockAdminService)
		wantStatus int
	}{
		{
			name: "sucesso com itens",
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().List(mock.Anything).Return([]admin.Admin{*sampleAdmin()}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "lista vazia",
			setup:      func(svc *mocks.MockAdminService) { svc.EXPECT().List(mock.Anything).Return([]admin.Admin{}, nil) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "erro interno → 500",
			setup:      func(svc *mocks.MockAdminService) { svc.EXPECT().List(mock.Anything).Return(nil, errors.New("db")) },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockAdminService(t)
			tc.setup(svc)
			h := admin.NewAdminHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			rr := httptest.NewRecorder()
			newAdminRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- Update ---

func TestAdminHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		adminID    string
		body       *bytes.Buffer
		setup      func(*mocks.MockAdminService)
		wantStatus int
	}{
		{
			name:    "sucesso",
			adminID: "1",
			body:    jsonBuf(map[string]any{"email": "novo@bondrota.com"}),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Update(mock.Anything, int64(1), "novo@bondrota.com").Return(sampleAdmin(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			adminID:    "abc",
			body:       jsonBuf(map[string]any{}),
			setup:      func(_ *mocks.MockAdminService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "body malformado → 400",
			adminID:    "1",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockAdminService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "não encontrado → 404",
			adminID: "99",
			body:    jsonBuf(map[string]any{"email": "x@x.com"}),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Update(mock.Anything, int64(99), "x@x.com").Return(nil, admin.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:    "erro interno → 500",
			adminID: "1",
			body:    jsonBuf(map[string]any{"email": "x@x.com"}),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Update(mock.Anything, int64(1), "x@x.com").Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockAdminService(t)
			tc.setup(svc)
			h := admin.NewAdminHandler(svc)
			req := httptest.NewRequest(http.MethodPatch, "/admin/"+tc.adminID, tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newAdminRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// --- Delete ---

func TestAdminHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		adminID    string
		setup      func(*mocks.MockAdminService)
		wantStatus int
	}{
		{
			name:       "sucesso → 204",
			adminID:    "1",
			setup:      func(svc *mocks.MockAdminService) { svc.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "id inválido → 400",
			adminID:    "abc",
			setup:      func(_ *mocks.MockAdminService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "não encontrado → 404",
			adminID: "99",
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Delete(mock.Anything, int64(99)).Return(admin.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:    "erro interno → 500",
			adminID: "1",
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().Delete(mock.Anything, int64(1)).Return(errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockAdminService(t)
			tc.setup(svc)
			h := admin.NewAdminHandler(svc)
			req := httptest.NewRequest(http.MethodDelete, "/admin/"+tc.adminID, nil)
			rr := httptest.NewRecorder()
			newAdminRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- ChangePassword ---

func changePasswordRequest(body string, claims *auth.Claims) *http.Request {
	req := httptest.NewRequest(http.MethodPut, "/admin/senha", bytes.NewBufferString(body))
	if claims != nil {
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsKey, claims))
	}
	return req
}

func adminClaims(userID int64) *auth.Claims {
	return &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		UserID:           userID,
		Role:             auth.RoleAdmin,
	}
}

// O alvo da troca vem do JWT. Mesmo mandando outro id no corpo, o service tem que
// receber o id de quem esta autenticado.
func TestAdminHandler_ChangePasswordIgnoresBodyTarget(t *testing.T) {
	svc := mocks.NewMockAdminService(t)
	svc.EXPECT().ChangePassword(mock.Anything, int64(7), "atual", "nova-senha-1").Return("novo-token", nil)
	h := admin.NewAdminHandler(svc, admin.SessionCookieConfig{Name: "sessao", Secure: true})

	rr := httptest.NewRecorder()
	newAdminRouter(h).ServeHTTP(rr, changePasswordRequest(
		`{"admin_id":999,"id":999,"senha_atual":"atual","nova_senha":"nova-senha-1"}`,
		adminClaims(7),
	))

	if rr.Code != http.StatusNoContent {
		t.Fatalf("want %d, got %d: %s", http.StatusNoContent, rr.Code, rr.Body.String())
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != "novo-token" {
		t.Fatalf("a sessao de quem trocou precisa receber o token novo, got %#v", cookies)
	}
	if !cookies[0].HttpOnly {
		t.Fatal("o cookie reemitido tem que continuar HttpOnly")
	}
}

// Senha atual errada nao pode responder 401: o painel derruba a sessao em qualquer
// 401, entao um erro de digitacao deslogaria quem esta trocando a senha.
func TestAdminHandler_ChangePasswordWrongCurrentIsNotUnauthorized(t *testing.T) {
	svc := mocks.NewMockAdminService(t)
	svc.EXPECT().ChangePassword(mock.Anything, int64(7), "errada", "nova-senha-1").
		Return("", auth.ErrInvalidCredentials)
	h := admin.NewAdminHandler(svc)

	rr := httptest.NewRecorder()
	newAdminRouter(h).ServeHTTP(rr, changePasswordRequest(
		`{"senha_atual":"errada","nova_senha":"nova-senha-1"}`,
		adminClaims(7),
	))

	if rr.Code == http.StatusUnauthorized {
		t.Fatal("401 desloga o usuario no painel; senha atual errada precisa de outro status")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}
	if len(rr.Result().Cookies()) != 0 {
		t.Fatal("nao pode reemitir cookie quando a troca falha")
	}
}

func TestAdminHandler_ChangePasswordErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		claims     *auth.Claims
		setup      func(*mocks.MockAdminService)
		wantStatus int
	}{
		{
			name:       "sem sessao",
			body:       `{"senha_atual":"a","nova_senha":"nova-senha-1"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "corpo invalido",
			body:       `{`,
			claims:     adminClaims(7),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "senha nova curta",
			body:   `{"senha_atual":"atual","nova_senha":"curta"}`,
			claims: adminClaims(7),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().ChangePassword(mock.Anything, int64(7), "atual", "curta").
					Return("", admin.ErrSenhaFraca)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "falha inesperada nao vaza detalhe",
			body:   `{"senha_atual":"atual","nova_senha":"nova-senha-1"}`,
			claims: adminClaims(7),
			setup: func(svc *mocks.MockAdminService) {
				svc.EXPECT().ChangePassword(mock.Anything, int64(7), "atual", "nova-senha-1").
					Return("", errors.New("connection refused to 10.0.0.5"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockAdminService(t)
			if tc.setup != nil {
				tc.setup(svc)
			}
			h := admin.NewAdminHandler(svc)

			rr := httptest.NewRecorder()
			newAdminRouter(h).ServeHTTP(rr, changePasswordRequest(tc.body, tc.claims))

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
			if tc.wantStatus == http.StatusInternalServerError && strings.Contains(rr.Body.String(), "10.0.0.5") {
				t.Fatalf("erro interno vazou para a resposta: %s", rr.Body.String())
			}
		})
	}
}
