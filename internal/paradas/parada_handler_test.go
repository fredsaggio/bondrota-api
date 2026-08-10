package paradas_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/mocks"
	"github.com/fredsaggio/bondrota-api/internal/paradas"
)

// --- helpers ---

func newParadaRouter(h *paradas.ParadaHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/paradas", h.Create)
	r.Get("/paradas", h.List)
	r.Get("/paradas/{id}", h.GetByID)
	r.Patch("/paradas/{id}", h.Update)
	r.Delete("/paradas/{id}", h.Delete)
	return r
}

func jsonBuf(v any) *bytes.Buffer {
	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(v)
	return &b
}

func sampleParada() *paradas.Parada {
	return &paradas.Parada{
		ID:        1,
		Nome:      "Terminal Recife",
		Latitude:  -8.063,
		Longitude: -34.871,
	}
}

var anyUpdateFunc = mock.MatchedBy(func(_ func(*paradas.Parada) (bool, error)) bool { return true })

// --- Create ---

func TestParadaHandler_Create(t *testing.T) {
	validBody := func() *bytes.Buffer {
		return jsonBuf(map[string]any{
			"nome":      "Terminal Recife",
			"latitude":  -8.063,
			"longitude": -34.871,
		})
	}

	tests := []struct {
		name       string
		body       *bytes.Buffer
		setup      func(*mocks.MockParadaStore)
		wantStatus int
	}{
		{
			name: "sucesso → 201",
			body: validBody(),
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in paradas.ParadaInput) bool {
					return in.Nome == "Terminal Recife"
				})).Return(sampleParada(), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "body malformado → 400",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockParadaStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "nome vazio → 400",
			body:       jsonBuf(map[string]any{"latitude": -8.0, "longitude": -34.0}),
			setup:      func(_ *mocks.MockParadaStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "erro interno → 500",
			body: validBody(),
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockParadaStore(t)
			tc.setup(st)
			h := paradas.NewParadaHandler(st)
			req := httptest.NewRequest(http.MethodPost, "/paradas", tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newParadaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// --- GetByID ---

func TestParadaHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*mocks.MockParadaStore)
		wantStatus int
	}{
		{
			name: "sucesso",
			id:   "1",
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleParada(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			setup:      func(_ *mocks.MockParadaStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrada → 404",
			id:   "99",
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, paradas.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockParadaStore(t)
			tc.setup(st)
			h := paradas.NewParadaHandler(st)
			req := httptest.NewRequest(http.MethodGet, "/paradas/"+tc.id, nil)
			rr := httptest.NewRecorder()
			newParadaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- List ---

func TestParadaHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*mocks.MockParadaStore)
		wantStatus int
	}{
		{
			name: "sucesso com itens",
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().List(mock.Anything).Return([]paradas.Parada{*sampleParada()}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "lista vazia",
			setup:      func(st *mocks.MockParadaStore) { st.EXPECT().List(mock.Anything).Return([]paradas.Parada{}, nil) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "erro interno → 500",
			setup:      func(st *mocks.MockParadaStore) { st.EXPECT().List(mock.Anything).Return(nil, errors.New("db")) },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockParadaStore(t)
			tc.setup(st)
			h := paradas.NewParadaHandler(st)
			req := httptest.NewRequest(http.MethodGet, "/paradas", nil)
			rr := httptest.NewRecorder()
			newParadaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- Update ---

func TestParadaHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       *bytes.Buffer
		setup      func(*mocks.MockParadaStore)
		wantStatus int
	}{
		{
			name: "sucesso",
			id:   "1",
			body: jsonBuf(map[string]any{"nome": "Terminal Integração"}),
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(sampleParada(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			body:       jsonBuf(map[string]any{}),
			setup:      func(_ *mocks.MockParadaStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "body malformado → 400",
			id:         "1",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockParadaStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrada → 404",
			id:   "99",
			body: jsonBuf(map[string]any{"nome": "X"}),
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().Update(mock.Anything, int64(99), anyUpdateFunc).Return(nil, paradas.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			body: jsonBuf(map[string]any{"nome": "X"}),
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockParadaStore(t)
			tc.setup(st)
			h := paradas.NewParadaHandler(st)
			req := httptest.NewRequest(http.MethodPatch, "/paradas/"+tc.id, tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newParadaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// --- Delete ---

func TestParadaHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*mocks.MockParadaStore)
		wantStatus int
	}{
		{
			name:       "sucesso → 204",
			id:         "1",
			setup:      func(st *mocks.MockParadaStore) { st.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			setup:      func(_ *mocks.MockParadaStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrada → 404",
			id:   "99",
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().Delete(mock.Anything, int64(99)).Return(paradas.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			// qualquer outro erro (ex: FK violation por rota interna) → 409 Conflict
			name: "parada em uso por rota interna → 409",
			id:   "1",
			setup: func(st *mocks.MockParadaStore) {
				st.EXPECT().Delete(mock.Anything, int64(1)).Return(errors.New("fk violation"))
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockParadaStore(t)
			tc.setup(st)
			h := paradas.NewParadaHandler(st)
			req := httptest.NewRequest(http.MethodDelete, "/paradas/"+tc.id, nil)
			rr := httptest.NewRecorder()
			newParadaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}
