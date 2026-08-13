package veiculos_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/mocks"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
)

func newVeiculoRouter(h *veiculos.VeiculoHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/veiculos/", h.Create)
	r.Put("/veiculos/{veiculoID}", h.Update)
	r.Delete("/veiculos/{veiculoID}", h.Delete)
	return r
}

func TestVeiculoHandler_RejeitaModeloComCaracterEspecial(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		st := mocks.NewMockVeiculoStore(t)
		h := veiculos.NewVeiculoHandler(st)
		req := httptest.NewRequest(http.MethodPost, "/veiculos/", strings.NewReader(`{
			"placa":"ABC1D23","modelo":"Ônibus #1722","categoria":"escolar",
			"capacidade":24,"status":"ativo"
		}`))
		rr := httptest.NewRecorder()

		newVeiculoRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("update", func(t *testing.T) {
		st := mocks.NewMockVeiculoStore(t)
		st.EXPECT().Update(mock.Anything, int64(1), mock.Anything).RunAndReturn(
			func(_ context.Context, _ int64, update func(*veiculos.Veiculo) (bool, error)) (*veiculos.Veiculo, error) {
				current := &veiculos.Veiculo{
					ID: 1, Placa: "ABC1D23", Modelo: "Ônibus 1722",
					Categoria: veiculos.CategoriaEscolar, Capacidade: veiculos.CapacidadeEscolar,
					Status: veiculos.StatusAtivo,
				}
				_, err := update(current)
				return nil, err
			},
		)
		h := veiculos.NewVeiculoHandler(st)
		req := httptest.NewRequest(http.MethodPut, "/veiculos/1", strings.NewReader(`{"modelo":"Ônibus/1722"}`))
		rr := httptest.NewRecorder()

		newVeiculoRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestVeiculoHandler_RejeitaPlacaComSimbolosMisturados(t *testing.T) {
	st := mocks.NewMockVeiculoStore(t)
	h := veiculos.NewVeiculoHandler(st)
	req := httptest.NewRequest(http.MethodPost, "/veiculos/", strings.NewReader(`{
		"placa":"A@B#C1D23","modelo":"Ônibus 1722","categoria":"escolar",
		"capacidade":24,"status":"ativo"
	}`))
	rr := httptest.NewRecorder()

	newVeiculoRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestVeiculoHandler_PlacaDuplicada(t *testing.T) {
	duplicateError := fmt.Errorf("db/veiculoStore: %w", &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "veiculos_placa_key",
	})
	const wantMessage = "Já existe outro veículo cadastrado com esta placa.\n"

	t.Run("create retorna conflito claro", func(t *testing.T) {
		st := mocks.NewMockVeiculoStore(t)
		st.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, duplicateError)
		h := veiculos.NewVeiculoHandler(st)
		req := httptest.NewRequest(http.MethodPost, "/veiculos/", strings.NewReader(`{
			"placa":"ABC1D23","modelo":"Onibus 1722","categoria":"escolar",
			"capacidade":24,"status":"ativo"
		}`))
		rr := httptest.NewRecorder()

		newVeiculoRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict || rr.Body.String() != wantMessage {
			t.Fatalf("want 409 with clear message, got %d: %q", rr.Code, rr.Body.String())
		}
	})

	t.Run("update retorna conflito claro", func(t *testing.T) {
		st := mocks.NewMockVeiculoStore(t)
		st.EXPECT().Update(mock.Anything, int64(1), mock.Anything).Return(nil, duplicateError)
		h := veiculos.NewVeiculoHandler(st)
		req := httptest.NewRequest(http.MethodPut, "/veiculos/1", strings.NewReader(`{"placa":"ABC1D23"}`))
		rr := httptest.NewRecorder()

		newVeiculoRouter(h).ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict || rr.Body.String() != wantMessage {
			t.Fatalf("want 409 with clear message, got %d: %q", rr.Code, rr.Body.String())
		}
	})
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
