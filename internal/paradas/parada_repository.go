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
		INSERT INTO paradas (nome, latitude, longitude)
		VALUES (@nome, @latitude, @longitude)
		RETURNING id, nome, latitude, longitude
	`
	args := pgx.StrictNamedArgs{
		"nome":      input.Nome,
		"latitude":  input.Latitude,
		"longitude": input.Longitude,
	}

	var p Parada
	err := s.db.QueryRow(ctx, q, args).Scan(&p.ID, &p.Nome, &p.Latitude, &p.Longitude)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (s *paradaStore) GetByID(ctx context.Context, paradaID int64) (*Parada, error) {
	const op = "db/paradaStore.GetByID"

	const q = `
		SELECT id, nome, latitude, longitude
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
		SELECT id, nome, latitude, longitude
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

func (s *paradaStore) Update(ctx context.Context, paradaID int64, updateFunc func(*Parada) (bool, error)) (*Parada, error) {
	const op = "db/paradaStore.Update"

	var parada Parada

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const selectQ = `
			SELECT id, nome, latitude, longitude
			FROM paradas
			WHERE id = @id
			FOR UPDATE
		`
		rows, err := tx.Query(ctx, selectQ, pgx.StrictNamedArgs{"id": paradaID})
		if err != nil {
			return fmt.Errorf("select: %w", err)
		}

		p, err := pgx.CollectExactlyOneRow(rows, scanParada)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("select: %w", err)
		}
		parada = p

		changed, err := updateFunc(&parada)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const updateQ = `
			UPDATE paradas
			SET nome = @nome, latitude = @latitude, longitude = @longitude
			WHERE id = @id
		`
		_, err = tx.Exec(ctx, updateQ, pgx.StrictNamedArgs{
			"id":        parada.ID,
			"nome":      parada.Nome,
			"latitude":  parada.Latitude,
			"longitude": parada.Longitude,
		})
		if err != nil {
			return fmt.Errorf("update: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &parada, nil
}

func (s *paradaStore) Delete(ctx context.Context, paradaID int64) error {
	const op = "db/paradaStore.Delete"

	const q = `DELETE FROM paradas WHERE id = @id`

	cmdTag, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{"id": paradaID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func scanParada(row pgx.CollectableRow) (Parada, error) {
	var p Parada
	err := row.Scan(&p.ID, &p.Nome, &p.Latitude, &p.Longitude)
	return p, err
}
