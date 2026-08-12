package municipios

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeStore struct {
	listByUFFn func(ctx context.Context, uf string) ([]Municipio, error)
	getByIDFn  func(ctx context.Context, codigoIBGE int64) (*Municipio, error)
}

func (s fakeStore) ListByUF(ctx context.Context, uf string) ([]Municipio, error) {
	return s.listByUFFn(ctx, uf)
}

func (s fakeStore) GetByID(ctx context.Context, codigoIBGE int64) (*Municipio, error) {
	return s.getByIDFn(ctx, codigoIBGE)
}

func (s fakeStore) Upsert(context.Context, []Municipio) error {
	return nil
}

// routeWithParam simula o chi preenchendo o parâmetro de path, já que os testes
// chamam o handler diretamente em vez de subir o roteador inteiro.
func routeWithParam(method, target, param, value string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(param, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestHandlerListByUF(t *testing.T) {
	t.Run("normalizes UF and returns municipalities", func(t *testing.T) {
		handler := NewHandler(fakeStore{listByUFFn: func(_ context.Context, uf string) ([]Municipio, error) {
			if uf != "AL" {
				t.Fatalf("unexpected uf: %s", uf)
			}
			return []Municipio{{CodigoIBGE: 2704302, Nome: "Maceio", UF: "AL"}}, nil
		}})
		req := httptest.NewRequest(http.MethodGet, "/municipios?uf=al", nil)
		response := httptest.NewRecorder()

		handler.ListByUF(response, req)

		if response.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
		}
	})

	t.Run("rejects invalid UF", func(t *testing.T) {
		handler := NewHandler(fakeStore{listByUFFn: func(context.Context, string) ([]Municipio, error) {
			t.Fatal("store should not be called")
			return nil, nil
		}})
		response := httptest.NewRecorder()

		handler.ListByUF(response, httptest.NewRequest(http.MethodGet, "/municipios?uf=Alagoas", nil))

		if response.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", response.Code)
		}
	})

	t.Run("handles store error", func(t *testing.T) {
		handler := NewHandler(fakeStore{listByUFFn: func(context.Context, string) ([]Municipio, error) {
			return nil, errors.New("db")
		}})
		response := httptest.NewRecorder()

		handler.ListByUF(response, httptest.NewRequest(http.MethodGet, "/municipios?uf=AL", nil))

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", response.Code)
		}
	})
}

func TestHandlerGetByID(t *testing.T) {
	t.Run("returns the municipio", func(t *testing.T) {
		handler := NewHandler(fakeStore{getByIDFn: func(_ context.Context, codigoIBGE int64) (*Municipio, error) {
			if codigoIBGE != 2704302 {
				t.Fatalf("unexpected codigo_ibge: %d", codigoIBGE)
			}
			return &Municipio{CodigoIBGE: 2704302, Nome: "Maceio", UF: "AL"}, nil
		}})
		req := routeWithParam(http.MethodGet, "/municipios/2704302", "codigoIBGE", "2704302")
		response := httptest.NewRecorder()

		handler.GetByID(response, req)

		if response.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
		}
	})

	t.Run("rejects a non-numeric id", func(t *testing.T) {
		handler := NewHandler(fakeStore{getByIDFn: func(context.Context, int64) (*Municipio, error) {
			t.Fatal("store should not be called")
			return nil, nil
		}})
		req := routeWithParam(http.MethodGet, "/municipios/abc", "codigoIBGE", "abc")
		response := httptest.NewRecorder()

		handler.GetByID(response, req)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", response.Code)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		handler := NewHandler(fakeStore{getByIDFn: func(context.Context, int64) (*Municipio, error) {
			return nil, ErrNotFound
		}})
		req := routeWithParam(http.MethodGet, "/municipios/999", "codigoIBGE", "999")
		response := httptest.NewRecorder()

		handler.GetByID(response, req)

		if response.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", response.Code)
		}
	})

	t.Run("handles store error", func(t *testing.T) {
		handler := NewHandler(fakeStore{getByIDFn: func(context.Context, int64) (*Municipio, error) {
			return nil, errors.New("db")
		}})
		req := routeWithParam(http.MethodGet, "/municipios/2704302", "codigoIBGE", "2704302")
		response := httptest.NewRecorder()

		handler.GetByID(response, req)

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", response.Code)
		}
	})
}
