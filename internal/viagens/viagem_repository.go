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

type viagemStore struct {
	db db.DB
}

func NewViagemStore(db db.DB) ViagemStore {
	return &viagemStore{db: db}
}

func (s *viagemStore) CreateViagem(ctx context.Context, input ViagemInput) (*Viagem, error) {
	const op = "db/viagemStore.CreateViagem"

	var viagem Viagem

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const q = `
			INSERT INTO viagens (ciclo_viagem_id, sentido)
			VALUES (@ciclo_viagem_id, @sentido)
			RETURNING id, ciclo_viagem_id, sentido, status, created_at, updated_at
		`

		rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
			"ciclo_viagem_id": input.CicloViagemID,
			"sentido":         input.Sentido,
		})
		if err != nil {
			if isViagemAlreadyCreated(err) {
				return brerror.ErrAlreadyExists
			}
			return fmt.Errorf("insert viagem: %w", err)
		}

		viagem, err = pgx.CollectExactlyOneRow(rows, scanViagem)
		if err != nil {
			if isViagemAlreadyCreated(err) {
				return brerror.ErrAlreadyExists
			}
			return fmt.Errorf("insert viagem: %w", err)
		}

		const horarioQ = `
			INSERT INTO viagem_horarios (viagem_id, tipo, horario)
			VALUES (@viagem_id, @tipo, @horario)
		`

		if _, err := tx.Exec(ctx, horarioQ, pgx.StrictNamedArgs{
			"viagem_id": viagem.ID,
			"tipo":      TipoHorarioPartidaPrevista,
			"horario":   input.PartidaPrevista,
		}); err != nil {
			return fmt.Errorf("insert partida prevista: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &viagem, nil
}

func (s *viagemStore) GetViagemByID(ctx context.Context, viagemID int64) (*ViagemComCiclo, error) {
	const op = "db/viagemStore.GetViagemByID"

	viagem, err := getViagemComCicloByID(ctx, s.db, viagemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, brerror.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return viagem, nil
}

func (s *viagemStore) ListViagens(ctx context.Context) ([]ViagemComCiclo, error) {
	const op = "db/viagemStore.ListViagens"

	const q = `
		SELECT
			v.id, v.ciclo_viagem_id, v.sentido, v.status, v.created_at, v.updated_at,
			c.id, c.data_viagem, c.turno, c.rota_interna_id,
			c.veiculo_id, c.motorista_id, c.status, c.expires_at, c.created_at, c.updated_at
		FROM viagens v
		JOIN ciclos_viagem c ON c.id = v.ciclo_viagem_id
		ORDER BY c.data_viagem DESC, c.turno ASC, v.sentido ASC, v.id DESC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	viagens, err := pgx.CollectRows(rows, scanViagemComCiclo)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if viagens == nil {
		return []ViagemComCiclo{}, nil
	}

	return viagens, nil
}

func (s *viagemStore) ListViagensByCiclo(ctx context.Context, cicloID int64) ([]Viagem, error) {
	const op = "db/viagemStore.ListViagensByCiclo"

	const q = `
		SELECT
			id, ciclo_viagem_id, sentido, status, created_at, updated_at
		FROM viagens
		WHERE ciclo_viagem_id = @ciclo_viagem_id
		ORDER BY sentido ASC, id ASC
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"ciclo_viagem_id": cicloID})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	viagens, err := pgx.CollectRows(rows, scanViagem)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if viagens == nil {
		return []Viagem{}, nil
	}

	return viagens, nil
}

