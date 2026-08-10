package paradas

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("parada not found")

type Parada struct {
	ID        int64
	Nome      string
	Latitude  float64
	Longitude float64
}

type ParadaInput struct {
	Nome      string
	Latitude  float64
	Longitude float64
}

type ParadaStore interface {
	Create(ctx context.Context, input ParadaInput) (*Parada, error)
	GetByID(ctx context.Context, paradaID int64) (*Parada, error)
	List(ctx context.Context) ([]Parada, error)
	Update(ctx context.Context, paradaID int64, updateFunc func(*Parada) (bool, error)) (*Parada, error)
	Delete(ctx context.Context, paradaID int64) error
}
