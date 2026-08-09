package geo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

const defaultOSRMBaseURL = "https://router.project-osrm.org"

type OSRMClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewOSRMClient(httpClient *http.Client, baseURL string) *OSRMClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultOSRMBaseURL
	}

	return &OSRMClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *OSRMClient) CalcularRota(ctx context.Context, coordenadas []Coordenada) (*RotaCalculada, error) {
	if err := validateCoordenadas(coordenadas); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.routeURL(coordenadas), nil)
	if err != nil {
		return nil, fmt.Errorf("create osrm request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call osrm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osrm returned status %d", resp.StatusCode)
	}

	var data osrmRouteResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode osrm response: %w", err)
	}
	if data.Code != "Ok" {
		return nil, fmt.Errorf("osrm returned code %q", data.Code)
	}
	if len(data.Routes) == 0 {
		return nil, brerror.ErrNotFound
	}

	route := data.Routes[0]
	if !json.Valid(route.Geometry) {
		return nil, errors.New("osrm returned invalid geometry")
	}

	return &RotaCalculada{
		DistanciaMetros: int(math.Round(route.Distance)),
		DuracaoSegundos: int(math.Round(route.Duration)),
		Geometry:        route.Geometry,
	}, nil
}

func (c *OSRMClient) CalcularMatriz(ctx context.Context, coordenadas []Coordenada) (*MatrizCustos, error) {
	if err := validateCoordenadas(coordenadas); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.tableURL(coordenadas), nil)
	if err != nil {
		return nil, fmt.Errorf("create osrm table request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call osrm table: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osrm table returned status %d", resp.StatusCode)
	}

	var data osrmTableResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode osrm table response: %w", err)
	}
	if data.Code != "Ok" {
		return nil, fmt.Errorf("osrm table returned code %q", data.Code)
	}

	distancias, err := normalizeTable(data.Distances, len(coordenadas), "distances")
	if err != nil {
		return nil, err
	}
	duracoes, err := normalizeTable(data.Durations, len(coordenadas), "durations")
	if err != nil {
		return nil, err
	}

	return &MatrizCustos{
		DistanciasMetros: distancias,
		DuracoesSegundos: duracoes,
	}, nil
}

func (c *OSRMClient) routeURL(coordenadas []Coordenada) string {
	u, _ := url.Parse(c.baseURL)
	u.Path = "/route/v1/driving/" + formatCoordenadasOSRM(coordenadas)
	q := u.Query()
	q.Set("overview", "full")
	q.Set("geometries", "geojson")
	q.Set("steps", "false")
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *OSRMClient) tableURL(coordenadas []Coordenada) string {
	u, _ := url.Parse(c.baseURL)
	u.Path = "/table/v1/driving/" + formatCoordenadasOSRM(coordenadas)
	q := u.Query()
	q.Set("annotations", "duration,distance")
	u.RawQuery = q.Encode()
	return u.String()
}

func formatCoordenadasOSRM(coordenadas []Coordenada) string {
	parts := make([]string, 0, len(coordenadas))
	for _, coordenada := range coordenadas {
		lon := strconv.FormatFloat(coordenada.Longitude, 'f', -1, 64)
		lat := strconv.FormatFloat(coordenada.Latitude, 'f', -1, 64)
		parts = append(parts, lon+","+lat)
	}
	return strings.Join(parts, ";")
}

func validateCoordenadas(coordenadas []Coordenada) error {
	if len(coordenadas) < 2 {
		return fmt.Errorf("%w: at least two coordinates are required", brerror.ErrInvalidInput)
	}
	for _, coordenada := range coordenadas {
		if coordenada.Latitude == 0 && coordenada.Longitude == 0 {
			return fmt.Errorf("%w: coordinates are required", brerror.ErrInvalidInput)
		}
		if coordenada.Latitude < -90 || coordenada.Latitude > 90 {
			return fmt.Errorf("%w: latitude must be between -90 and 90", brerror.ErrInvalidInput)
		}
		if coordenada.Longitude < -180 || coordenada.Longitude > 180 {
			return fmt.Errorf("%w: longitude must be between -180 and 180", brerror.ErrInvalidInput)
		}
	}
	return nil
}

type osrmRouteResponse struct {
	Code   string      `json:"code"`
	Routes []osrmRoute `json:"routes"`
}

type osrmRoute struct {
	Distance float64         `json:"distance"`
	Duration float64         `json:"duration"`
	Geometry json.RawMessage `json:"geometry"`
}

type osrmTableResponse struct {
	Code      string       `json:"code"`
	Distances [][]*float64 `json:"distances"`
	Durations [][]*float64 `json:"durations"`
}

func normalizeTable(values [][]*float64, size int, field string) ([][]float64, error) {
	if len(values) != size {
		return nil, fmt.Errorf("osrm table returned invalid %s matrix", field)
	}

	result := make([][]float64, size)
	for i, row := range values {
		if len(row) != size {
			return nil, fmt.Errorf("osrm table returned invalid %s matrix", field)
		}

		result[i] = make([]float64, size)
		for j, value := range row {
			if value == nil {
				return nil, fmt.Errorf("%w: osrm table has no route from coordinate %d to %d", brerror.ErrNotFound, i, j)
			}
			result[i][j] = *value
		}
	}

	return result, nil
}
