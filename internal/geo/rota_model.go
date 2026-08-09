package geo

import (
	"context"
	"encoding/json"
)

type Coordenada struct {
	Latitude  float64
	Longitude float64
}

type RotaCalculada struct {
	DistanciaMetros int
	DuracaoSegundos int
	Geometry        json.RawMessage
}

type Roteador interface {
	CalcularRota(ctx context.Context, coordenadas []Coordenada) (*RotaCalculada, error)
}
