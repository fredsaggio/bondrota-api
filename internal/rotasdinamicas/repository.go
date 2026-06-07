package rotasdinamicas

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type rotaDinamicaStore struct {
	db db.DB
}

func NewRotaDinamicaStore(db db.DB) RotaDinamicaStore {
	return &rotaDinamicaStore{db: db}
}

func (s *rotaDinamicaStore) Create(ctx context.Context, input RotaDinamicaInput) (*RotaDinamicaComDestinos, error) {
	const op = "db/rotaDinamicaStore.Create"

	var rota RotaDinamica
	var destinos []RotaDinamicaDestino

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var err error

		rota, err = insertRotaDinamica(ctx, tx, input)
		if err != nil {
			return mapRotaDinamicaDBError("insert rota dinamica", err)
		}

		destinos = make([]RotaDinamicaDestino, 0, len(input.Destinos))
		for _, destinoInput := range input.Destinos {
			destino, err := insertRotaDinamicaDestino(ctx, tx, rota.ID, destinoInput)
			if err != nil {
				return mapRotaDinamicaDBError("insert rota dinamica destino", err)
			}
			destinos = append(destinos, destino)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &RotaDinamicaComDestinos{
		Rota:     rota,
		Destinos: destinos,
	}, nil
}

func (s *rotaDinamicaStore) GetByViagem(ctx context.Context, viagemID int64) (*RotaDinamicaComDestinos, error) {
	const op = "db/rotaDinamicaStore.GetByViagem"

	rota, err := getRotaDinamicaByViagem(ctx, s.db, viagemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, brerror.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	destinos, err := s.ListDestinos(ctx, rota.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &RotaDinamicaComDestinos{
		Rota:     *rota,
		Destinos: destinos,
	}, nil
}

func (s *rotaDinamicaStore) GetExpiresAtByViagem(ctx context.Context, viagemID int64) (time.Time, error) {
	const op = "db/rotaDinamicaStore.GetExpiresAtByViagem"

	const q = `
		SELECT c.expires_at
		FROM viagens v
		JOIN ciclos_viagem c ON c.id = v.ciclo_viagem_id
		WHERE v.id = @viagem_id
	`

	var expiresAt time.Time
	if err := s.db.QueryRow(ctx, q, pgx.StrictNamedArgs{"viagem_id": viagemID}).Scan(&expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, brerror.ErrNotFound
		}
		return time.Time{}, fmt.Errorf("%s: %w", op, err)
	}

	return expiresAt, nil
}

func (s *rotaDinamicaStore) ListDestinos(ctx context.Context, rotaDinamicaID int64) ([]RotaDinamicaDestino, error) {
	const op = "db/rotaDinamicaStore.ListDestinos"

	destinos, err := listRotaDinamicaDestinos(ctx, s.db, rotaDinamicaID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return destinos, nil
}

func (s *rotaDinamicaStore) DeleteByViagem(ctx context.Context, viagemID int64) error {
	const op = "db/rotaDinamicaStore.DeleteByViagem"

	const q = `
		DELETE FROM rotas_dinamicas
		WHERE viagem_id = @viagem_id
	`

	cmdTag, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{"viagem_id": viagemID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return brerror.ErrNotFound
	}

	return nil
}

func insertRotaDinamica(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, input RotaDinamicaInput) (RotaDinamica, error) {
	const q = `
		INSERT INTO rotas_dinamicas (
			viagem_id, provider,
			origem_nome, origem_latitude, origem_longitude,
			destino_final_nome, destino_final_latitude, destino_final_longitude,
			distancia_metros, duracao_segundos, geometry, expires_at
		)
		VALUES (
			@viagem_id, @provider,
			@origem_nome, @origem_latitude, @origem_longitude,
			@destino_final_nome, @destino_final_latitude, @destino_final_longitude,
			@distancia_metros, @duracao_segundos, @geometry, @expires_at
		)
		RETURNING
			id, viagem_id, provider,
			origem_nome, origem_latitude, origem_longitude,
			destino_final_nome, destino_final_latitude, destino_final_longitude,
			distancia_metros, duracao_segundos, geometry,
			expires_at, created_at, updated_at
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{
		"viagem_id":               input.ViagemID,
		"provider":                input.Provider,
		"origem_nome":             input.Origem.Nome,
		"origem_latitude":         input.Origem.Latitude,
		"origem_longitude":        input.Origem.Longitude,
		"destino_final_nome":      input.DestinoFinal.Nome,
		"destino_final_latitude":  input.DestinoFinal.Latitude,
		"destino_final_longitude": input.DestinoFinal.Longitude,
		"distancia_metros":        input.DistanciaMetros,
		"duracao_segundos":        input.DuracaoSegundos,
		"geometry":                input.Geometry,
		"expires_at":              input.ExpiresAt,
	})
	if err != nil {
		return RotaDinamica{}, err
	}

	return pgx.CollectExactlyOneRow(rows, scanRotaDinamica)
}

func insertRotaDinamicaDestino(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, rotaDinamicaID int64, input RotaDinamicaDestinoInput) (RotaDinamicaDestino, error) {
	const q = `
		INSERT INTO rota_dinamica_destinos (rota_dinamica_id, destino_id, ordem)
		VALUES (@rota_dinamica_id, @destino_id, @ordem)
		RETURNING id, rota_dinamica_id, destino_id, ordem, created_at
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{
		"rota_dinamica_id": rotaDinamicaID,
		"destino_id":       input.DestinoID,
		"ordem":            input.Ordem,
	})
	if err != nil {
		return RotaDinamicaDestino{}, err
	}

	return pgx.CollectExactlyOneRow(rows, scanRotaDinamicaDestino)
}

func getRotaDinamicaByViagem(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, viagemID int64) (*RotaDinamica, error) {
	const q = `
		SELECT
			id, viagem_id, provider,
			origem_nome, origem_latitude, origem_longitude,
			destino_final_nome, destino_final_latitude, destino_final_longitude,
			distancia_metros, duracao_segundos, geometry,
			expires_at, created_at, updated_at
		FROM rotas_dinamicas
		WHERE viagem_id = @viagem_id
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"viagem_id": viagemID})
	if err != nil {
		return nil, err
	}

	rota, err := pgx.CollectExactlyOneRow(rows, scanRotaDinamica)
	if err != nil {
		return nil, err
	}

	return &rota, nil
}

func listRotaDinamicaDestinos(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, rotaDinamicaID int64) ([]RotaDinamicaDestino, error) {
	const q = `
		SELECT id, rota_dinamica_id, destino_id, ordem, created_at
		FROM rota_dinamica_destinos
		WHERE rota_dinamica_id = @rota_dinamica_id
		ORDER BY ordem ASC
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"rota_dinamica_id": rotaDinamicaID})
	if err != nil {
		return nil, err
	}

	destinos, err := pgx.CollectRows(rows, scanRotaDinamicaDestino)
	if err != nil {
		return nil, err
	}
	if destinos == nil {
		return []RotaDinamicaDestino{}, nil
	}

	return destinos, nil
}