func (s *viagemStore) ListHorariosByViagem(ctx context.Context, viagemID int64) ([]ViagemHorario, error) {
	const op = "db/viagemStore.ListHorariosByViagem"

	const q = `
		SELECT id, viagem_id, tipo, horario, created_at, updated_at
		FROM viagem_horarios
		WHERE viagem_id = @viagem_id
		ORDER BY created_at ASC, id ASC
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"viagem_id": viagemID})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	horarios, err := pgx.CollectRows(rows, scanViagemHorario)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if horarios == nil {
		return []ViagemHorario{}, nil
	}

	return horarios, nil
}

func (s *viagemStore) RegistrarHorarioViagem(ctx context.Context, viagemID int64, tipo TipoHorarioViagem, horario time.Time) (*ViagemHorario, error) {
	const op = "db/viagemStore.RegistrarHorarioViagem"

	const q = `
		INSERT INTO viagem_horarios (viagem_id, tipo, horario)
		VALUES (@viagem_id, @tipo, @horario)
		RETURNING id, viagem_id, tipo, horario, created_at, updated_at
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"viagem_id": viagemID,
		"tipo":      tipo,
		"horario":   horario,
	})
	if err != nil {
		if isHorarioViagemAlreadyRegistered(err) {
			return nil, fmt.Errorf("%s: %w", op, brerror.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	viagemHorario, err := pgx.CollectExactlyOneRow(rows, scanViagemHorario)
	if err != nil {
		if isHorarioViagemAlreadyRegistered(err) {
			return nil, fmt.Errorf("%s: %w", op, brerror.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &viagemHorario, nil
}

func (s *viagemStore) AtualizarStatusERegistrarHorarioViagem(ctx context.Context, viagemID int64, from StatusViagem, to StatusViagem, tipo TipoHorarioViagem, horario time.Time) (*Viagem, error) {
	const op = "db/viagemStore.AtualizarStatusERegistrarHorarioViagem"

	var viagem Viagem

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		current, err := getViagemByID(ctx, tx, viagemID, true)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return brerror.ErrNotFound
			}
			return fmt.Errorf("select viagem: %w", err)
		}
		if current.Status != from {
			return fmt.Errorf("%w: status da viagem nao permite transicao", brerror.ErrAlreadyExists)
		}

		const updateQ = `
			UPDATE viagens
			SET status = @status
			WHERE id = @id
			RETURNING id, ciclo_viagem_id, sentido, status, created_at, updated_at
		`

		rows, err := tx.Query(ctx, updateQ, pgx.StrictNamedArgs{
			"id":     current.ID,
			"status": to,
		})
		if err != nil {
			return fmt.Errorf("update viagem: %w", err)
		}

		viagem, err = pgx.CollectExactlyOneRow(rows, scanViagem)
		if err != nil {
			return fmt.Errorf("update viagem: %w", err)
		}

		const horarioQ = `
			INSERT INTO viagem_horarios (viagem_id, tipo, horario)
			VALUES (@viagem_id, @tipo, @horario)
		`

		if _, err := tx.Exec(ctx, horarioQ, pgx.StrictNamedArgs{
			"viagem_id": current.ID,
			"tipo":      tipo,
			"horario":   horario,
		}); err != nil {
			if isHorarioViagemAlreadyRegistered(err) {
				return brerror.ErrAlreadyExists
			}
			return fmt.Errorf("insert horario viagem: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &viagem, nil
}

func (s *viagemStore) UpdateViagem(ctx context.Context, viagemID int64, updateFunc func(*Viagem) (bool, error)) (*Viagem, error) {
	const op = "db/viagemStore.UpdateViagem"

	var viagem Viagem

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		current, err := getViagemByID(ctx, tx, viagemID, true)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return brerror.ErrNotFound
			}
			return fmt.Errorf("select viagem: %w", err)
		}
		viagem = *current

		changed, err := updateFunc(&viagem)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const q = `
			UPDATE viagens
			SET status = @status
			WHERE id = @id
			RETURNING id, ciclo_viagem_id, sentido, status, created_at, updated_at
		`

		rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
			"id":     viagem.ID,
			"status": viagem.Status,
		})
		if err != nil {
			return fmt.Errorf("update viagem: %w", err)
		}

		viagem, err = pgx.CollectExactlyOneRow(rows, scanViagem)
		if err != nil {
			return fmt.Errorf("update viagem: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &viagem, nil
}

func getViagemByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, viagemID int64, forUpdate bool) (*Viagem, error) {
	q := `
		SELECT
			id, ciclo_viagem_id, sentido, status, created_at, updated_at
		FROM viagens
		WHERE id = @id
	`
	if forUpdate {
		q += " FOR UPDATE"
	}

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"id": viagemID})
	if err != nil {
		return nil, err
	}

	viagem, err := pgx.CollectExactlyOneRow(rows, scanViagem)
	if err != nil {
		return nil, err
	}

	return &viagem, nil
}

func getViagemComCicloByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, viagemID int64) (*ViagemComCiclo, error) {
	const q = `
		SELECT
			v.id, v.ciclo_viagem_id, v.sentido, v.status, v.created_at, v.updated_at,
			c.id, c.data_viagem, c.turno, c.rota_interna_id,
			c.veiculo_id, c.motorista_id, c.status, c.expires_at, c.created_at, c.updated_at
		FROM viagens v
		JOIN ciclos_viagem c ON c.id = v.ciclo_viagem_id
		WHERE v.id = @id
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"id": viagemID})
	if err != nil {
		return nil, err
	}

	viagem, err := pgx.CollectExactlyOneRow(rows, scanViagemComCiclo)
	if err != nil {
		return nil, err
	}

	return &viagem, nil
}

func scanViagem(row pgx.CollectableRow) (Viagem, error) {
	var viagem Viagem
	err := row.Scan(
		&viagem.ID,
		&viagem.CicloViagemID,
		&viagem.Sentido,
		&viagem.Status,
		&viagem.CreatedAt,
		&viagem.UpdatedAt,
	)
	return viagem, err
}

func scanViagemHorario(row pgx.CollectableRow) (ViagemHorario, error) {
	var horario ViagemHorario
	err := row.Scan(
		&horario.ID,
		&horario.ViagemID,
		&horario.Tipo,
		&horario.Horario,
		&horario.CreatedAt,
		&horario.UpdatedAt,
	)
	return horario, err
}

func scanViagemComCiclo(row pgx.CollectableRow) (ViagemComCiclo, error) {
	var data ViagemComCiclo
	err := row.Scan(
		&data.Viagem.ID,
		&data.Viagem.CicloViagemID,
		&data.Viagem.Sentido,
		&data.Viagem.Status,
		&data.Viagem.CreatedAt,
		&data.Viagem.UpdatedAt,
		&data.Ciclo.ID,
		&data.Ciclo.DataViagem,
		&data.Ciclo.Turno,
		&data.Ciclo.RotaInternaID,
		&data.Ciclo.VeiculoID,
		&data.Ciclo.MotoristaID,
		&data.Ciclo.Status,
		&data.Ciclo.ExpiresAt,
		&data.Ciclo.CreatedAt,
		&data.Ciclo.UpdatedAt,
	)
	return data, err
}

func isViagemAlreadyCreated(err error) bool {
	return db.IsUniqueViolation(err, "uq_viagens_ciclo_sentido")
}

func isHorarioViagemAlreadyRegistered(err error) bool {
	return db.IsUniqueViolation(err, "uq_viagem_horarios_viagem_tipo")
}
