package rotasinternas

import (
	"context"
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