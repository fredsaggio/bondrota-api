package viagens_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/go-chi/chi/v5"
)

type fakeViagemService struct {
	getFn      func(ctx context.Context, viagemID int64) (*viagens.ViagemComCiclo, error)
	listFn     func(ctx context.Context, params viagens.ViagemListParams) (viagens.ViagemListResult, error)
	resumoFn   func(ctx context.Context) (viagens.ViagemResumo, error)
	horariosFn func(ctx context.Context, viagemID int64) ([]viagens.ViagemHorario, error)
	iniciarFn  func(ctx context.Context, viagemID int64) (*viagens.Viagem, error)
	concluirFn func(ctx context.Context, viagemID int64) (*viagens.Viagem, error)
	cancelarFn func(ctx context.Context, viagemID int64) (*viagens.Viagem, error)
}

func (s fakeViagemService) GetByID(ctx context.Context, viagemID int64) (*viagens.ViagemComCiclo, error) {
	return s.getFn(ctx, viagemID)
}

func (s fakeViagemService) List(ctx context.Context, params viagens.ViagemListParams) (viagens.ViagemListResult, error) {
	return s.listFn(ctx, params)
}

func (s fakeViagemService) Resumo(ctx context.Context) (viagens.ViagemResumo, error) {
	return s.resumoFn(ctx)
}

func (s fakeViagemService) ListHorariosByViagem(ctx context.Context, viagemID int64) ([]viagens.ViagemHorario, error) {
	return s.horariosFn(ctx, viagemID)
}

func (s fakeViagemService) Iniciar(ctx context.Context, viagemID int64) (*viagens.Viagem, error) {
	return s.iniciarFn(ctx, viagemID)
}

func (s fakeViagemService) Concluir(ctx context.Context, viagemID int64) (*viagens.Viagem, error) {
	return s.concluirFn(ctx, viagemID)
}

func (s fakeViagemService) Cancelar(ctx context.Context, viagemID int64) (*viagens.Viagem, error) {
	return s.cancelarFn(ctx, viagemID)
}

type fakePresencaService struct {
	listReservasFn      func(ctx context.Context, viagemID int64) ([]viagens.ViagemReservaComReserva, error)
	atualizarPresencaFn func(ctx context.Context, viagemID, reservaID int64, status viagens.StatusPresencaViagem) (*viagens.ViagemReserva, error)
}

func (s fakePresencaService) ListReservasByViagem(ctx context.Context, viagemID int64) ([]viagens.ViagemReservaComReserva, error) {
	return s.listReservasFn(ctx, viagemID)
}

func (s fakePresencaService) AtualizarPresenca(ctx context.Context, viagemID, reservaID int64, status viagens.StatusPresencaViagem) (*viagens.ViagemReserva, error) {
	return s.atualizarPresencaFn(ctx, viagemID, reservaID, status)
}

func newViagemRouter(h *viagens.ViagemHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/viagens", h.List)
	r.Get("/viagens/resumo", h.Resumo)
	r.Get("/viagens/{viagemID}", h.GetByID)
	r.Post("/viagens/{viagemID}/iniciar", h.Iniciar)
	r.Post("/viagens/{viagemID}/concluir", h.Concluir)
	r.Post("/viagens/{viagemID}/cancelar", h.Cancelar)
	r.Get("/viagens/{viagemID}/horarios", h.ListHorarios)
	r.Get("/viagens/{viagemID}/reservas", h.ListReservas)
	r.Put("/viagens/{viagemID}/reservas/{reservaID}/presenca", h.AtualizarPresenca)
	return r
}

func sampleViagemComNomes() viagens.ViagemComCicloENomes {
	return viagens.ViagemComCicloENomes{
		ViagemComCiclo: sampleViagemComCiclo(),
		MunicipioNome:  "Maceio",
		VeiculoPlaca:   "ABC1D23",
	}
}

