package pontos

import (
	"context"
	"errors"
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

func (s *pontoStore) GetByID(ctx context.Context, id int64) (*Ponto, error) {
	const op = "db/pontoStore.GetByID"

	ponto, err := getPontoByID(ctx, s.db, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return ponto, nil
}

func getPontoByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, id int64) (*Ponto, error) {
	const q = `
		SELECT id, nome, rua, cidade, latitude, longitude
		FROM pontos
		WHERE id = @id
	`
	args := pgx.StrictNamedArgs{"id": id}

	rows, err := querier.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}

	ponto, err := pgx.CollectExactlyOneRow(rows, scanPonto)
	if err != nil {
		return nil, err
	}

	return &ponto, nil
}

func (s *pontoStore) List(ctx context.Context) ([]Ponto, error) {
	const op = "db/pontoStore.List"

	const q = `
		SELECT id, nome, rua, cidade, latitude, longitude
		FROM pontos
		ORDER BY id DESC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pontos, err := pgx.CollectRows(rows, scanPonto)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return pontos, nil
}

func (s *pontoStore) ListByCity(ctx context.Context, cidade string) ([]Ponto, error) {
	const op = "db/pontoStore.ListByCity"

	const q = `
		SELECT id, nome, rua, cidade, latitude, longitude
		FROM pontos
		WHERE cidade = @cidade
		ORDER BY nome ASC
	`
	args := pgx.StrictNamedArgs{"cidade": cidade}

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pontos, err := pgx.CollectRows(rows, scanPonto)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return pontos, nil
}

func (s *pontoStore) Update(ctx context.Context, id int64, updateFunc func(*Ponto) (bool, error)) (*Ponto, error) {
	const op = "db/pontoStore.Update"

	var ponto Ponto

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		p, err := getPontoByIDForUpdate(ctx, tx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("select: %w", err)
		}
		ponto = *p

		changed, err := updateFunc(&ponto)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const updateQ = `
			UPDATE pontos
			SET nome = @nome, rua = @rua, cidade = @cidade,
			    latitude = @latitude, longitude = @longitude
			WHERE id = @id
		`
		updateArgs := pgx.StrictNamedArgs{
			"id":        ponto.ID,
			"nome":      ponto.Nome,
			"rua":       ponto.Rua,
			"cidade":    ponto.Cidade,
			"latitude":  ponto.Latitude,
			"longitude": ponto.Longitude,
		}

		if _, err := tx.Exec(ctx, updateQ, updateArgs); err != nil {
			return fmt.Errorf("update: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &ponto, nil
}

func getPontoByIDForUpdate(ctx context.Context, tx pgx.Tx, id int64) (*Ponto, error) {
	const q = `
		SELECT id, nome, rua, cidade, latitude, longitude
		FROM pontos
		WHERE id = @id
		FOR UPDATE
	`
	args := pgx.StrictNamedArgs{"id": id}

	rows, err := tx.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}

	ponto, err := pgx.CollectExactlyOneRow(rows, scanPonto)
	if err != nil {
		return nil, err
	}

	return &ponto, nil
}

func (s *pontoStore) Delete(ctx context.Context, id int64) error {
	const op = "db/pontoStore.Delete"

	const q = `
		DELETE FROM pontos
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


func scanPonto(row pgx.CollectableRow) (Ponto, error) {
	var p Ponto
	err := row.Scan(&p.ID, &p.Nome, &p.Rua, &p.Cidade, &p.Latitude, &p.Longitude)
	return p, err
}