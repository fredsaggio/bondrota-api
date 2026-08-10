package municipios

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeStore struct {
	listByUFFn func(ctx context.Context, uf string) ([]Municipio, error)
}

func (s fakeStore) ListByUF(ctx context.Context, uf string) ([]Municipio, error) {
	return s.listByUFFn(ctx, uf)
}

func (s fakeStore) Upsert(context.Context, []Municipio) error {
	return nil
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
