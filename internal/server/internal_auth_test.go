package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/go-chi/chi/v5"
)

type processadorPlanejamentoStub struct{}

func (processadorPlanejamentoStub) Processar(context.Context) (viagens.ResumoProcessamentoPlanejamento, error) {
	return viagens.ResumoProcessamentoPlanejamento{Concluidos: 1}, nil
}

func TestRequireBearerSecret(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	tests := []struct {
		name       string
		configured string
		header     string
		wantStatus int
	}{
		{name: "valid", configured: secret, header: "Bearer " + secret, wantStatus: http.StatusNoContent},
		{name: "missing header", configured: secret, wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", configured: secret, header: "Basic " + secret, wantStatus: http.StatusUnauthorized},
		{name: "wrong secret", configured: secret, header: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "empty configured secret", header: "Bearer ", wantStatus: http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			handler := requireBearerSecret(tc.configured)(next)
			req := httptest.NewRequest(http.MethodPost, "/internal/planejamentos/processar", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

func TestConfigDoesNotExposePlanningCronSecret(t *testing.T) {
	data, err := json.Marshal(Config{BaseCity: "Campo Alegre", PlanningCronSecret: "secret"})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if strings.Contains(string(data), "secret") {
		t.Fatalf("config exposed cron secret: %s", data)
	}
}

func TestInternalPlanningRouteUsesDedicatedSecret(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	processadorHandler := viagens.NewProcessadorPlanejamentoHandler(processadorPlanejamentoStub{})
	srv := NewServer(Handlers{ProcessadorHandler: processadorHandler}, nil, Config{PlanningCronSecret: secret})
	router := chi.NewRouter()
	srv.RegisterRoutes(router)

	t.Run("accepts cron secret without user JWT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/internal/planejamentos/processar", nil)
		req.Header.Set("Authorization", "Bearer "+secret)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("rejects user request without cron secret", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/internal/planejamentos/processar", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestManualPlanningRouteIsNotRegistered(t *testing.T) {
	srv := NewServer(Handlers{}, nil, Config{})
	router := chi.NewRouter()
	srv.RegisterRoutes(router)

	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method == http.MethodPost && route == "/planejamentos/viagens" {
			t.Fatalf("manual planning route must not be public")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}
}
