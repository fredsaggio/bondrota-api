package viagens

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type horarioTurnoViagemStore struct {
	db db.DB
}

func NewHorarioTurnoViagemStore(db db.DB) HorarioTurnoViagemStore {
	return &horarioTurnoViagemStore{db: db}
}

func (s *horarioTurnoViagemStore) Create(ctx context.Context, input HorarioTurnoViagemInput) (*HorarioTurnoViagem, error) {
	const op = "db/horarioTurnoViagemStore.Create"

	const q = `
		INSERT INTO horarios_turno_viagem (municipio_destino_id, turno, horario_ida, horario_volta)
		VALUES (@municipio_destino_id, @turno, @horario_ida::time, @horario_volta::time)
		RETURNING id, municipio_destino_id, turno, horario_ida::text, horario_volta::text, created_at, updated_at
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"municipio_destino_id": input.MunicipioDestinoID,
		"turno":                input.Turno,
		"horario_ida":          formatHorarioTurno(input.HorarioIda),
		"horario_volta":        formatHorarioTurno(input.HorarioVolta),
	})
	if err != nil {
		if isHorarioTurnoAlreadyExists(err) {
			return nil, fmt.Errorf("%s: %w", op, brerror.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	horario, err := pgx.CollectExactlyOneRow(rows, scanHorarioTurnoViagem)
	if err != nil {
		if isHorarioTurnoAlreadyExists(err) {
			return nil, fmt.Errorf("%s: %w", op, brerror.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &horario, nil
}

func (s *horarioTurnoViagemStore) GetByID(ctx context.Context, id int64) (*HorarioTurnoViagem, error) {
	const op = "db/horarioTurnoViagemStore.GetByID"

	horario, err := getHorarioTurnoViagemByID(ctx, s.db, id, false)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, brerror.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return horario, nil
}

func (s *horarioTurnoViagemStore) GetByMunicipioDestinoTurno(ctx context.Context, municipioDestinoID int64, turno TurnoViagem) (*HorarioTurnoViagem, error) {
	const op = "db/horarioTurnoViagemStore.GetByMunicipioDestinoTurno"

	const q = `
		SELECT id, municipio_destino_id, turno, horario_ida::text, horario_volta::text, created_at, updated_at
		FROM horarios_turno_viagem
		WHERE municipio_destino_id = @municipio_destino_id
			AND turno = @turno
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"municipio_destino_id": municipioDestinoID,
		"turno":                turno,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	horario, err := pgx.CollectExactlyOneRow(rows, scanHorarioTurnoViagem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, brerror.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &horario, nil
}

func (s *horarioTurnoViagemStore) List(ctx context.Context) ([]HorarioTurnoViagem, error) {
	const op = "db/horarioTurnoViagemStore.List"

	const q = `
		SELECT id, municipio_destino_id, turno, horario_ida::text, horario_volta::text, created_at, updated_at
		FROM horarios_turno_viagem
		ORDER BY municipio_destino_id ASC, turno ASC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	horarios, err := pgx.CollectRows(rows, scanHorarioTurnoViagem)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if horarios == nil {
		return []HorarioTurnoViagem{}, nil
	}

	return horarios, nil
}

func (s *horarioTurnoViagemStore) Update(ctx context.Context, id int64, updateFunc func(*HorarioTurnoViagem) (bool, error)) (*HorarioTurnoViagem, error) {
	const op = "db/horarioTurnoViagemStore.Update"

	var horario HorarioTurnoViagem

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		current, err := getHorarioTurnoViagemByID(ctx, tx, id, true)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return brerror.ErrNotFound
			}
			return fmt.Errorf("select horario turno viagem: %w", err)
		}
		horario = *current

		changed, err := updateFunc(&horario)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const q = `
			UPDATE horarios_turno_viagem
			SET municipio_destino_id = @municipio_destino_id,
				turno = @turno,
				horario_ida = @horario_ida::time,
				horario_volta = @horario_volta::time
			WHERE id = @id
			RETURNING id, municipio_destino_id, turno, horario_ida::text, horario_volta::text, created_at, updated_at
		`

		rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
			"id":                   horario.ID,
			"municipio_destino_id": horario.MunicipioDestinoID,
			"turno":                horario.Turno,
			"horario_ida":          formatHorarioTurno(horario.HorarioIda),
			"horario_volta":        formatHorarioTurno(horario.HorarioVolta),
		})
		if err != nil {
			if isHorarioTurnoAlreadyExists(err) {
				return brerror.ErrAlreadyExists
			}
			return fmt.Errorf("update horario turno viagem: %w", err)
		}

		horario, err = pgx.CollectExactlyOneRow(rows, scanHorarioTurnoViagem)
		if err != nil {
			if isHorarioTurnoAlreadyExists(err) {
				return brerror.ErrAlreadyExists
			}
			return fmt.Errorf("update horario turno viagem: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &horario, nil
}

func (s *horarioTurnoViagemStore) Delete(ctx context.Context, id int64) error {
	const op = "db/horarioTurnoViagemStore.Delete"

	const q = `DELETE FROM horarios_turno_viagem WHERE id = @id`

	cmdTag, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{"id": id})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return brerror.ErrNotFound
	}

	return nil
}

func getHorarioTurnoViagemByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, id int64, forUpdate bool) (*HorarioTurnoViagem, error) {
	q := `
		SELECT id, municipio_destino_id, turno, horario_ida::text, horario_volta::text, created_at, updated_at
		FROM horarios_turno_viagem
		WHERE id = @id
	`
	if forUpdate {
		q += " FOR UPDATE"
	}

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"id": id})
	if err != nil {
		return nil, err
	}

	horario, err := pgx.CollectExactlyOneRow(rows, scanHorarioTurnoViagem)
	if err != nil {
		return nil, err
	}

	return &horario, nil
}

func scanHorarioTurnoViagem(row pgx.CollectableRow) (HorarioTurnoViagem, error) {
	var horario HorarioTurnoViagem
	var horarioIda string
	var horarioVolta string

	err := row.Scan(
		&horario.ID,
		&horario.MunicipioDestinoID,
		&horario.Turno,
		&horarioIda,
		&horarioVolta,
		&horario.CreatedAt,
		&horario.UpdatedAt,
	)
	if err != nil {
		return HorarioTurnoViagem{}, err
	}

	horario.HorarioIda, err = parseHorarioTurno(horarioIda)
	if err != nil {
		return HorarioTurnoViagem{}, err
	}
	horario.HorarioVolta, err = parseHorarioTurno(horarioVolta)
	if err != nil {
		return HorarioTurnoViagem{}, err
	}

	return horario, nil
}

func parseHorarioTurno(value string) (time.Duration, error) {
	layouts := []string{"15:04:05.999999", "15:04:05", "15:04"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return time.Duration(parsed.Hour())*time.Hour +
				time.Duration(parsed.Minute())*time.Minute +
				time.Duration(parsed.Second())*time.Second +
				time.Duration(parsed.Nanosecond()), nil
		}
	}
	return 0, fmt.Errorf("invalid horario: %s", value)
}

func formatHorarioTurno(horario time.Duration) string {
	totalSeconds := int64(horario.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func isHorarioTurnoAlreadyExists(err error) bool {
	return db.IsUniqueViolation(err, "uq_horarios_turno_viagem_municipio_destino_turno")
}
