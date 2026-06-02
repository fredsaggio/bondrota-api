package pontos

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("ponto not found")

type Ponto struct {
	ID        int64   `json:"id"`
	Nome      string  `json:"nome"`
	Rua       string  `json:"rua"`
	Cidade    string  `json:"cidade"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type PontoInput struct {
	Nome      string  `json:"nome"`
	Rua       string  `json:"rua"`
	Cidade    string  `json:"cidade"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
type PontoStore interface {
	Create(ctx context.Context, input PontoInput) (*Ponto, error)
	GetByID(ctx context.Context, id int64) (*Ponto, error)
	Update(ctx context.Context, id int64, updateFunc func(*Ponto) (bool, error)) (*Ponto, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]Ponto, error)
	ListByCity(ctx context.Context, cidade string) ([]Ponto, error)
}
