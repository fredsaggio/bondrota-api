package rotasinternas

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("rota interna not found")

type RotaInterna struct {
	ID       int64
	Cidade   string
	Paradas  []Parada
}

type Parada struct {
	ID            int64
	RotaInternaID int64
	Nome          string
	Latitude      float64
	Longitude     float64
	Ordem         int
}

type ParadaInput struct {
	Nome      string
	Latitude  float64
	Longitude float64
	Ordem     int
}

type CreateRotaInternaInput struct {
	Cidade  string
	Paradas []ParadaInput
}

type UpdateParadasInput struct {
	Paradas []ParadaInput
}

type RotaInternaStore interface {
	Create(ctx context.Context, input CreateRotaInternaInput) (*RotaInterna, error)
	GetByID(ctx context.Context, id int64) (*RotaInterna, error)
	List(ctx context.Context) ([]RotaInterna, error)
	ListByCity(ctx context.Context, cidade string) ([]RotaInterna, error)
	UpdateParadas(ctx context.Context, id int64, input UpdateParadasInput) (*RotaInterna, error)
	Delete(ctx context.Context, id int64) error
}