func scanRotaDinamica(row pgx.CollectableRow) (RotaDinamica, error) {
	var rota RotaDinamica
	err := row.Scan(
		&rota.ID,
		&rota.ViagemID,
		&rota.Provider,
		&rota.OrigemNome,
		&rota.OrigemLatitude,
		&rota.OrigemLongitude,
		&rota.DestinoFinalNome,
		&rota.DestinoFinalLatitude,
		&rota.DestinoFinalLongitude,
		&rota.DistanciaMetros,
		&rota.DuracaoSegundos,
		&rota.Geometry,
		&rota.ExpiresAt,
		&rota.CreatedAt,
		&rota.UpdatedAt,
	)
	return rota, err
}

func scanRotaDinamicaDestino(row pgx.CollectableRow) (RotaDinamicaDestino, error) {
	var destino RotaDinamicaDestino
	err := row.Scan(
		&destino.ID,
		&destino.RotaDinamicaID,
		&destino.DestinoID,
		&destino.Ordem,
		&destino.CreatedAt,
	)
	return destino, err
}

func mapRotaDinamicaDBError(msg string, err error) error {
	if isRotaDinamicaAlreadyCreated(err) || isRotaDinamicaDestinoDuplicated(err) {
		return brerror.ErrAlreadyExists
	}
	if isRotaDinamicaMissingReference(err) {
		return brerror.ErrNotFound
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func isRotaDinamicaAlreadyCreated(err error) bool {
	return db.IsUniqueViolation(err, "rotas_dinamicas_viagem_id_key")
}

func isRotaDinamicaDestinoDuplicated(err error) bool {
	return db.IsUniqueViolation(err, "uq_rota_dinamica_destino") ||
		db.IsUniqueViolation(err, "uq_rota_dinamica_ordem")
}

func isRotaDinamicaMissingReference(err error) bool {
	return db.IsForeignKeyViolation(err, "rotas_dinamicas_viagem_id_fkey") ||
		db.IsForeignKeyViolation(err, "rota_dinamica_destinos_destino_id_fkey") ||
		db.IsForeignKeyViolation(err, "rota_dinamica_destinos_rota_dinamica_id_fkey")
}
