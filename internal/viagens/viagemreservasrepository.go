package viagens

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type viagemReservaStore struct {
	db db.DB
}

func NewViagemReservaStore(db db.DB) ViagemReservaStore {
	return &viagemReservaStore{db: db}
}

func (s *viagemReservaStore) CreateViagemReserva(ctx context.Context, input ViagemReservaInput) (*ViagemReserva, error) {
	const op = "db/viagemReservaStore.CreateViagemReserva"

	const q = `
		INSERT INTO viagem_reservas (viagem_id, reserva_id)
		VALUES (@viagem_id, @reserva_id)
		RETURNING id, viagem_id, reserva_id, status_presenca, created_at, updated_at
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"viagem_id":  input.ViagemID,
		"reserva_id": input.ReservaID,
	})
	if err != nil {
		if isViagemReservaAlreadyAllocated(err) {
			return nil, fmt.Errorf("%s: %w", op, brerror.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	viagemReserva, err := pgx.CollectExactlyOneRow(rows, scanViagemReserva)
	if err != nil {
		if isViagemReservaAlreadyAllocated(err) {
			return nil, fmt.Errorf("%s: %w", op, brerror.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &viagemReserva, nil
}

func (s *viagemReservaStore) ListReservasByViagem(ctx context.Context, viagemID int64) ([]ViagemReservaComReserva, error) {
	const op = "db/viagemReservaStore.ListReservasByViagem"

	const q = `
		SELECT
			vr.id, vr.viagem_id, vr.reserva_id, vr.status_presenca, vr.created_at, vr.updated_at,
			r.cliente_id, r.vinculo_id, r.data_viagem, r.turno, r.destino_id,
			r.rota_interna_id, r.cidade, r.sentido
		FROM viagem_reservas vr
		JOIN reservas r ON r.id = vr.reserva_id
		WHERE vr.viagem_id = @viagem_id
		ORDER BY r.destino_id ASC, r.cliente_id ASC, vr.id ASC
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"viagem_id": viagemID})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reservas, err := pgx.CollectRows(rows, scanViagemReservaComReserva)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if reservas == nil {
		return []ViagemReservaComReserva{}, nil
	}

	return reservas, nil
}

func (s *viagemReservaStore) UpdatePresenca(ctx context.Context, viagemID, reservaID int64, updateFunc func(*ViagemReserva) (bool, error)) (*ViagemReserva, error) {
	const op = "db/viagemReservaStore.UpdatePresenca"

	var viagemReserva ViagemReserva

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		current, err := getViagemReservaByViagemAndReservaForUpdate(ctx, tx, viagemID, reservaID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return brerror.ErrNotFound
			}
			return fmt.Errorf("select viagem reserva: %w", err)
		}
		viagemReserva = *current
		oldStatus := viagemReserva.StatusPresenca

		changed, err := updateFunc(&viagemReserva)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const q = `
			UPDATE viagem_reservas
			SET status_presenca = @status_presenca
			WHERE id = @id
			RETURNING id, viagem_id, reserva_id, status_presenca, created_at, updated_at
		`

		rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
			"id":              viagemReserva.ID,
			"status_presenca": viagemReserva.StatusPresenca,
		})
		if err != nil {
			return fmt.Errorf("update viagem reserva: %w", err)
		}

		viagemReserva, err = pgx.CollectExactlyOneRow(rows, scanViagemReserva)
		if err != nil {
			return fmt.Errorf("update viagem reserva: %w", err)
		}

		if oldStatus == StatusPresencaAguardando && viagemReserva.StatusPresenca != StatusPresencaAguardando {
			const registroQ = `
				INSERT INTO viagem_reserva_confirmacoes (viagem_reserva_id, registro_presenca)
				VALUES (@viagem_reserva_id, NOW())
			`

			if _, err := tx.Exec(ctx, registroQ, pgx.StrictNamedArgs{
				"viagem_reserva_id": viagemReserva.ID,
			}); err != nil {
				if isRegistroPresencaAlreadyRegistered(err) {
					return brerror.ErrAlreadyExists
				}
				return fmt.Errorf("insert registro presenca: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &viagemReserva, nil
}

func getViagemReservaByViagemAndReservaForUpdate(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, viagemID, reservaID int64) (*ViagemReserva, error) {
	const q = `
		SELECT id, viagem_id, reserva_id, status_presenca, created_at, updated_at
		FROM viagem_reservas
		WHERE viagem_id = @viagem_id
			AND reserva_id = @reserva_id
		FOR UPDATE
	`

	return scanViagemReservaByViagemAndReserva(ctx, querier, q, viagemID, reservaID)
}

func scanViagemReservaByViagemAndReserva(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, q string, viagemID, reservaID int64) (*ViagemReserva, error) {
	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{
		"viagem_id":  viagemID,
		"reserva_id": reservaID,
	})
	if err != nil {
		return nil, err
	}

	viagemReserva, err := pgx.CollectExactlyOneRow(rows, scanViagemReserva)
	if err != nil {
		return nil, err
	}

	return &viagemReserva, nil
}

func scanViagemReserva(row pgx.CollectableRow) (ViagemReserva, error) {
	var viagemReserva ViagemReserva
	err := row.Scan(
		&viagemReserva.ID,
		&viagemReserva.ViagemID,
		&viagemReserva.ReservaID,
		&viagemReserva.StatusPresenca,
		&viagemReserva.CreatedAt,
		&viagemReserva.UpdatedAt,
	)
	return viagemReserva, err
}

func scanViagemReservaComReserva(row pgx.CollectableRow) (ViagemReservaComReserva, error) {
	var data ViagemReservaComReserva
	err := row.Scan(
		&data.ID,
		&data.ViagemID,
		&data.ReservaID,
		&data.StatusPresenca,
		&data.CreatedAt,
		&data.UpdatedAt,
		&data.ClienteID,
		&data.VinculoID,
		&data.DataViagem,
		&data.Turno,
		&data.DestinoID,
		&data.RotaInternaID,
		&data.Cidade,
		&data.Sentido,
	)
	return data, err
}

func isViagemReservaAlreadyAllocated(err error) bool {
	return db.IsUniqueViolation(err, "uq_viagem_reservas_viagem_reserva") ||
		db.IsUniqueViolation(err, "uq_viagem_reservas_reserva_ativa")
}

func isRegistroPresencaAlreadyRegistered(err error) bool {
	return db.IsUniqueViolation(err, "viagem_reserva_confirmacoes_pkey")
}
