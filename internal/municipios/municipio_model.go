package municipios

import "context"

type Municipio struct {
	CodigoIBGE int64
	Nome       string
	UF         string
	Ativo      bool
}

type Store interface {
	ListByUF(ctx context.Context, uf string) ([]Municipio, error)
	Upsert(ctx context.Context, municipios []Municipio) error
}
