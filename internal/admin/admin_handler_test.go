package admin_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/mocks"
)

// --- router helper ---

func newAdminRouter(h *admin.AdminHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/admin/login", h.Login)
	r.Post("/admin", h.Create)
	r.Get("/admin", h.List)
	r.Get("/admin/{adminID}", h.GetByID)
	r.Patch("/admin/{adminID}", h.Update)
	r.Delete("/admin/{adminID}", h.Delete)
	return r
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
