package destinos

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type destinoStore struct {
	db db.DB
}

func NewDestinoStore(db db.DB) DestinoStore {
	return &destinoStore{db: db}
}

func (s *destinoStore) Create(ctx context.Context, input DestinoInput) (*Destino, error) {
	const op = "db/destinoStore.Create"
	var p Destino

	const q = `
		INSERT INTO destinos (nome, rua, municipio_id, latitude, longitude)
		VALUES (@nome, @rua, @municipio_id, @latitude, @longitude)
		RETURNING id, nome, rua, municipio_id, latitude, longitude
	`

	args := pgx.StrictNamedArgs{
		"nome":         input.Nome,
		"rua":          input.Rua,
		"municipio_id": input.MunicipioID,
		"latitude":     input.Latitude,
		"longitude":    input.Longitude,
	}

	err := s.db.QueryRow(ctx, q, args).Scan(&p.ID, &p.Nome, &p.Rua, &p.MunicipioID, &p.Latitude, &p.Longitude)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &p, nil
}

func (s *destinoStore) GetByID(ctx context.Context, id int64) (*Destino, error) {
	const op = "db/destinoStore.GetByID"

	destino, err := getDestinoByID(ctx, s.db, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return destino, nil
}

func getDestinoByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, id int64) (*Destino, error) {
	const q = `
		SELECT id, nome, rua, municipio_id, latitude, longitude
		FROM destinos
		WHERE id = @id
	`
	args := pgx.StrictNamedArgs{"id": id}

	rows, err := querier.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}

	destino, err := pgx.CollectExactlyOneRow(rows, scanDestino)
	if err != nil {
		return nil, err
	}

	return &destino, nil
}

func (s *destinoStore) List(ctx context.Context) ([]Destino, error) {
	const op = "db/destinoStore.List"

	const q = `
		SELECT id, nome, rua, municipio_id, latitude, longitude
		FROM destinos
		ORDER BY id DESC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	destinos, err := pgx.CollectRows(rows, scanDestino)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return destinos, nil
}

func (s *destinoStore) ListByMunicipio(ctx context.Context, municipioID int64) ([]Destino, error) {
	const op = "db/destinoStore.ListByMunicipio"

	const q = `
		SELECT id, nome, rua, municipio_id, latitude, longitude
		FROM destinos
		WHERE municipio_id = @municipio_id
		ORDER BY nome ASC
	`
	args := pgx.StrictNamedArgs{"municipio_id": municipioID}

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	destinos, err := pgx.CollectRows(rows, scanDestino)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return destinos, nil
}

func (s *destinoStore) Update(ctx context.Context, id int64, updateFunc func(*Destino) (bool, error)) (*Destino, error) {
	const op = "db/destinoStore.Update"

	var destino Destino

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		p, err := getDestinoByIDForUpdate(ctx, tx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("select: %w", err)
		}
		destino = *p

		changed, err := updateFunc(&destino)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const updateQ = `
			UPDATE destinos
			SET nome = @nome, rua = @rua, municipio_id = @municipio_id,
			    latitude = @latitude, longitude = @longitude
			WHERE id = @id
		`
		updateArgs := pgx.StrictNamedArgs{
			"id":           destino.ID,
			"nome":         destino.Nome,
			"rua":          destino.Rua,
			"municipio_id": destino.MunicipioID,
			"latitude":     destino.Latitude,
			"longitude":    destino.Longitude,
		}

		if _, err := tx.Exec(ctx, updateQ, updateArgs); err != nil {
			return fmt.Errorf("update: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &destino, nil
}

func getDestinoByIDForUpdate(ctx context.Context, tx pgx.Tx, id int64) (*Destino, error) {
	const q = `
		SELECT id, nome, rua, municipio_id, latitude, longitude
		FROM destinos
		WHERE id = @id
		FOR UPDATE
	`
	args := pgx.StrictNamedArgs{"id": id}

	rows, err := tx.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}

	destino, err := pgx.CollectExactlyOneRow(rows, scanDestino)
	if err != nil {
		return nil, err
	}

	return &destino, nil
}

func (s *destinoStore) Delete(ctx context.Context, id int64) error {
	const op = "db/destinoStore.Delete"

	const q = `
		DELETE FROM destinos
		WHERE id = @id
	`
	args := pgx.StrictNamedArgs{"id": id}

	cmdTag, err := s.db.Exec(ctx, q, args)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func scanDestino(row pgx.CollectableRow) (Destino, error) {
	var p Destino
	err := row.Scan(&p.ID, &p.Nome, &p.Rua, &p.MunicipioID, &p.Latitude, &p.Longitude)
	return p, err
}
