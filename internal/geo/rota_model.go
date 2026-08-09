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

type MatrizCustos struct {
	DistanciasMetros [][]float64
	DuracoesSegundos [][]float64
}

type Roteador interface {
	CalcularRota(ctx context.Context, coordenadas []Coordenada) (*RotaCalculada, error)
	CalcularMatriz(ctx context.Context, coordenadas []Coordenada) (*MatrizCustos, error)
}
