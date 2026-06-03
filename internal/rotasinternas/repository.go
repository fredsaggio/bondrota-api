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

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rotas, err := collectRotas(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if len(rotas) == 0 {
		return nil, ErrNotFound
	}

	return &rotas[0], nil
}

func (s *rotaInternaStore) List(ctx context.Context) ([]RotaInterna, error) {
	const op = "db/rotaInternaStore.List"

	const q = `
		SELECT
			r.id, r.cidade,
			p.id, p.rota_interna_id, p.nome, p.latitude, p.longitude, p.ordem
		FROM rotas_internas r
		LEFT JOIN rota_interna_paradas p ON p.rota_interna_id = r.id
		ORDER BY r.id DESC, p.ordem ASC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rotas, err := collectRotas(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return rotas, nil
}

func (s *rotaInternaStore) ListByCity(ctx context.Context, cidade string) ([]RotaInterna, error) {
	const op = "db/rotaInternaStore.ListByCity"

	const q = `
		SELECT
			r.id, r.cidade,
			p.id, p.rota_interna_id, p.nome, p.latitude, p.longitude, p.ordem
		FROM rotas_internas r
		LEFT JOIN rota_interna_paradas p ON p.rota_interna_id = r.id
		WHERE r.cidade = @cidade
		ORDER BY r.id DESC, p.ordem ASC
	`
	args := pgx.StrictNamedArgs{"cidade": cidade}

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rotas, err := collectRotas(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return rotas, nil
}

func (s *rotaInternaStore) UpdateParadas(ctx context.Context, id int64, input UpdateParadasInput) (*RotaInterna, error) {
	const op = "db/rotaInternaStore.UpdateParadas"

	var rota RotaInterna

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const selectQ = `
			SELECT id, cidade
			FROM rotas_internas
			WHERE id = @id
			FOR UPDATE
		`
		err := tx.QueryRow(ctx, selectQ, pgx.StrictNamedArgs{"id": id}).Scan(&rota.ID, &rota.Cidade)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("select rota: %w", err)
		}

		const deleteQ = `DELETE FROM rota_interna_paradas WHERE rota_interna_id = @rota_interna_id`
		if _, err := tx.Exec(ctx, deleteQ, pgx.StrictNamedArgs{"rota_interna_id": id}); err != nil {
			return fmt.Errorf("delete paradas: %w", err)
		}

		paradas, err := insertParadas(ctx, tx, id, input.Paradas)
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

func (s *rotaInternaStore) Delete(ctx context.Context, id int64) error {
	const op = "db/rotaInternaStore.Delete"

	const q = `DELETE FROM rotas_internas WHERE id = @id`

	cmdTag, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{"id": id})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func collectRotas(rows pgx.Rows) ([]RotaInterna, error) {
	defer rows.Close()

	var rotas []RotaInterna
	index := map[int64]int{}

	for rows.Next() {
		var (
			rid     int64
			cidade  string
			pID     *int64
			pRotaID *int64
			pNome   *string
			pLat    *float64
			pLng    *float64
			pOrdem  *int
		)
		if err := rows.Scan(&rid, &cidade, &pID, &pRotaID, &pNome, &pLat, &pLng, &pOrdem); err != nil {
			return nil, err
		}
		if _, ok := index[rid]; !ok {
			rotas = append(rotas, RotaInterna{ID: rid, Cidade: cidade, Paradas: []Parada{}})
			index[rid] = len(rotas) - 1
		}
		if pID != nil {
			i := index[rid]
			rotas[i].Paradas = append(rotas[i].Paradas, Parada{
				ID:            *pID,
				RotaInternaID: *pRotaID,
				Nome:          *pNome,
				Latitude:      *pLat,
				Longitude:     *pLng,
				Ordem:         *pOrdem,
			})
		}
	}

	return rotas, rows.Err()
}

func insertParadas(ctx context.Context, tx pgx.Tx, rotaID int64, paradas []ParadaInput) ([]Parada, error) {
	const q = `
		INSERT INTO rota_interna_paradas (rota_interna_id, nome, latitude, longitude, ordem)
		VALUES (@rota_interna_id, @nome, @latitude, @longitude, @ordem)
		RETURNING id, rota_interna_id, nome, latitude, longitude, ordem
	`

	batch := &pgx.Batch{}
	for _, p := range paradas {
		batch.Queue(q, pgx.StrictNamedArgs{
			"rota_interna_id": rotaID,
			"nome":            p.Nome,
			"latitude":        p.Latitude,
			"longitude":       p.Longitude,
			"ordem":           p.Ordem,
		})
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	inserted := make([]Parada, 0, len(paradas))
	for range paradas {
		var parada Parada
		err := results.QueryRow().Scan(
			&parada.ID, &parada.RotaInternaID, &parada.Nome,
			&parada.Latitude, &parada.Longitude, &parada.Ordem,
		)
		if err != nil {
			return nil, fmt.Errorf("insert parada: %w", err)
		}
		inserted = append(inserted, parada)
	}

	return inserted, nil
}
