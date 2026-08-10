package reservas_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/mocks"
	"github.com/fredsaggio/bondrota-api/internal/reservas"
)

func newRouter(h *reservas.ReservaHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/clientes/{clienteID}/vinculos/{vinculoID}/reservas", h.CreateByVinculo)
	r.Get("/clientes/{clienteID}/vinculos/{vinculoID}/reservas/disponibilidade", h.ConsultarDisponibilidade)
	r.Get("/reservas", h.List)
	r.Get("/reservas/{reservaID}", h.GetByID)
	r.Get("/clientes/{clienteID}/reservas", h.ListByCliente)
	r.Get("/clientes/{clienteID}/vinculos/{vinculoID}/reservas", h.ListByVinculo)
	r.Patch("/reservas/{reservaID}", h.Update)
	r.Post("/reservas/{reservaID}/cancel", h.Cancel)
	r.Delete("/reservas/{reservaID}", h.Delete)
	return r
}

func sampleReserva() *reservas.Reserva {
	return &reservas.Reserva{
		ID:            1,
		ClienteID:     10,
		VinculoID:     20,
		DataViagem:    time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Turno:         reservas.TurnoMatutino,
		DestinoID:     5,
		RotaInternaID: 3,
		Sentido:       reservas.SentidoIda,
		Status:        reservas.StatusConfirmada,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func jsonBody(v any) *bytes.Buffer {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(v)
	return &buf
}

// anyUpdateFunc é um matcher para o argumento func(*Reserva)(bool,error).
var anyUpdateFunc = mock.MatchedBy(func(_ func(*reservas.Reserva) (bool, error)) bool { return true })

// --- CreateByVinculo ---

func TestHandler_CreateByVinculo(t *testing.T) {
	validBody := map[string]any{"data_viagem": "2026-07-01", "sentido": "ida"}

	tests := []struct {
		name       string
		clienteID  string
		vinculoID  string
		body       *bytes.Buffer
		setup      func(*mocks.MockReservaService)
		wantStatus int
	}{
		{
			name:      "sucesso",
			clienteID: "10", vinculoID: "20",
			body: jsonBody(validBody),
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in reservas.ReservaInput) bool {
					return in.ClienteID == 10 && in.VinculoID == 20 && in.Sentido == reservas.SentidoIda
				})).Return(sampleReserva(), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:      "clienteID inválido",
			clienteID: "abc", vinculoID: "20",
			body:       jsonBody(validBody),
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "body malformado",
			clienteID: "10", vinculoID: "20",
			body:       bytes.NewBufferString("not-json"),
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "data_viagem ausente",
			clienteID: "10", vinculoID: "20",
			body:       jsonBody(map[string]any{"sentido": "ida"}),
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "sentido inválido → 422 (service valida)",
			clienteID: "10", vinculoID: "20",
			body: jsonBody(map[string]any{"data_viagem": "2026-07-01", "sentido": "nenhum"}),
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, reservas.ErrSentidoInvalido)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:      "vinculo não encontrado → 404",
			clienteID: "10", vinculoID: "20",
			body: jsonBody(validBody),
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, reservas.ErrVinculoNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "prazo encerrado → 409",
			clienteID: "10", vinculoID: "20",
			body: jsonBody(validBody),
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, reservas.ErrPrazoReservaEncerrado)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:      "horario não configurado → 422",
			clienteID: "10", vinculoID: "20",
			body: jsonBody(validBody),
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, reservas.ErrHorarioNaoConfigurado)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:      "erro interno → 500",
			clienteID: "10", vinculoID: "20",
			body: jsonBody(validBody),
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db err"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockReservaService(t)
			tc.setup(svc)

			h := reservas.NewReservaHandler(svc)
			req := httptest.NewRequest(http.MethodPost,
				"/clientes/"+tc.clienteID+"/vinculos/"+tc.vinculoID+"/reservas", tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandler_ConsultarDisponibilidade(t *testing.T) {
	fechamento := time.Date(2026, 7, 1, 16, 30, 0, 0, testLocation)
	partida := time.Date(2026, 7, 1, 17, 0, 0, 0, testLocation)
	consultado := time.Date(2026, 7, 1, 16, 0, 0, 0, testLocation)

	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockReservaService)
		wantStatus int
	}{
		{
			name: "sucesso",
			path: "/clientes/10/vinculos/20/reservas/disponibilidade?data_viagem=2026-07-01&sentido=ida",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().ConsultarDisponibilidade(mock.Anything, mock.MatchedBy(func(input reservas.DisponibilidadeReservaInput) bool {
					return input.ClienteID == 10 && input.VinculoID == 20 && input.Sentido == reservas.SentidoIda
				})).Return(&reservas.DisponibilidadeReserva{
					DataViagem:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
					Turno:        reservas.TurnoMatutino,
					Sentido:      reservas.SentidoIda,
					PartidaEm:    partida,
					FechamentoEm: fechamento,
					ConsultadoEm: consultado,
					Disponivel:   true,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "data ausente",
			path:       "/clientes/10/vinculos/20/reservas/disponibilidade?sentido=ida",
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "sentido invalido",
			path: "/clientes/10/vinculos/20/reservas/disponibilidade?data_viagem=2026-07-01&sentido=nenhum",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().ConsultarDisponibilidade(mock.Anything, mock.Anything).Return(nil, reservas.ErrSentidoInvalido)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockReservaService(t)
			tc.setup(svc)
			h := reservas.NewReservaHandler(svc)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			newRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
			if tc.wantStatus == http.StatusOK && !bytes.Contains(rr.Body.Bytes(), []byte(`"fechamento_em":"2026-07-01T16:30:00-03:00"`)) {
				t.Fatalf("unexpected response: %s", rr.Body.String())
			}
		})
	}
}

