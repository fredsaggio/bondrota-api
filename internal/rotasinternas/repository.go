package rotasinternas

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type rotaInternaStore struct {
	db db.DB
}

func NewRotaInternaStore(db db.DB) RotaInternaStore {
	return &rotaInternaStore{db: db}
}

func (s *rotaInternaStore) Create(ctx context.Context, input CreateRotaInternaInput) (*RotaInterna, error) {
	const op = "db/rotaInternaStore.Create"

	var rota RotaInterna

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const q = `
			INSERT INTO rotas_internas (cidade)
			VALUES (@cidade)
			RETURNING id, cidade
		`
		args := pgx.StrictNamedArgs{"cidade": input.Cidade}

		err := tx.QueryRow(ctx, q, args).Scan(&rota.ID, &rota.Cidade)
		if err != nil {
			return fmt.Errorf("insert rota: %w", err)
		}

		paradas, err := insertParadas(ctx, tx, rota.ID, input.Paradas)
		if err != nil {
			return err
		}
		rota.Paradas = paradas

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &rota, nil
}

func (s *rotaInternaStore) GetByID(ctx context.Context, id int64) (*RotaInterna, error) {
	const op = "db/rotaInternaStore.GetByID"

	rota, err := getRotaInternaByID(ctx, s.db, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return rota, nil
}

func getRotaInternaByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, id int64) (*RotaInterna, error) {
	const q = `
		SELECT
			r.id, r.cidade,
			p.id, p.rota_interna_id, p.nome, p.latitude, p.longitude, p.ordem
		FROM rotas_internas r
		LEFT JOIN rota_interna_paradas p ON p.rota_interna_id = r.id
		WHERE r.id = @id
		ORDER BY p.ordem ASC
	`
	args := pgx.StrictNamedArgs{"id": id}

	rows, err := querier.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rota *RotaInterna
	for rows.Next() {
		var (
			rid    int64
			cidade string
			p      Parada
		)
		if err := rows.Scan(&rid, &cidade, &p.ID, &p.RotaInternaID, &p.Nome, &p.Latitude, &p.Longitude, &p.Ordem); err != nil {
			return nil, err
		}
		if rota == nil {
			rota = &RotaInterna{ID: rid, Cidade: cidade}
		}
		rota.Paradas = append(rota.Paradas, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if rota == nil {
		return nil, pgx.ErrNoRows
	}

	return rota, nil
}

func insertParadas(ctx context.Context, tx pgx.Tx, rotaID int64, paradas []ParadaInput) ([]Parada, error) {
	const q = `
		INSERT INTO rota_interna_paradas (rota_interna_id, nome, latitude, longitude, ordem)
		VALUES (@rota_interna_id, @nome, @latitude, @longitude, @ordem)
		RETURNING id, rota_interna_id, nome, latitude, longitude, ordem
	`

	result := make([]Parada, 0, len(paradas))
	for _, p := range paradas {
		args := pgx.StrictNamedArgs{
			"rota_interna_id": rotaID,
			"nome":            p.Nome,
			"latitude":        p.Latitude,
			"longitude":       p.Longitude,
			"ordem":           p.Ordem,
		}

		var parada Parada
		err := tx.QueryRow(ctx, q, args).Scan(
			&parada.ID, &parada.RotaInternaID, &parada.Nome,
			&parada.Latitude, &parada.Longitude, &parada.Ordem,
		)
		if err != nil {
			return nil, fmt.Errorf("insert parada: %w", err)
		}
		result = append(result, parada)
	}

	return result, nil
}
