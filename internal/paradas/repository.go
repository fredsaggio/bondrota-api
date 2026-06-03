package paradas

import (
	"context"
	"errors"
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

func (s *paradaStore) GetByID(ctx context.Context, paradaID int64) (*Parada, error) {
	const op = "db/paradaStore.GetByID"

	const q = `
		SELECT id, nome, latitude, longitude, cidade
		FROM paradas
		WHERE id = @id
	`
	args := pgx.StrictNamedArgs{"id": paradaID}

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	p, err := pgx.CollectExactlyOneRow(rows, scanParada)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (s *paradaStore) List(ctx context.Context) ([]Parada, error) {
	const op = "db/paradaStore.List"

	const q = `
		SELECT id, nome, latitude, longitude, cidade
		FROM paradas
		ORDER BY id DESC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	paradas, err := pgx.CollectRows(rows, scanParada)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return paradas, nil
}

func (s *paradaStore) ListByCity(ctx context.Context, cidade string) ([]Parada, error) {
	const op = "db/paradaStore.ListByCity"

	const q = `
		SELECT id, nome, latitude, longitude, cidade
		FROM paradas
		WHERE cidade = @cidade
		ORDER BY nome ASC
	`
	args := pgx.StrictNamedArgs{"cidade": cidade}

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	paradas, err := pgx.CollectRows(rows, scanParada)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return paradas, nil
}

func scanParada(row pgx.CollectableRow) (Parada, error) {
	var p Parada
	err := row.Scan(&p.ID, &p.Nome, &p.Latitude, &p.Longitude, &p.Cidade)
	return p, err
}