package rotasinternas

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("rota interna not found")
var ErrOrdemDuplicada = errors.New("ordens das paradas devem ser únicas")
var ErrSemParadas = errors.New("rota deve ter ao menos uma parada") // Adicionar
var ErrParadaInvalida = errors.New("parada_id e ordem devem ser maiores que zero")

type RotaInterna struct {
	ID      int64
	Paradas []ParadaOrdenada
}

type ParadaOrdenada struct {
	ID        int64
	Nome      string
	Latitude  float64
	Longitude float64
	Ordem     int
}

type ParadaInput struct {
	ParadaID int64
	Ordem    int
}

type CreateRotaInternaInput struct {
	Paradas []ParadaInput
}

type UpdateParadasInput struct {
	Paradas []ParadaInput
}

type RotaInternaStore interface {
	Create(ctx context.Context, input CreateRotaInternaInput) (*RotaInterna, error)
	GetByID(ctx context.Context, rotaInternaID int64) (*RotaInterna, error)
	List(ctx context.Context) ([]RotaInterna, error)
	UpdateParadas(ctx context.Context, rotaInternaID int64, input UpdateParadasInput) (*RotaInterna, error)
	Delete(ctx context.Context, rotaInternaID int64) error
}

type RotaInternaService interface {
	Create(ctx context.Context, input CreateRotaInternaInput) (*RotaInterna, error)
	GetByID(ctx context.Context, rotaInternaID int64) (*RotaInterna, error)
	List(ctx context.Context) ([]RotaInterna, error)
	UpdateParadas(ctx context.Context, rotaInternaID int64, input UpdateParadasInput) (*RotaInterna, error)
	Delete(ctx context.Context, rotaInternaID int64) error
}
