package destinos

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("destino not found")

type Destino struct {
	ID        int64
	Nome      string
	Rua       string
	Cidade    string
	Latitude  float64
	Longitude float64
}

type DestinoInput struct {
	Nome      string  `json:"nome"`
	Rua       string  `json:"rua"`
	Cidade    string  `json:"cidade"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
type DestinoStore interface {
	Create(ctx context.Context, input DestinoInput) (*Destino, error)
	GetByID(ctx context.Context, id int64) (*Destino, error)
	Update(ctx context.Context, id int64, updateFunc func(*Destino) (bool, error)) (*Destino, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]Destino, error)
	ListByCity(ctx context.Context, cidade string) ([]Destino, error)
}