// --- GetByID ---

func TestHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		reservaID  string
		setup      func(*mocks.MockReservaService)
		wantStatus int
	}{
		{
			name:      "sucesso",
			reservaID: "1",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleReserva(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido",
			reservaID:  "abc",
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "não encontrado → 404",
			reservaID: "99",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, reservas.ErrReservaNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "erro interno → 500",
			reservaID: "1",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockReservaService(t)
			tc.setup(svc)
			h := reservas.NewReservaHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/reservas/"+tc.reservaID, nil)
			rr := httptest.NewRecorder()
			newRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- List ---

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*mocks.MockReservaService)
		wantStatus int
	}{
		{
			name: "sucesso com itens",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().List(mock.Anything).Return([]reservas.Reserva{*sampleReserva()}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "lista vazia",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().List(mock.Anything).Return([]reservas.Reserva{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "erro interno → 500",
			setup:      func(svc *mocks.MockReservaService) { svc.EXPECT().List(mock.Anything).Return(nil, errors.New("db")) },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockReservaService(t)
			tc.setup(svc)
			h := reservas.NewReservaHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/reservas", nil)
			rr := httptest.NewRecorder()
			newRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- ListByCliente ---

func TestHandler_ListByCliente(t *testing.T) {
	tests := []struct {
		name       string
		clienteID  string
		setup      func(*mocks.MockReservaService)
		wantStatus int
	}{
		{
			name:      "sucesso",
			clienteID: "10",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().ListByCliente(mock.Anything, int64(10)).Return([]reservas.Reserva{*sampleReserva()}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido",
			clienteID:  "abc",
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "erro interno → 500",
			clienteID: "10",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().ListByCliente(mock.Anything, int64(10)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockReservaService(t)
			tc.setup(svc)
			h := reservas.NewReservaHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/clientes/"+tc.clienteID+"/reservas", nil)
			rr := httptest.NewRecorder()
			newRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- ListByVinculo ---

func TestHandler_ListByVinculo(t *testing.T) {
	tests := []struct {
		name       string
		clienteID  string
		vinculoID  string
		setup      func(*mocks.MockReservaService)
		wantStatus int
	}{
		{
			name:      "sucesso",
			clienteID: "10", vinculoID: "20",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().ListByVinculo(mock.Anything, int64(10), int64(20)).Return([]reservas.Reserva{*sampleReserva()}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "id inválido",
			clienteID: "abc", vinculoID: "20",
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "vinculo não encontrado → 404",
			clienteID: "10", vinculoID: "99",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().ListByVinculo(mock.Anything, int64(10), int64(99)).Return(nil, reservas.ErrVinculoNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "erro interno → 500",
			clienteID: "10", vinculoID: "20",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().ListByVinculo(mock.Anything, int64(10), int64(20)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockReservaService(t)
			tc.setup(svc)
			h := reservas.NewReservaHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/clientes/"+tc.clienteID+"/vinculos/"+tc.vinculoID+"/reservas", nil)
			rr := httptest.NewRecorder()
			newRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- Update ---

func TestHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		reservaID  string
		body       *bytes.Buffer
		setup      func(*mocks.MockReservaService)
		wantStatus int
	}{
		{
			name:      "sucesso",
			reservaID: "1",
			body:      jsonBody(map[string]any{"data_viagem": "2026-08-01", "sentido": "volta"}),
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(sampleReserva(), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido",
			reservaID:  "abc",
			body:       jsonBody(map[string]any{}),
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "body malformado",
			reservaID:  "1",
			body:       bytes.NewBufferString("not-json"),
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "não encontrado → 404",
			reservaID: "99",
			body:      jsonBody(map[string]any{"data_viagem": "2026-08-01", "sentido": "volta"}),
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Update(mock.Anything, int64(99), anyUpdateFunc).Return(nil, reservas.ErrReservaNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "turno inválido → 422",
			reservaID: "1",
			body:      jsonBody(map[string]any{"turno": "XX"}),
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Update(mock.Anything, int64(1), anyUpdateFunc).Return(nil, reservas.ErrTurnoInvalido)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockReservaService(t)
			tc.setup(svc)
			h := reservas.NewReservaHandler(svc)
			req := httptest.NewRequest(http.MethodPatch, "/reservas/"+tc.reservaID, tc.body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			newRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// --- Cancel ---

func TestHandler_Cancel(t *testing.T) {
	tests := []struct {
		name       string
		reservaID  string
		setup      func(*mocks.MockReservaService)
		wantStatus int
	}{
		{
			name:      "sucesso",
			reservaID: "1",
			setup: func(svc *mocks.MockReservaService) {
				r := sampleReserva()
				r.Status = reservas.StatusCancelada
				svc.EXPECT().Cancel(mock.Anything, int64(1)).Return(r, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id inválido",
			reservaID:  "abc",
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "não encontrado → 404",
			reservaID: "99",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Cancel(mock.Anything, int64(99)).Return(nil, reservas.ErrReservaNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "erro interno → 500",
			reservaID: "1",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Cancel(mock.Anything, int64(1)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockReservaService(t)
			tc.setup(svc)
			h := reservas.NewReservaHandler(svc)
			req := httptest.NewRequest(http.MethodPost, "/reservas/"+tc.reservaID+"/cancel", nil)
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsKey, &auth.Claims{
				UserID: 1,
				Role:   auth.RoleAdmin,
			}))
			rr := httptest.NewRecorder()
			newRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

// --- Delete ---

func TestHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		reservaID  string
		setup      func(*mocks.MockReservaService)
		wantStatus int
	}{
		{
			name:       "sucesso → 204",
			reservaID:  "1",
			setup:      func(svc *mocks.MockReservaService) { svc.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "id inválido",
			reservaID:  "abc",
			setup:      func(_ *mocks.MockReservaService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "não encontrado → 404",
			reservaID: "99",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Delete(mock.Anything, int64(99)).Return(reservas.ErrReservaNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "erro interno → 500",
			reservaID: "1",
			setup: func(svc *mocks.MockReservaService) {
				svc.EXPECT().Delete(mock.Anything, int64(1)).Return(errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockReservaService(t)
			tc.setup(svc)
			h := reservas.NewReservaHandler(svc)
			req := httptest.NewRequest(http.MethodDelete, "/reservas/"+tc.reservaID, nil)
			rr := httptest.NewRecorder()
			newRouter(h).ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}
