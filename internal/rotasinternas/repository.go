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
		err := tx.QueryRow(ctx, q, pgx.StrictNamedArgs{"cidade": input.Cidade}).Scan(&rota.ID, &rota.Cidade)
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

func (s *rotaInternaStore) GetByID(ctx context.Context, rotaInternaID int64) (*RotaInterna, error) {
	const op = "db/rotaInternaStore.GetByID"

	const q = `
		SELECT
			r.id, r.cidade,
			p.id, p.nome, p.latitude, p.longitude, p.cidade,
			rip.ordem
		FROM rotas_internas r
		LEFT JOIN rota_interna_paradas rip ON rip.rota_interna_id = r.id
		LEFT JOIN paradas p ON p.id = rip.parada_id
		WHERE r.id = @id
		ORDER BY rip.ordem ASC
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"id": rotaInternaID})
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
			p.id, p.nome, p.latitude, p.longitude, p.cidade,
			rip.ordem
		FROM rotas_internas r
		LEFT JOIN rota_interna_paradas rip ON rip.rota_interna_id = r.id
		LEFT JOIN paradas p ON p.id = rip.parada_id
		ORDER BY r.id DESC, rip.ordem ASC
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
			p.id, p.nome, p.latitude, p.longitude, p.cidade,
			rip.ordem
		FROM rotas_internas r
		LEFT JOIN rota_interna_paradas rip ON rip.rota_interna_id = r.id
		LEFT JOIN paradas p ON p.id = rip.parada_id
		WHERE r.cidade = @cidade
		ORDER BY r.id DESC, rip.ordem ASC
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"cidade": cidade})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rotas, err := collectRotas(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return rotas, nil
}

func (s *rotaInternaStore) UpdateParadas(ctx context.Context, rotaInternaID int64, input UpdateParadasInput) (*RotaInterna, error) {
	const op = "db/rotaInternaStore.UpdateParadas"

	var rota RotaInterna

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const selectQ = `
			SELECT id, cidade
			FROM rotas_internas
			WHERE id = @id
			FOR UPDATE
		`
		err := tx.QueryRow(ctx, selectQ, pgx.StrictNamedArgs{"id": rotaInternaID}).Scan(&rota.ID, &rota.Cidade)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("select rota: %w", err)
		}

		const deleteQ = `DELETE FROM rota_interna_paradas WHERE rota_interna_id = @rota_interna_id`
		if _, err := tx.Exec(ctx, deleteQ, pgx.StrictNamedArgs{"rota_interna_id": rotaInternaID}); err != nil {
			return fmt.Errorf("delete paradas: %w", err)
		}

		paradas, err := insertParadas(ctx, tx, rotaInternaID, input.Paradas)
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

func (s *rotaInternaStore) Delete(ctx context.Context, rotaInternaID int64) error {
	const op = "db/rotaInternaStore.Delete"

	const q = `DELETE FROM rotas_internas WHERE id = @id`

	cmdTag, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{"id": rotaInternaID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func insertParadas(ctx context.Context, tx pgx.Tx, rotaInternaID int64, paradas []ParadaInput) ([]ParadaOrdenada, error) {
	const q = `
		WITH inserted AS (
			INSERT INTO rota_interna_paradas (rota_interna_id, parada_id, ordem)
			VALUES (@rota_interna_id, @parada_id, @ordem)
			RETURNING parada_id, ordem
		)
		SELECT i.ordem, p.id, p.nome, p.latitude, p.longitude, p.cidade
		FROM inserted i
		JOIN paradas p ON p.id = i.parada_id
	`

	batch := &pgx.Batch{}
	for _, p := range paradas {
		batch.Queue(q, pgx.StrictNamedArgs{
			"rota_interna_id": rotaInternaID,
			"parada_id":       p.ParadaID,
			"ordem":           p.Ordem,
		})
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	inserted := make([]ParadaOrdenada, 0, len(paradas))
	for range paradas {
		var po ParadaOrdenada
		err := results.QueryRow().Scan(
			&po.Ordem, &po.ID, &po.Nome, &po.Latitude, &po.Longitude, &po.Cidade,
		)
		if err != nil {
			return nil, fmt.Errorf("insert parada: %w", err)
		}
		inserted = append(inserted, po)
	}

	return inserted, nil
}

func collectRotas(rows pgx.Rows) ([]RotaInterna, error) {
	defer rows.Close()

	var rotas []RotaInterna
	index := map[int64]int{}

	for rows.Next() {
		var (
			rid    int64
			cidade string
			pID    *int64
			pNome  *string
			pLat   *float64
			pLng   *float64
			pCidade *string
			pOrdem *int
		)
		if err := rows.Scan(&rid, &cidade, &pID, &pNome, &pLat, &pLng, &pCidade, &pOrdem); err != nil {
			return nil, err
		}
		if _, ok := index[rid]; !ok {
			rotas = append(rotas, RotaInterna{ID: rid, Cidade: cidade, Paradas: []ParadaOrdenada{}})
			index[rid] = len(rotas) - 1
		}
		if pID != nil {
			i := index[rid]
			rotas[i].Paradas = append(rotas[i].Paradas, ParadaOrdenada{
				ID:        *pID,
				Nome:      *pNome,
				Latitude:  *pLat,
				Longitude: *pLng,
				Cidade:    *pCidade,
				Ordem:     *pOrdem,
			})
		}
	}

	return rotas, rows.Err()
}