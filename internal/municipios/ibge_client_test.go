package municipios

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestIBGEClientListByUF(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/estados/AL/municipios" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`[{"id":2700300,"nome":"Arapiraca"},{"id":2704302,"nome":"Maceio"}]`)),
			Request:    r,
		}, nil
	})}

	client := NewIBGEClient(httpClient, "https://ibge.test")
	items, err := client.List(context.Background(), "al")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 || items[0].CodigoIBGE != 2700300 || items[1].UF != "AL" {
		t.Fatalf("unexpected municipios: %+v", items)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
