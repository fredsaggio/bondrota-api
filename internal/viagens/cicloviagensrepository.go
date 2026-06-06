package viagens

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type cicloViagemStore struct {
	db db.DB
}

func NewCicloViagemStore(db db.DB) CicloViagemStore {
	return &cicloViagemStore{db: db}
}

func (s *cicloViagemStore) CreateCiclo(ctx context.Context, input CicloViagemInput) (*CicloViagem, error) {
	const op = "db/cicloViagemStore.CreateCiclo"

	const q = `
		INSERT INTO ciclos_viagem (
			data_viagem, turno, cidade, rota_interna_id, veiculo_id, motorista_id, expires_at
		)
		VALUES (
			@data_viagem, @turno, @cidade, @rota_interna_id, @veiculo_id, @motorista_id, @expires_at
		)
		RETURNING
			id, data_viagem, turno, cidade, rota_interna_id, veiculo_id, motorista_id,
			status, expires_at, created_at, updated_at
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":     input.DataViagem,
		"turno":           input.Turno,
		"cidade":          input.Cidade,
		"rota_interna_id": input.RotaInternaID,
		"veiculo_id":      input.VeiculoID,
		"motorista_id":    input.MotoristaID,
		"expires_at":      input.ExpiresAt,
	})
	if err != nil {
		if isCicloAlreadyAllocated(err) {
			return nil, fmt.Errorf("%s: %w", op, brerror.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	ciclo, err := pgx.CollectExactlyOneRow(rows, scanCicloViagem)
	if err != nil {
		if isCicloAlreadyAllocated(err) {
			return nil, fmt.Errorf("%s: %w", op, brerror.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &ciclo, nil
}

func (s *cicloViagemStore) GetCicloByID(ctx context.Context, cicloID int64) (*CicloViagem, error) {
	const op = "db/cicloViagemStore.GetCicloByID"

	ciclo, err := getCicloViagemByID(ctx, s.db, cicloID, false)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, brerror.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return ciclo, nil
}

func (s *cicloViagemStore) ListCiclos(ctx context.Context) ([]CicloViagem, error) {
	const op = "db/cicloViagemStore.ListCiclos"

	const q = `
		SELECT
			id, data_viagem, turno, cidade, rota_interna_id, veiculo_id, motorista_id,
			status, expires_at, created_at, updated_at
		FROM ciclos_viagem
		ORDER BY data_viagem DESC, turno ASC, id DESC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	ciclos, err := pgx.CollectRows(rows, scanCicloViagem)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if ciclos == nil {
		return []CicloViagem{}, nil
	}

	return ciclos, nil
}

func (s *cicloViagemStore) UpdateCiclo(ctx context.Context, cicloID int64, updateFunc func(*CicloViagem) (bool, error)) (*CicloViagem, error) {
	const op = "db/cicloViagemStore.UpdateCiclo"

	var ciclo CicloViagem

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		current, err := getCicloViagemByID(ctx, tx, cicloID, true)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return brerror.ErrNotFound
			}
			return fmt.Errorf("select ciclo viagem: %w", err)
		}
		ciclo = *current

		changed, err := updateFunc(&ciclo)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const q = `
			UPDATE ciclos_viagem
			SET data_viagem = @data_viagem,
				turno = @turno,
				cidade = @cidade,
				rota_interna_id = @rota_interna_id,
				veiculo_id = @veiculo_id,
				motorista_id = @motorista_id,
				status = @status,
				expires_at = @expires_at
			WHERE id = @id
			RETURNING
				id, data_viagem, turno, cidade, rota_interna_id, veiculo_id, motorista_id,
				status, expires_at, created_at, updated_at
		`

		rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
			"id":              ciclo.ID,
			"data_viagem":     ciclo.DataViagem,
			"turno":           ciclo.Turno,
			"cidade":          ciclo.Cidade,
			"rota_interna_id": ciclo.RotaInternaID,
			"veiculo_id":      ciclo.VeiculoID,
			"motorista_id":    ciclo.MotoristaID,
			"status":          ciclo.Status,
			"expires_at":      ciclo.ExpiresAt,
		})
		if err != nil {
			if isCicloAlreadyAllocated(err) {
				return brerror.ErrAlreadyExists
			}
			return fmt.Errorf("update ciclo viagem: %w", err)
		}

		ciclo, err = pgx.CollectExactlyOneRow(rows, scanCicloViagem)
		if err != nil {
			if isCicloAlreadyAllocated(err) {
				return brerror.ErrAlreadyExists
			}
			return fmt.Errorf("update ciclo viagem: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &ciclo, nil
}

func getCicloViagemByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, cicloID int64, forUpdate bool) (*CicloViagem, error) {
	q := `
		SELECT
			id, data_viagem, turno, cidade, rota_interna_id, veiculo_id, motorista_id,
			status, expires_at, created_at, updated_at
		FROM ciclos_viagem
		WHERE id = @id
	`
	if forUpdate {
		q += " FOR UPDATE"
	}

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"id": cicloID})
	if err != nil {
		return nil, err
	}

	ciclo, err := pgx.CollectExactlyOneRow(rows, scanCicloViagem)
	if err != nil {
		return nil, err
	}

	return &ciclo, nil
}

func scanCicloViagem(row pgx.CollectableRow) (CicloViagem, error) {
	var ciclo CicloViagem
	err := row.Scan(
		&ciclo.ID,
		&ciclo.DataViagem,
		&ciclo.Turno,
		&ciclo.Cidade,
		&ciclo.RotaInternaID,
		&ciclo.VeiculoID,
		&ciclo.MotoristaID,
		&ciclo.Status,
		&ciclo.ExpiresAt,
		&ciclo.CreatedAt,
		&ciclo.UpdatedAt,
	)
	return ciclo, err
}

func isCicloAlreadyAllocated(err error) bool {
	return db.IsUniqueViolation(err, "uq_ciclos_viagem_ativos_veiculo_data_turno") ||
		db.IsUniqueViolation(err, "uq_ciclos_viagem_ativos_motorista_data_turno")
}
