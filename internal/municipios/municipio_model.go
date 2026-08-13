package municipios

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("Município não encontrado.")

type Municipio struct {
	CodigoIBGE int64
	Nome       string
	UF         string
	Ativo      bool
}

type Store interface {
	ListByUF(ctx context.Context, uf string) ([]Municipio, error)
	GetByID(ctx context.Context, codigoIBGE int64) (*Municipio, error)
	Upsert(ctx context.Context, municipios []Municipio) error
}
