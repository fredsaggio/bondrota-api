package paradas

import (
	"context"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type paradaStore struct {
	db db.DB
}

func NewParadaStore(db db.DB) ParadaStore {
	return &paradaStore{db: db}
}

func (s *paradaStore) Create(ctx context.Context, input ParadaInput) (*Parada, error) {
	const op = "db/paradaStore.Create"

	const q = `
		INSERT INTO paradas (nome, latitude, longitude, cidade)
		VALUES (@nome, @latitude, @longitude, @cidade)
		RETURNING id, nome, latitude, longitude, cidade
	`
	args := pgx.StrictNamedArgs{
		"nome":      input.Nome,
		"latitude":  input.Latitude,
		"longitude": input.Longitude,
		"cidade":    input.Cidade,
	}

	var p Parada
	err := s.db.QueryRow(ctx, q, args).Scan(&p.ID, &p.Nome, &p.Latitude, &p.Longitude, &p.Cidade)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}