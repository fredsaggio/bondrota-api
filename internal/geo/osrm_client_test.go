package geo_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/geo"
)

func TestOSRMClient_CalcularRota(t *testing.T) {
	t.Run("calls route endpoint and parses first route", func(t *testing.T) {
		httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/route/v1/driving/-36.35,-9.78;-35.775,-9.558" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			q := r.URL.Query()
			if q.Get("overview") != "full" || q.Get("geometries") != "geojson" || q.Get("steps") != "false" {
				t.Fatalf("unexpected query: %s", r.URL.RawQuery)
			}

			return jsonResponse(http.StatusOK, `{
				"code": "Ok",
				"routes": [
					{
						"distance": 100000.4,
						"duration": 7200.5,
						"geometry": {
							"type": "LineString",
							"coordinates": [[-36.35,-9.78],[-35.775,-9.558]]
						}
					}
				]
			}`), nil
		})}

		client := geo.NewOSRMClient(httpClient, "http://osrm.test")

		rota, err := client.CalcularRota(context.Background(), []geo.Coordenada{
			{Latitude: -9.78, Longitude: -36.35},
			{Latitude: -9.558, Longitude: -35.775},
		})

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if rota.DistanciaMetros != 100000 {
			t.Fatalf("unexpected distance: %d", rota.DistanciaMetros)
		}
		if rota.DuracaoSegundos != 7201 {
			t.Fatalf("unexpected duration: %d", rota.DuracaoSegundos)
		}
		if len(rota.Geometry) == 0 {
			t.Fatal("expected geometry")
		}
	})

	t.Run("requires at least two coordinates", func(t *testing.T) {
		client := geo.NewOSRMClient(nil, "")

		_, err := client.CalcularRota(context.Background(), []geo.Coordenada{
			{Latitude: -9.78, Longitude: -36.35},
		})

		if !errors.Is(err, brerror.ErrInvalidInput) {
			t.Fatalf("expected invalid input, got %v", err)
		}
	})

	t.Run("maps empty routes to not found", func(t *testing.T) {
		httpClient := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"code":"Ok","routes":[]}`), nil
		})}

		client := geo.NewOSRMClient(httpClient, "http://osrm.test")

		_, err := client.CalcularRota(context.Background(), []geo.Coordenada{
			{Latitude: -9.78, Longitude: -36.35},
			{Latitude: -9.558, Longitude: -35.775},
		})

		if !errors.Is(err, brerror.ErrNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})

	t.Run("returns error on non ok osrm code", func(t *testing.T) {
		httpClient := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"code":"NoRoute","routes":[]}`), nil
		})}

		client := geo.NewOSRMClient(httpClient, "http://osrm.test")

		_, err := client.CalcularRota(context.Background(), []geo.Coordenada{
			{Latitude: -9.78, Longitude: -36.35},
			{Latitude: -9.558, Longitude: -35.775},
		})

		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestOSRMClient_CalcularMatriz(t *testing.T) {
	t.Run("calls table endpoint and parses costs", func(t *testing.T) {
		httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/table/v1/driving/-36.35,-9.78;-35.775,-9.558" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if r.URL.Query().Get("annotations") != "duration,distance" {
				t.Fatalf("unexpected query: %s", r.URL.RawQuery)
			}

			return jsonResponse(http.StatusOK, `{
				"code":"Ok",
				"distances":[[0,100000.4],[100500.2,0]],
				"durations":[[0,7200.5],[7300.1,0]]
			}`), nil
		})}

		client := geo.NewOSRMClient(httpClient, "http://osrm.test")
		matriz, err := client.CalcularMatriz(context.Background(), []geo.Coordenada{
			{Latitude: -9.78, Longitude: -36.35},
			{Latitude: -9.558, Longitude: -35.775},
		})

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if matriz.DistanciasMetros[0][1] != 100000.4 || matriz.DuracoesSegundos[1][0] != 7300.1 {
			t.Fatalf("unexpected matrix: %+v", matriz)
		}
	})

	t.Run("rejects unreachable coordinate pair", func(t *testing.T) {
		httpClient := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{
				"code":"Ok",
				"distances":[[0,null],[10,0]],
				"durations":[[0,null],[5,0]]
			}`), nil
		})}

		client := geo.NewOSRMClient(httpClient, "http://osrm.test")
		_, err := client.CalcularMatriz(context.Background(), []geo.Coordenada{
			{Latitude: -9.78, Longitude: -36.35},
			{Latitude: -9.558, Longitude: -35.775},
		})

		if !errors.Is(err, brerror.ErrNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
