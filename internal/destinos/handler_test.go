package destinos_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/destinos"
	"github.com/fredsaggio/bondrota-api/internal/mocks"
)

// --- helpers ---

func newDestinoRouter(h *destinos.DestinoHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/destinos", h.Create)
	r.Get("/destinos", h.List)
	r.Get("/destinos/{id}", h.GetByID)
	r.Get("/destinos/cidade/{cidade}", h.ListByCity)
	r.Patch("/destinos/{id}", h.Update)
	r.Delete("/destinos/{id}", h.Delete)
	return r
}

func jsonBuf(v any) *bytes.Buffer {
	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(v)
	return &b
}

func sampleDestino() *destinos.Destino {
	return &destinos.Destino{
		ID:        1,
		Nome:      "UFPE",
		Rua:       "Av. Jornalista Anibal Fernandes",
		Cidade:    "recife",
		Latitude:  -8.052000,
		Longitude: -34.951000,
	}
}

var anyUpdateFunc = mock.MatchedBy(func(_ func(*destinos.Destino) (bool, error)) bool { return true })

// --- Create ---

func TestDestinoHandler_Create(t *testing.T) {
	validBody := func() *bytes.Buffer {
		return jsonBuf(map[string]any{
			"nome":      "UFPE",
			"rua":       "Av. Jornalista Anibal Fernandes",
			"cidade":    "Recife",
			"latitude":  -8.052,
			"longitude": -34.951,
		})
	}

	tests := []struct {
		name       string
		body       *bytes.Buffer
		setup      func(*mocks.MockDestinoStore)
		wantStatus int
	}{
		{
			name: "sucesso → 201",
			body: validBody(),
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in destinos.DestinoInput) bool {
					return in.Nome == "UFPE" && in.Cidade == "recife" // normaliza para lowercase
				})).Return(sampleDestino(), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "body malformado → 400",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockDestinoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "nome vazio → 400",
			body:       jsonBuf(map[string]any{"rua": "Rua X", "cidade": "Recife", "latitude": -8.0, "longitude": -34.0}),
			setup:      func(_ *mocks.MockDestinoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rua vazia → 400",
			body:       jsonBuf(map[string]any{"nome": "UFPE", "cidade": "Recife", "latitude": -8.0, "longitude": -34.0}),
			setup:      func(_ *mocks.MockDestinoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "cidade vazia → 400",
			body:       jsonBuf(map[string]any{"nome": "UFPE", "rua": "Rua X", "latitude": -8.0, "longitude": -34.0}),
			setup:      func(_ *mocks.MockDestinoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "latitude e longitude zero → 400",
			body:       jsonBuf(map[string]any{"nome": "UFPE", "rua": "Rua X", "cidade": "Recife"}),
			setup:      func(_ *mocks.MockDestinoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "erro interno → 500",
			body: validBody(),
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockDestinoStore(t)
			tc.setup(st)
			h := destinos.NewDestinoHandler(st)
			req := httptest.NewRequest(http.MethodPost, "/destinos", tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newDestinoRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// --- GetByID ---

func TestDestinoHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*mocks.MockDestinoStore)
		wantStatus int
	}{
		{
			name: "sucesso",
			id:   "1",
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleDestino(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			setup:      func(_ *mocks.MockDestinoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrado → 404",
			id:   "99",
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, destinos.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockDestinoStore(t)
			tc.setup(st)
			h := destinos.NewDestinoHandler(st)
			req := httptest.NewRequest(http.MethodGet, "/destinos/"+tc.id, nil)
			rr := httptest.NewRecorder()
			newDestinoRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- List ---

func TestDestinoHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*mocks.MockDestinoStore)
		wantStatus int
	}{
		{
			name: "sucesso com itens",
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().List(mock.Anything).Return([]destinos.Destino{*sampleDestino()}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "lista vazia",
			setup:      func(st *mocks.MockDestinoStore) { st.EXPECT().List(mock.Anything).Return([]destinos.Destino{}, nil) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "erro interno → 500",
			setup:      func(st *mocks.MockDestinoStore) { st.EXPECT().List(mock.Anything).Return(nil, errors.New("db")) },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockDestinoStore(t)
			tc.setup(st)
			h := destinos.NewDestinoHandler(st)
			req := httptest.NewRequest(http.MethodGet, "/destinos", nil)
			rr := httptest.NewRecorder()
			newDestinoRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- ListByCity ---

func TestDestinoHandler_ListByCity(t *testing.T) {
	tests := []struct {
		name       string
		cidade     string
		setup      func(*mocks.MockDestinoStore)
		wantStatus int
	}{
		{
			name:   "sucesso — normaliza para lowercase",
			cidade: "Recife",
			setup: func(st *mocks.MockDestinoStore) {
				// handler faz strings.ToLower antes de passar ao store
				st.EXPECT().ListByCity(mock.Anything, "recife").Return([]destinos.Destino{*sampleDestino()}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "lista vazia",
			cidade: "Olinda",
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().ListByCity(mock.Anything, "olinda").Return([]destinos.Destino{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "erro interno → 500",
			cidade: "Recife",
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().ListByCity(mock.Anything, "recife").Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockDestinoStore(t)
			tc.setup(st)
			h := destinos.NewDestinoHandler(st)
			req := httptest.NewRequest(http.MethodGet, "/destinos/cidade/"+tc.cidade, nil)
			rr := httptest.NewRecorder()
			newDestinoRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- Update ---

func TestDestinoHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       *bytes.Buffer
		setup      func(*mocks.MockDestinoStore)
		wantStatus int
	}{
		{
			name: "sucesso",
			id:   "1",
			body: jsonBuf(map[string]any{"nome": "UFPE Sede Nova"}),
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(sampleDestino(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			body:       jsonBuf(map[string]any{}),
			setup:      func(_ *mocks.MockDestinoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "body malformado → 400",
			id:         "1",
			body:       bytes.NewBufferString("bad"),
			setup:      func(_ *mocks.MockDestinoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrado → 404",
			id:   "99",
			body: jsonBuf(map[string]any{"nome": "X"}),
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().Update(mock.Anything, int64(99), anyUpdateFunc).Return(nil, destinos.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "erro interno → 500",
			id:   "1",
			body: jsonBuf(map[string]any{"nome": "X"}),
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockDestinoStore(t)
			tc.setup(st)
			h := destinos.NewDestinoHandler(st)
			req := httptest.NewRequest(http.MethodPatch, "/destinos/"+tc.id, tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newDestinoRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// --- Delete ---

func TestDestinoHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*mocks.MockDestinoStore)
		wantStatus int
	}{
		{
			name:       "sucesso → 204",
			id:         "1",
			setup:      func(st *mocks.MockDestinoStore) { st.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			setup:      func(_ *mocks.MockDestinoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrado → 404",
			id:   "99",
			setup: func(st *mocks.MockDestinoStore) {
				st.EXPECT().Delete(mock.Anything, int64(99)).Return(destinos.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "erro interno → 500",
			id:         "1",
			setup:      func(st *mocks.MockDestinoStore) { st.EXPECT().Delete(mock.Anything, int64(1)).Return(errors.New("db")) },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockDestinoStore(t)
			tc.setup(st)
			h := destinos.NewDestinoHandler(st)
			req := httptest.NewRequest(http.MethodDelete, "/destinos/"+tc.id, nil)
			rr := httptest.NewRecorder()
			newDestinoRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}
