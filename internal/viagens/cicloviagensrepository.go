package viagens

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (s *cicloViagemStore) CreateCicloComViagens(ctx context.Context, input CicloViagemInput, partidas map[SentidoViagem]time.Time) (*CicloComViagens, error) {
	const op = "db/cicloViagemStore.CreateCicloComViagens"

	var ciclo CicloViagem
	var viagemIda Viagem
	var viagemVolta Viagem

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var err error

		ciclo, err = insertCicloViagem(ctx, tx, input)
		if err != nil {
			if isCicloAlreadyAllocated(err) {
				return brerror.ErrAlreadyExists
			}
			return fmt.Errorf("insert ciclo viagem: %w", err)
		}

		viagemIda, err = insertViagemComPartida(ctx, tx, ViagemInput{
			CicloViagemID:   ciclo.ID,
			Sentido:         SentidoIda,
			PartidaPrevista: partidas[SentidoIda],
		})
		if err != nil {
			return fmt.Errorf("insert viagem ida: %w", err)
		}
		if err := insertViagemReservasConfirmadas(ctx, tx, viagemIda.ID, input, SentidoIda); err != nil {
			return fmt.Errorf("insert reservas ida: %w", err)
		}

		viagemVolta, err = insertViagemComPartida(ctx, tx, ViagemInput{
			CicloViagemID:   ciclo.ID,
			Sentido:         SentidoVolta,
			PartidaPrevista: partidas[SentidoVolta],
		})
		if err != nil {
			return fmt.Errorf("insert viagem volta: %w", err)
		}
		if err := insertViagemReservasConfirmadas(ctx, tx, viagemVolta.ID, input, SentidoVolta); err != nil {
			return fmt.Errorf("insert reservas volta: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &CicloComViagens{
		Ciclo: ciclo,
		Viagens: []Viagem{
			viagemIda,
			viagemVolta,
		},
	}, nil
}

func (s *cicloViagemStore) CreateCiclosComViagens(ctx context.Context, inputs []CicloViagemComReservasInput, partidas map[SentidoViagem]time.Time) (*PlanejamentoViagens, error) {
	const op = "db/cicloViagemStore.CreateCiclosComViagens"

	ciclos := make([]CicloComViagens, 0, len(inputs))

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		for _, input := range inputs {
			ciclo, err := insertCicloViagem(ctx, tx, input.Ciclo)
			if err != nil {
				if isCicloAlreadyAllocated(err) {
					return brerror.ErrAlreadyExists
				}
				return fmt.Errorf("insert ciclo viagem: %w", err)
			}

			viagemIda, err := insertViagemComPartida(ctx, tx, ViagemInput{
				CicloViagemID:   ciclo.ID,
				Sentido:         SentidoIda,
				PartidaPrevista: partidas[SentidoIda],
			})
			if err != nil {
				return fmt.Errorf("insert viagem ida: %w", err)
			}
			if err := insertViagemReservasByIDs(ctx, tx, viagemIda.ID, input.ReservaIDsIda); err != nil {
				return fmt.Errorf("insert reservas ida: %w", err)
			}

			viagemVolta, err := insertViagemComPartida(ctx, tx, ViagemInput{
				CicloViagemID:   ciclo.ID,
				Sentido:         SentidoVolta,
				PartidaPrevista: partidas[SentidoVolta],
			})
			if err != nil {
				return fmt.Errorf("insert viagem volta: %w", err)
			}
			if err := insertViagemReservasByIDs(ctx, tx, viagemVolta.ID, input.ReservaIDsVolta); err != nil {
				return fmt.Errorf("insert reservas volta: %w", err)
			}

			ciclos = append(ciclos, CicloComViagens{
				Ciclo: ciclo,
				Viagens: []Viagem{
					viagemIda,
					viagemVolta,
				},
			})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &PlanejamentoViagens{Ciclos: ciclos}, nil
}

func (s *cicloViagemStore) ListReservaIDsConfirmadasParaPlanejamento(ctx context.Context, filtro PlanejamentoReservasFiltro) ([]int64, error) {
	const op = "db/cicloViagemStore.ListReservaIDsConfirmadasParaPlanejamento"

	const q = `
		SELECT id
		FROM reservas
		WHERE data_viagem = @data_viagem
			AND turno = @turno
			AND cidade = @cidade
			AND rota_interna_id = @rota_interna_id
			AND sentido = @sentido
			AND status = 'confirmada'
		ORDER BY id
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":     filtro.DataViagem,
		"turno":           filtro.Turno,
		"cidade":          filtro.Cidade,
		"rota_interna_id": filtro.RotaInternaID,
		"sentido":         filtro.Sentido,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reservaIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if reservaIDs == nil {
		return []int64{}, nil
	}

	return reservaIDs, nil
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

func insertCicloViagem(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, input CicloViagemInput) (CicloViagem, error) {
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

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":     input.DataViagem,
		"turno":           input.Turno,
		"cidade":          input.Cidade,
		"rota_interna_id": input.RotaInternaID,
		"veiculo_id":      input.VeiculoID,
		"motorista_id":    input.MotoristaID,
		"expires_at":      input.ExpiresAt,
	})
	if err != nil {
		return CicloViagem{}, err
	}

	return pgx.CollectExactlyOneRow(rows, scanCicloViagem)
}

func insertViagemComPartida(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}, input ViagemInput) (Viagem, error) {
	const q = `
		INSERT INTO viagens (ciclo_viagem_id, sentido)
		VALUES (@ciclo_viagem_id, @sentido)
		RETURNING id, ciclo_viagem_id, sentido, status, created_at, updated_at
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{
		"ciclo_viagem_id": input.CicloViagemID,
		"sentido":         input.Sentido,
	})
	if err != nil {
		if isViagemAlreadyCreated(err) {
			return Viagem{}, brerror.ErrAlreadyExists
		}
		return Viagem{}, err
	}

	viagem, err := pgx.CollectExactlyOneRow(rows, scanViagem)
	if err != nil {
		if isViagemAlreadyCreated(err) {
			return Viagem{}, brerror.ErrAlreadyExists
		}
		return Viagem{}, err
	}

	const horarioQ = `
		INSERT INTO viagem_horarios (viagem_id, tipo, horario)
		VALUES (@viagem_id, @tipo, @horario)
	`

	if _, err := querier.Exec(ctx, horarioQ, pgx.StrictNamedArgs{
		"viagem_id": viagem.ID,
		"tipo":      TipoHorarioPartidaPrevista,
		"horario":   input.PartidaPrevista,
	}); err != nil {
		if isHorarioViagemAlreadyRegistered(err) {
			return Viagem{}, brerror.ErrAlreadyExists
		}
		return Viagem{}, err
	}

	return viagem, nil
}

func insertViagemReservasConfirmadas(ctx context.Context, querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}, viagemID int64, input CicloViagemInput, sentido SentidoViagem) error {
	const q = `
		INSERT INTO viagem_reservas (viagem_id, reserva_id)
		SELECT @viagem_id, id
		FROM reservas
		WHERE data_viagem = @data_viagem
			AND turno = @turno
			AND cidade = @cidade
			AND rota_interna_id = @rota_interna_id
			AND sentido = @sentido
			AND status = 'confirmada'
	`

	if _, err := querier.Exec(ctx, q, pgx.StrictNamedArgs{
		"viagem_id":       viagemID,
		"data_viagem":     input.DataViagem,
		"turno":           input.Turno,
		"cidade":          input.Cidade,
		"rota_interna_id": input.RotaInternaID,
		"sentido":         sentido,
	}); err != nil {
		if isViagemReservaAlreadyAllocated(err) {
			return brerror.ErrAlreadyExists
		}
		return err
	}

	return nil
}

func insertViagemReservasByIDs(ctx context.Context, querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}, viagemID int64, reservaIDs []int64) error {
	if len(reservaIDs) == 0 {
		return nil
	}

	const q = `
		INSERT INTO viagem_reservas (viagem_id, reserva_id)
		SELECT @viagem_id, UNNEST(@reserva_ids::BIGINT[])
	`

	if _, err := querier.Exec(ctx, q, pgx.StrictNamedArgs{
		"viagem_id":   viagemID,
		"reserva_ids": reservaIDs,
	}); err != nil {
		if isViagemReservaAlreadyAllocated(err) {
			return brerror.ErrAlreadyExists
		}
		return err
	}

	return nil
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
