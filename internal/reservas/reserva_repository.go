package reservas

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type reservaStore struct {
	db db.DB
}

func NewReservaStore(db db.DB) ReservaStore {
	return &reservaStore{db: db}
}

func (s *reservaStore) Create(ctx context.Context, input ReservaInput) (*Reserva, error) {
	const op = "db/reservaStore.Create"

	const q = `
		INSERT INTO reservas (
			cliente_id, vinculo_id, data_viagem, turno, destino_id, rota_interna_id, sentido
		)
		VALUES (
			@cliente_id, @vinculo_id, @data_viagem, @turno, @destino_id, @rota_interna_id, @sentido
		)
		RETURNING
			id, cliente_id, vinculo_id, data_viagem, turno, destino_id, rota_interna_id,
			sentido, status, created_at, updated_at
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"cliente_id":      input.ClienteID,
		"vinculo_id":      input.VinculoID,
		"data_viagem":     input.DataViagem,
		"turno":           input.Turno,
		"destino_id":      input.DestinoID,
		"rota_interna_id": input.RotaInternaID,
		"sentido":         input.Sentido,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reserva, err := pgx.CollectExactlyOneRow(rows, scanReserva)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &reserva, nil
}

func (s *reservaStore) GetByID(ctx context.Context, reservaID int64) (*Reserva, error) {
	const op = "db/reservaStore.GetByID"

	reserva, err := getReservaByID(ctx, s.db, reservaID, false)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReservaNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return reserva, nil
}

func (s *reservaStore) List(ctx context.Context) ([]Reserva, error) {
	const op = "db/reservaStore.List"

	const q = `
		SELECT
			id, cliente_id, vinculo_id, data_viagem, turno, destino_id, rota_interna_id,
			sentido, status, created_at, updated_at
		FROM reservas
		ORDER BY data_viagem DESC, id DESC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reservas, err := pgx.CollectRows(rows, scanReserva)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return reservas, nil
}

func (s *reservaStore) ListByCliente(ctx context.Context, clienteID int64) ([]Reserva, error) {
	const op = "db/reservaStore.ListByCliente"

	const q = `
		SELECT
			id, cliente_id, vinculo_id, data_viagem, turno, destino_id, rota_interna_id,
			sentido, status, created_at, updated_at
		FROM reservas
		WHERE cliente_id = @cliente_id
		ORDER BY data_viagem DESC, id DESC
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"cliente_id": clienteID})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reservas, err := pgx.CollectRows(rows, scanReserva)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if reservas == nil {
		return []Reserva{}, nil
	}

	return reservas, nil
}

func (s *reservaStore) ListByVinculo(ctx context.Context, clienteID, vinculoID int64) ([]Reserva, error) {
	const op = "db/reservaStore.ListByVinculo"

	const q = `
		SELECT
			id, cliente_id, vinculo_id, data_viagem, turno, destino_id, rota_interna_id,
			sentido, status, created_at, updated_at
		FROM reservas
		WHERE cliente_id = @cliente_id
			AND vinculo_id = @vinculo_id
		ORDER BY data_viagem DESC, id DESC
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"cliente_id": clienteID,
		"vinculo_id": vinculoID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reservas, err := pgx.CollectRows(rows, scanReserva)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if reservas == nil {
		return []Reserva{}, nil
	}

	return reservas, nil
}

func (s *reservaStore) Update(ctx context.Context, reservaID int64, updateFunc func(*Reserva) (bool, error)) (*Reserva, error) {
	const op = "db/reservaStore.Update"

	var reserva Reserva

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		current, err := getReservaByID(ctx, tx, reservaID, true)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReservaNotFound
			}
			return fmt.Errorf("select reserva: %w", err)
		}
		reserva = *current

		changed, err := updateFunc(&reserva)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const q = `
			UPDATE reservas
			SET data_viagem = @data_viagem,
				turno = @turno,
				sentido = @sentido,
				status = @status
			WHERE id = @id
			RETURNING
				id, cliente_id, vinculo_id, data_viagem, turno, destino_id, rota_interna_id,
				sentido, status, created_at, updated_at
		`

		rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
			"id":          reserva.ID,
			"data_viagem": reserva.DataViagem,
			"turno":       reserva.Turno,
			"sentido":     reserva.Sentido,
			"status":      reserva.Status,
		})
		if err != nil {
			return fmt.Errorf("update reserva: %w", err)
		}

		reserva, err = pgx.CollectExactlyOneRow(rows, scanReserva)
		if err != nil {
			return fmt.Errorf("update reserva: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &reserva, nil
}

func (s *reservaStore) GetVinculoSnapshot(ctx context.Context, vinculoID int64) (VinculoSnapshot, error) {
	const op = "db/reservaStore.GetVinculoSnapshot"

	snapshot, err := getVinculoSnapshot(ctx, s.db, vinculoID)
	if err != nil {
		return VinculoSnapshot{}, fmt.Errorf("%s: %w", op, err)
	}

	return snapshot, nil
}

func (s *reservaStore) Delete(ctx context.Context, reservaID int64) error {
	const op = "db/reservaStore.Delete"

	const q = `DELETE FROM reservas WHERE id = @id`

	cmdTag, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{"id": reservaID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrReservaNotFound
	}

	return nil
}

func getReservaByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, reservaID int64, forUpdate bool) (*Reserva, error) {
	q := `
		SELECT
			id, cliente_id, vinculo_id, data_viagem, turno, destino_id, rota_interna_id,
			sentido, status, created_at, updated_at
		FROM reservas
		WHERE id = @id
	`
	if forUpdate {
		q += " FOR UPDATE"
	}

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"id": reservaID})
	if err != nil {
		return nil, err
	}

	reserva, err := pgx.CollectExactlyOneRow(rows, scanReserva)
	if err != nil {
		return nil, err
	}

	return &reserva, nil
}

func getVinculoSnapshot(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, vinculoID int64) (VinculoSnapshot, error) {
	const q = `
		SELECT v.cliente_id, v.turno, v.destino_id, v.rota_interna_id
		FROM cliente_vinculos v
		WHERE v.id = @vinculo_id
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"vinculo_id": vinculoID})
	if err != nil {
		return VinculoSnapshot{}, fmt.Errorf("select vinculo snapshot: %w", err)
	}

	snapshot, err := pgx.CollectExactlyOneRow(rows, func(row pgx.CollectableRow) (VinculoSnapshot, error) {
		var s VinculoSnapshot
		err := row.Scan(&s.ClienteID, &s.Turno, &s.DestinoID, &s.RotaInternaID)
		return s, err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return VinculoSnapshot{}, ErrVinculoNotFound
		}
		return VinculoSnapshot{}, fmt.Errorf("select vinculo snapshot: %w", err)
	}

	return snapshot, nil
}

func scanReserva(row pgx.CollectableRow) (Reserva, error) {
	var r Reserva
	err := row.Scan(
		&r.ID,
		&r.ClienteID,
		&r.VinculoID,
		&r.DataViagem,
		&r.Turno,
		&r.DestinoID,
		&r.RotaInternaID,
		&r.Sentido,
		&r.Status,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	return r, err
}