func TestViagemHandler_List(t *testing.T) {
	h := viagens.NewViagemHandler(fakeViagemService{
		listFn: func(_ context.Context, _ viagens.ViagemListParams) (viagens.ViagemListResult, error) {
			return viagens.ViagemListResult{
				Items:      []viagens.ViagemComCicloENomes{sampleViagemComNomes()},
				NextCursor: &viagens.ViagemCursor{DataViagem: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), ID: 10},
				HasMore:    true,
			}, nil
		},
	}, fakePresencaService{})

	rr := httptest.NewRecorder()
	newViagemRouter(h).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/viagens", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var response struct {
		Items []struct {
			MunicipioNome string `json:"municipio_nome"`
			VeiculoPlaca  string `json:"veiculo_placa"`
		} `json:"items"`
		NextCursor string `json:"next_cursor"`
		HasMore    bool   `json:"has_more"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].MunicipioNome != "Maceio" || response.Items[0].VeiculoPlaca != "ABC1D23" {
		t.Fatalf("nomes resolvidos nao vieram: %#v", response.Items)
	}
	if response.NextCursor == "" || !response.HasMore {
		t.Fatalf("esperava next_cursor e has_more: %#v", response)
	}
}

// O recorte por motorista tem que virar filtro da consulta. Se ele voltasse a ser
// aplicado sobre a pagina ja recortada, um motorista veria paginas incompletas
// (ou vazias) mesmo havendo mais viagens dele adiante.
func TestViagemHandler_ListPassaFiltroDeMotoristaParaAConsulta(t *testing.T) {
	var received viagens.ViagemListParams
	h := viagens.NewViagemHandler(fakeViagemService{
		listFn: func(_ context.Context, params viagens.ViagemListParams) (viagens.ViagemListResult, error) {
			received = params
			return viagens.ViagemListResult{}, nil
		},
	}, fakePresencaService{})

	req := httptest.NewRequest(http.MethodGet, "/viagens", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsKey, &auth.Claims{
		UserID: 4,
		Role:   auth.RoleMotorista,
	}))
	rr := httptest.NewRecorder()
	newViagemRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if received.MotoristaID != 4 {
		t.Fatalf("esperava MotoristaID=4 nos params, got %d", received.MotoristaID)
	}
}

func TestViagemHandler_ListAdminNaoRecebeFiltroDeMotorista(t *testing.T) {
	var received viagens.ViagemListParams
	h := viagens.NewViagemHandler(fakeViagemService{
		listFn: func(_ context.Context, params viagens.ViagemListParams) (viagens.ViagemListResult, error) {
			received = params
			return viagens.ViagemListResult{}, nil
		},
	}, fakePresencaService{})

	req := httptest.NewRequest(http.MethodGet, "/viagens", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsKey, &auth.Claims{
		UserID: 9,
		Role:   auth.RoleAdmin,
	}))
	rr := httptest.NewRecorder()
	newViagemRouter(h).ServeHTTP(rr, req)

	if received.MotoristaID != 0 {
		t.Fatalf("admin nao pode ser restringido a um motorista, got %d", received.MotoristaID)
	}
}

func TestViagemHandler_ListParams(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		check      func(t *testing.T, params viagens.ViagemListParams)
	}{
		{
			name:       "repassa q, limit e intervalo de data",
			query:      "?q=maceio&limit=10&data_inicio=2026-07-01&data_fim=2026-07-31",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, params viagens.ViagemListParams) {
				if params.Busca != "maceio" || params.Limit != 10 {
					t.Fatalf("params inesperados: %#v", params)
				}
				if params.DataInicio == nil || params.DataFim == nil {
					t.Fatal("intervalo de data nao foi repassado")
				}
			},
		},
		{name: "limit invalido", query: "?limit=abc", wantStatus: http.StatusBadRequest},
		{name: "cursor invalido", query: "?cursor=***", wantStatus: http.StatusBadRequest},
		{name: "data_inicio invalida", query: "?data_inicio=01-07-2026", wantStatus: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var received viagens.ViagemListParams
			called := false
			h := viagens.NewViagemHandler(fakeViagemService{
				listFn: func(_ context.Context, params viagens.ViagemListParams) (viagens.ViagemListResult, error) {
					called = true
					received = params
					return viagens.ViagemListResult{}, nil
				},
			}, fakePresencaService{})

			rr := httptest.NewRecorder()
			newViagemRouter(h).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/viagens"+tc.query, nil))

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
			if tc.wantStatus == http.StatusBadRequest && called {
				t.Fatal("parametro invalido nao pode chegar no service")
			}
			if tc.check != nil {
				tc.check(t, received)
			}
		})
	}
}

func TestViagemHandler_Resumo(t *testing.T) {
	h := viagens.NewViagemHandler(fakeViagemService{
		resumoFn: func(_ context.Context) (viagens.ViagemResumo, error) {
			return viagens.ViagemResumo{
				PorStatus:       map[viagens.StatusViagem]int64{viagens.StatusViagemProgramada: 3},
				PorTurno:        map[viagens.TurnoViagem]int64{viagens.TurnoNoturno: 2},
				HojeTotal:       5,
				HojeEmAndamento: 1,
				Proximas:        []viagens.ViagemComCicloENomes{sampleViagemComNomes()},
			}, nil
		},
	}, fakePresencaService{})

	rr := httptest.NewRecorder()
	newViagemRouter(h).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/viagens/resumo", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var response struct {
		PorStatus       map[string]int64 `json:"por_status"`
		PorTurno        map[string]int64 `json:"por_turno"`
		HojeTotal       int64            `json:"hoje_total"`
		HojeEmAndamento int64            `json:"hoje_em_andamento"`
		Proximas        []struct {
			MunicipioNome string `json:"municipio_nome"`
		} `json:"proximas"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.PorStatus["programada"] != 3 || response.PorTurno["NT"] != 2 {
		t.Fatalf("agregados inesperados: %#v", response)
	}
	if response.HojeTotal != 5 || response.HojeEmAndamento != 1 {
		t.Fatalf("contagem de hoje inesperada: %#v", response)
	}
	if len(response.Proximas) != 1 || response.Proximas[0].MunicipioNome != "Maceio" {
		t.Fatalf("proximas viagens inesperadas: %#v", response.Proximas)
	}
}

