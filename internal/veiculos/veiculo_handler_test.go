package veiculos_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/mocks"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
)

func newVeiculoRouter(h *veiculos.VeiculoHandler) http.Handler {
	r := chi.NewRouter()
	r.Delete("/veiculos/{veiculoID}", h.Delete)
	return r
}

func TestVeiculoHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*mocks.MockVeiculoStore)
		wantStatus int
	}{
		{
			name:       "sucesso → 204",
			id:         "1",
			setup:      func(st *mocks.MockVeiculoStore) { st.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "id inválido → 400",
			id:         "abc",
			setup:      func(_ *mocks.MockVeiculoStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "não encontrado → 404",
			id:   "99",
			setup: func(st *mocks.MockVeiculoStore) {
				st.EXPECT().Delete(mock.Anything, int64(99)).Return(veiculos.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			// veículo alocado em um ciclo de viagem (ON DELETE RESTRICT)
			name: "veículo em uso → 409",
			id:   "1",
			setup: func(st *mocks.MockVeiculoStore) {
				st.EXPECT().Delete(mock.Anything, int64(1)).
					Return(fmt.Errorf("db/veiculoStore.Delete: %w", &pgconn.PgError{Code: "23503", ConstraintName: "ciclos_viagem_veiculo_id_fkey"}))
			},
			wantStatus: http.StatusConflict,
		},
		{
			// falha genérica não pode virar 409: o admin leria "em uso" numa queda do banco
			name: "erro inesperado do banco → 500",
			id:   "1",
			setup: func(st *mocks.MockVeiculoStore) {
				st.EXPECT().Delete(mock.Anything, int64(1)).Return(errors.New("connection refused"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockVeiculoStore(t)
			tc.setup(st)
			h := veiculos.NewVeiculoHandler(st)
			req := httptest.NewRequest(http.MethodDelete, "/veiculos/"+tc.id, nil)
			rr := httptest.NewRecorder()
			newVeiculoRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}
