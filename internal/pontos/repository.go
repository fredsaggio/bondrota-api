package pontos

import (
	"context"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type pontoStore struct {
	db db.DB
}

func NewPontoStore(db db.DB) PontoStore {
	return &pontoStore{db: db}
}

func (s *pontoStore) Create(ctx context.Context, input PontoInput) (*Ponto, error) {
	const op = "db/pontoStore.Create"
	var p Ponto

	const q = `
		INSERT INTO pontos (nome, rua, cidade, latitude, longitude)
		VALUES (@nome, @rua, @cidade, @latitude, @longitude)
		RETURNING id, nome, rua, cidade, latitude, longitude
	`

	args := pgx.StrictNamedArgs{
		"nome":      input.Nome,
		"rua":       input.Rua,
		"cidade":    input.Cidade,
		"latitude":  input.Latitude,
		"longitude": input.Longitude,
	}

	err := s.db.QueryRow(ctx, q, args).Scan(&p.ID, &p.Nome, &p.Rua, &p.Cidade, &p.Latitude, &p.Longitude)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &p, nil
}
