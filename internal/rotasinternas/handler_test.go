package rotasinternas_test

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
	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
)

// --- helpers ---

func newRotaInternaRouter(h *rotasinternas.RotaInternaHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/rotas-internas", h.Create)
	r.Get("/rotas-internas", h.List)
	r.Get("/rotas-internas/{id}", h.GetByID)
	r.Patch("/rotas-internas/{id}/paradas", h.UpdateParadas)
	r.Delete("/rotas-internas/{id}", h.Delete)
	return r
}

func jsonBuf(v any) *bytes.Buffer {
	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(v)
	return &b
}

func sampleRotaInterna() *rotasinternas.RotaInterna {
	return &rotasinternas.RotaInterna{
		ID: 1,
		Paradas: []rotasinternas.ParadaOrdenada{
			{ID: 1, Nome: "P1", Latitude: 1.0, Longitude: 1.0, Ordem: 1},
			{ID: 2, Nome: "P2", Latitude: 2.0, Longitude: 2.0, Ordem: 2},
		},
	}
}

// --- Create ---

func TestRotaInternaHandler_Create(t *testing.T) {
	validBody := func() *bytes.Buffer {
		return jsonBuf(map[string]any{
			"paradas": []map[string]any{
				{"parada_id": 1, "ordem": 1},
				{"parada_id": 2, "ordem": 2},
			},
		})
	}

	tests := []struct {
		name       string
		body       *bytes.Buffer
		setup      func(*mocks.MockRotaInternaService)
		wantStatus int
	}{
		{
			name: "sucesso → 201",
			body: validBody(),
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in rotasinternas.CreateRotaInternaInput) bool {
					return len(in.Paradas) == 2
				})).Return(sampleRotaInterna(), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "body malformado → 400",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockRotaInternaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "sem paradas → 422",
			body: jsonBuf(map[string]any{"paradas": []map[string]any{}}),
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, rotasinternas.ErrSemParadas)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "ordem duplicada → 422",
			body: jsonBuf(map[string]any{
				"paradas": []map[string]any{
					{"parada_id": 1, "ordem": 1},
					{"parada_id": 2, "ordem": 1},
				},
			}),
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, rotasinternas.ErrOrdemDuplicada)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "parada inválida → 422",
			body: validBody(),
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, rotasinternas.ErrParadaInvalida)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "erro interno → 500",
			body: validBody(),
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockRotaInternaService(t)
			tc.setup(svc)
			h := rotasinternas.NewRotaInternaHandler(svc)
			req := httptest.NewRequest(http.MethodPost, "/rotas-internas", tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newRotaInternaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- GetByID ---

func TestRotaInternaHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*mocks.MockRotaInternaService)
		wantStatus int
	}{
		{
			name: "sucesso",
			id:   "1",
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleRotaInterna(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			setup:      func(_ *mocks.MockRotaInternaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrada → 404",
			id:   "99",
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, rotasinternas.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockRotaInternaService(t)
			tc.setup(svc)
			h := rotasinternas.NewRotaInternaHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/rotas-internas/"+tc.id, nil)
			rr := httptest.NewRecorder()
			newRotaInternaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- List ---

func TestRotaInternaHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*mocks.MockRotaInternaService)
		wantStatus int
	}{
		{
			name: "sucesso com itens",
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().List(mock.Anything).Return([]rotasinternas.RotaInterna{*sampleRotaInterna()}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "lista vazia",
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().List(mock.Anything).Return([]rotasinternas.RotaInterna{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "erro interno → 500",
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().List(mock.Anything).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockRotaInternaService(t)
			tc.setup(svc)
			h := rotasinternas.NewRotaInternaHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/rotas-internas", nil)
			rr := httptest.NewRecorder()
			newRotaInternaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- UpdateParadas ---

func TestRotaInternaHandler_UpdateParadas(t *testing.T) {
	validBody := func() *bytes.Buffer {
		return jsonBuf(map[string]any{
			"paradas": []map[string]any{
				{"parada_id": 3, "ordem": 1},
			},
		})
	}

	tests := []struct {
		name       string
		id         string
		body       *bytes.Buffer
		setup      func(*mocks.MockRotaInternaService)
		wantStatus int
	}{
		{
			name: "sucesso",
			id:   "1",
			body: validBody(),
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().UpdateParadas(mock.Anything, int64(1), mock.Anything).Return(sampleRotaInterna(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			body:       validBody(),
			setup:      func(_ *mocks.MockRotaInternaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "body malformado → 400",
			id:         "1",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockRotaInternaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrada → 404",
			id:   "99",
			body: validBody(),
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().UpdateParadas(mock.Anything, int64(99), mock.Anything).Return(nil, rotasinternas.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "sem paradas → 422",
			id:   "1",
			body: jsonBuf(map[string]any{"paradas": []map[string]any{}}),
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().UpdateParadas(mock.Anything, int64(1), mock.Anything).Return(nil, rotasinternas.ErrSemParadas)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			body: validBody(),
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().UpdateParadas(mock.Anything, int64(1), mock.Anything).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockRotaInternaService(t)
			tc.setup(svc)
			h := rotasinternas.NewRotaInternaHandler(svc)
			req := httptest.NewRequest(http.MethodPatch, "/rotas-internas/"+tc.id+"/paradas", tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newRotaInternaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- Delete ---

func TestRotaInternaHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*mocks.MockRotaInternaService)
		wantStatus int
	}{
		{
			name:       "sucesso → 204",
			id:         "1",
			setup:      func(svc *mocks.MockRotaInternaService) { svc.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			setup:      func(_ *mocks.MockRotaInternaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrada → 404",
			id:   "99",
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().Delete(mock.Anything, int64(99)).Return(rotasinternas.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			setup: func(svc *mocks.MockRotaInternaService) {
				svc.EXPECT().Delete(mock.Anything, int64(1)).Return(errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockRotaInternaService(t)
			tc.setup(svc)
			h := rotasinternas.NewRotaInternaHandler(svc)
			req := httptest.NewRequest(http.MethodDelete, "/rotas-internas/"+tc.id, nil)
			rr := httptest.NewRecorder()
			newRotaInternaRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}