func TestRequireAssignedMotoristaOrAdmin(t *testing.T) {
	tests := []struct {
		name       string
		claims     *auth.Claims
		getFn      func(context.Context, int64) (*viagens.ViagemComCiclo, error)
		wantStatus int
	}{
		{
			name:   "assigned motorista",
			claims: &auth.Claims{UserID: 4, Role: auth.RoleMotorista},
			getFn: func(_ context.Context, _ int64) (*viagens.ViagemComCiclo, error) {
				v := sampleViagemComCiclo()
				return &v, nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:   "other motorista",
			claims: &auth.Claims{UserID: 5, Role: auth.RoleMotorista},
			getFn: func(_ context.Context, _ int64) (*viagens.ViagemComCiclo, error) {
				v := sampleViagemComCiclo()
				return &v, nil
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin bypass",
			claims:     &auth.Claims{UserID: 1, Role: auth.RoleAdmin},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "missing claims",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := viagens.NewViagemHandler(fakeViagemService{getFn: tc.getFn}, fakePresencaService{})
			r := chi.NewRouter()
			r.With(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tc.claims != nil {
						r = r.WithContext(context.WithValue(r.Context(), auth.ClaimsKey, tc.claims))
					}
					next.ServeHTTP(w, r)
				})
			}, h.RequireAssignedMotoristaOrAdmin).Get("/viagens/{viagemID}", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/viagens/10", nil))
			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestViagemHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		svc        fakeViagemService
		wantStatus int
	}{
		{
			name: "success",
			path: "/viagens/10",
			svc: fakeViagemService{
				getFn: func(_ context.Context, viagemID int64) (*viagens.ViagemComCiclo, error) {
					if viagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", viagemID)
					}
					v := sampleViagemComCiclo()
					return &v, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			path:       "/viagens/abc",
			svc:        fakeViagemService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			path: "/viagens/99",
			svc: fakeViagemService{
				getFn: func(_ context.Context, _ int64) (*viagens.ViagemComCiclo, error) {
					return nil, brerror.ErrNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "internal error",
			path: "/viagens/10",
			svc: fakeViagemService{
				getFn: func(_ context.Context, _ int64) (*viagens.ViagemComCiclo, error) {
					return nil, errors.New("db")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := viagens.NewViagemHandler(tc.svc, fakePresencaService{})
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			newViagemRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestViagemHandler_StatusActions(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		svc        fakeViagemService
		wantStatus int
	}{
		{
			name:   "iniciar success",
			method: http.MethodPost,
			path:   "/viagens/10/iniciar",
			svc: fakeViagemService{
				iniciarFn: func(_ context.Context, viagemID int64) (*viagens.Viagem, error) {
					if viagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", viagemID)
					}
					v := sampleViagem()
					v.Status = viagens.StatusViagemEmAndamento
					return &v, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "iniciar invalid id",
			method:     http.MethodPost,
			path:       "/viagens/abc/iniciar",
			svc:        fakeViagemService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "concluir success",
			method: http.MethodPost,
			path:   "/viagens/10/concluir",
			svc: fakeViagemService{
				concluirFn: func(_ context.Context, viagemID int64) (*viagens.Viagem, error) {
					if viagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", viagemID)
					}
					v := sampleViagem()
					v.Status = viagens.StatusViagemConcluida
					return &v, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "cancelar conflict",
			method: http.MethodPost,
			path:   "/viagens/10/cancelar",
			svc: fakeViagemService{
				cancelarFn: func(_ context.Context, _ int64) (*viagens.Viagem, error) {
					return nil, brerror.ErrAlreadyExists
				},
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := viagens.NewViagemHandler(tc.svc, fakePresencaService{})
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			newViagemRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestViagemHandler_ListHorarios(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		svc        fakeViagemService
		wantStatus int
	}{
		{
			name: "success",
			path: "/viagens/10/horarios",
			svc: fakeViagemService{
				horariosFn: func(_ context.Context, viagemID int64) ([]viagens.ViagemHorario, error) {
					if viagemID != 10 {
						t.Fatalf("unexpected viagemID: %d", viagemID)
					}
					return []viagens.ViagemHorario{
						{
							ID:        1,
							ViagemID:  10,
							Tipo:      viagens.TipoHorarioPartidaPrevista,
							Horario:   testTime(),
							CreatedAt: testTime(),
							UpdatedAt: testTime(),
						},
					}, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			path:       "/viagens/abc/horarios",
			svc:        fakeViagemService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			path: "/viagens/10/horarios",
			svc: fakeViagemService{
				horariosFn: func(_ context.Context, _ int64) ([]viagens.ViagemHorario, error) {
					return nil, errors.New("db")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := viagens.NewViagemHandler(tc.svc, fakePresencaService{})
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			newViagemRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestViagemHandler_ListReservas(t *testing.T) {
	h := viagens.NewViagemHandler(fakeViagemService{}, fakePresencaService{
		listReservasFn: func(_ context.Context, viagemID int64) ([]viagens.ViagemReservaComReserva, error) {
			if viagemID != 10 {
				t.Fatalf("unexpected viagemID: %d", viagemID)
			}
			return []viagens.ViagemReservaComReserva{sampleViagemReservaComReserva()}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/viagens/10/reservas", nil)
	rr := httptest.NewRecorder()

	newViagemRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestViagemHandler_AtualizarPresenca(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       map[string]any
		svc        fakePresencaService
		wantStatus int
	}{
		{
			name: "success",
			path: "/viagens/10/reservas/20/presenca",
			body: map[string]any{"status_presenca": "embarcou"},
			svc: fakePresencaService{
				atualizarPresencaFn: func(_ context.Context, viagemID, reservaID int64, status viagens.StatusPresencaViagem) (*viagens.ViagemReserva, error) {
					if viagemID != 10 || reservaID != 20 || status != viagens.StatusPresencaEmbarcou {
						t.Fatalf("unexpected update: %d %d %s", viagemID, reservaID, status)
					}
					vr := sampleViagemReserva()
					vr.StatusPresenca = viagens.StatusPresencaEmbarcou
					return &vr, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid ids",
			path:       "/viagens/abc/reservas/20/presenca",
			body:       map[string]any{"status_presenca": "embarcou"},
			svc:        fakePresencaService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			path:       "/viagens/10/reservas/20/presenca",
			body:       nil,
			svc:        fakePresencaService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "service not found",
			path: "/viagens/10/reservas/20/presenca",
			body: map[string]any{"status_presenca": "faltou"},
			svc: fakePresencaService{
				atualizarPresencaFn: func(_ context.Context, _, _ int64, _ viagens.StatusPresencaViagem) (*viagens.ViagemReserva, error) {
					return nil, brerror.ErrNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := viagens.NewViagemHandler(fakeViagemService{}, tc.svc)
			var req *http.Request
			if tc.body == nil {
				req = httptest.NewRequest(http.MethodPut, tc.path, bytesBuffer("{"))
			} else {
				req = httptest.NewRequest(http.MethodPut, tc.path, body(tc.body))
			}
			rr := httptest.NewRecorder()

			newViagemRouter(h).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d: %s", tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func bytesBuffer(s string) *strings.Reader {
	return strings.NewReader(s)
}
