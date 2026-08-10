package municipios

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const DefaultIBGEBaseURL = "https://servicodados.ibge.gov.br/api/v1/localidades"

type IBGEClient struct {
	httpClient *http.Client
	baseURL    string
}

type ibgeEstado struct {
	Sigla string `json:"sigla"`
}

type ibgeMunicipio struct {
	ID   int64  `json:"id"`
	Nome string `json:"nome"`
}

func NewIBGEClient(httpClient *http.Client, baseURL string) *IBGEClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultIBGEBaseURL
	}
	return &IBGEClient{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *IBGEClient) List(ctx context.Context, onlyUF string) ([]Municipio, error) {
	ufs := []string{strings.ToUpper(strings.TrimSpace(onlyUF))}
	if ufs[0] == "" {
		estados, err := getJSON[[]ibgeEstado](ctx, c, "/estados?orderBy=nome")
		if err != nil {
			return nil, fmt.Errorf("list estados: %w", err)
		}
		ufs = make([]string, 0, len(estados))
		for _, estado := range estados {
			ufs = append(ufs, estado.Sigla)
		}
	}

	result := make([]Municipio, 0)
	for _, uf := range ufs {
		path := "/estados/" + url.PathEscape(uf) + "/municipios?orderBy=nome"
		items, err := getJSON[[]ibgeMunicipio](ctx, c, path)
		if err != nil {
			return nil, fmt.Errorf("list municipios for %s: %w", uf, err)
		}
		for _, item := range items {
			result = append(result, Municipio{
				CodigoIBGE: item.ID,
				Nome:       item.Nome,
				UF:         uf,
				Ativo:      true,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].UF != result[j].UF {
			return result[i].UF < result[j].UF
		}
		return result[i].Nome < result[j].Nome
	})
	return result, nil
}

func getJSON[T any](ctx context.Context, client *IBGEClient, path string) (T, error) {
	var result T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+path, nil)
	if err != nil {
		return result, err
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return result, fmt.Errorf("IBGE returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return result, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}
