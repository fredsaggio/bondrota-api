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
			data_viagem, turno, municipio_destino_id, rota_interna_id, veiculo_id, motorista_id, expires_at
		)
		VALUES (
			@data_viagem, @turno, @municipio_destino_id, @rota_interna_id, @veiculo_id, @motorista_id, @expires_at
		)
		RETURNING
			id, data_viagem, turno, municipio_destino_id, rota_interna_id, veiculo_id, motorista_id,
			status, expires_at, created_at, updated_at
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":          input.DataViagem,
		"turno":                input.Turno,
		"municipio_destino_id": input.MunicipioDestinoID,
		"rota_interna_id":      input.RotaInternaID,
		"veiculo_id":           input.VeiculoID,
		"motorista_id":         input.MotoristaID,
		"expires_at":           input.ExpiresAt,
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

func (s *cicloViagemStore) CreatePlanejamentoIda(ctx context.Context, inputs []CicloIdaComReservasInput, partida time.Time) (*PlanejamentoViagens, error) {
	const op = "db/cicloViagemStore.CreatePlanejamentoIda"

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
				PartidaPrevista: partida,
			})
			if err != nil {
				return fmt.Errorf("insert viagem ida: %w", err)
			}
			if err := insertViagemReservasByIDs(ctx, tx, viagemIda.ID, input.ReservaIDs); err != nil {
				return fmt.Errorf("insert reservas ida: %w", err)
			}

			ciclos = append(ciclos, CicloComViagens{
				Ciclo:   ciclo,
				Viagens: []Viagem{viagemIda},
			})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &PlanejamentoViagens{Sentido: SentidoIda, Ciclos: ciclos}, nil
}

func (s *cicloViagemStore) CreatePlanejamentoVolta(ctx context.Context, inputs []CicloVoltaComReservasInput, partida time.Time) (*PlanejamentoViagens, error) {
	const op = "db/cicloViagemStore.CreatePlanejamentoVolta"

	ciclos := make([]CicloComViagens, 0, len(inputs))

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		for _, input := range inputs {
			ciclo, err := getCicloViagemByID(ctx, tx, input.Ciclo.ID, true)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return brerror.ErrNotFound
				}
				return fmt.Errorf("select ciclo viagem: %w", err)
			}
			if ciclo.Status == StatusCicloCancelado {
				return brerror.ErrNotFound
			}

			viagemVolta, err := insertViagemComPartida(ctx, tx, ViagemInput{
				CicloViagemID:   ciclo.ID,
				Sentido:         SentidoVolta,
				PartidaPrevista: partida,
			})
			if err != nil {
				return fmt.Errorf("insert viagem volta: %w", err)
			}
			if err := insertViagemReservasByIDs(ctx, tx, viagemVolta.ID, input.ReservaIDs); err != nil {
				return fmt.Errorf("insert reservas volta: %w", err)
			}

			ciclos = append(ciclos, CicloComViagens{
				Ciclo:   *ciclo,
				Viagens: []Viagem{viagemVolta},
			})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &PlanejamentoViagens{Sentido: SentidoVolta, Ciclos: ciclos}, nil
}

func (s *cicloViagemStore) ListReservasConfirmadasParaPlanejamento(ctx context.Context, filtro PlanejamentoReservasFiltro) ([]PlanejamentoReserva, error) {
	const op = "db/cicloViagemStore.ListReservasConfirmadasParaPlanejamento"

	const q = `
		SELECT r.id, r.destino_id
		FROM reservas r
		JOIN destinos d ON d.id = r.destino_id
		WHERE r.data_viagem = @data_viagem
			AND r.turno = @turno
			AND d.municipio_id = @municipio_destino_id
			AND r.rota_interna_id = @rota_interna_id
			AND r.sentido = @sentido
			AND r.status = 'confirmada'
		ORDER BY r.id
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":          filtro.DataViagem,
		"turno":                filtro.Turno,
		"municipio_destino_id": filtro.MunicipioDestinoID,
		"rota_interna_id":      filtro.RotaInternaID,
		"sentido":              filtro.Sentido,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reservas, err := pgx.CollectRows(rows, scanPlanejamentoReserva)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if reservas == nil {
		return []PlanejamentoReserva{}, nil
	}

	return reservas, nil
}

func (s *cicloViagemStore) ListReservasElegiveisParaVolta(ctx context.Context, filtro PlanejamentoReservasFiltro) ([]PlanejamentoReserva, error) {
	const op = "db/cicloViagemStore.ListReservasElegiveisParaVolta"

	const q = `
		SELECT r.id, r.destino_id
		FROM reservas r
		JOIN destinos d ON d.id = r.destino_id
		WHERE r.data_viagem = @data_viagem
			AND r.turno = @turno
			AND d.municipio_id = @municipio_destino_id
			AND r.rota_interna_id = @rota_interna_id
			AND r.sentido = 'volta'
			AND r.status = 'confirmada'
			AND EXISTS (
				SELECT 1
				FROM reservas reserva_ida
				JOIN viagem_reservas vr ON vr.reserva_id = reserva_ida.id
				JOIN viagens viagem_ida ON viagem_ida.id = vr.viagem_id
				JOIN ciclos_viagem ciclo_ida ON ciclo_ida.id = viagem_ida.ciclo_viagem_id
				WHERE reserva_ida.cliente_id = r.cliente_id
					AND reserva_ida.sentido = 'ida'
					AND viagem_ida.sentido = 'ida'
					AND vr.status_presenca = 'embarcou'
					AND ciclo_ida.data_viagem = @data_viagem
					AND ciclo_ida.turno = @turno
					AND ciclo_ida.municipio_destino_id = @municipio_destino_id
					AND ciclo_ida.rota_interna_id = @rota_interna_id
			)
		ORDER BY r.id
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":          filtro.DataViagem,
		"turno":                filtro.Turno,
		"municipio_destino_id": filtro.MunicipioDestinoID,
		"rota_interna_id":      filtro.RotaInternaID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reservas, err := pgx.CollectRows(rows, scanPlanejamentoReserva)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if reservas == nil {
		return []PlanejamentoReserva{}, nil
	}

	return reservas, nil
}

func (s *cicloViagemStore) ListCiclosParaPlanejamentoVolta(ctx context.Context, filtro PlanejamentoReservasFiltro) ([]CicloPlanejamentoVolta, error) {
	const op = "db/cicloViagemStore.ListCiclosParaPlanejamentoVolta"

	const q = `
		SELECT
			c.id, c.data_viagem, c.turno, c.municipio_destino_id, c.rota_interna_id,
			c.veiculo_id, c.motorista_id, c.status, c.expires_at, c.created_at, c.updated_at,
			v.capacidade
		FROM ciclos_viagem c
		JOIN veiculos v ON v.id = c.veiculo_id
		JOIN viagens ida ON ida.ciclo_viagem_id = c.id AND ida.sentido = 'ida'
		WHERE c.data_viagem = @data_viagem
			AND c.turno = @turno
			AND c.municipio_destino_id = @municipio_destino_id
			AND c.rota_interna_id = @rota_interna_id
			AND c.status <> 'cancelado'
			AND ida.status <> 'cancelada'
		ORDER BY c.id
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":          filtro.DataViagem,
		"turno":                filtro.Turno,
		"municipio_destino_id": filtro.MunicipioDestinoID,
		"rota_interna_id":      filtro.RotaInternaID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	ciclos, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (CicloPlanejamentoVolta, error) {
		var data CicloPlanejamentoVolta
		err := row.Scan(
			&data.Ciclo.ID,
			&data.Ciclo.DataViagem,
			&data.Ciclo.Turno,
			&data.Ciclo.MunicipioDestinoID,
			&data.Ciclo.RotaInternaID,
			&data.Ciclo.VeiculoID,
			&data.Ciclo.MotoristaID,
			&data.Ciclo.Status,
			&data.Ciclo.ExpiresAt,
			&data.Ciclo.CreatedAt,
			&data.Ciclo.UpdatedAt,
			&data.Capacidade,
		)
		return data, err
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if ciclos == nil {
		return []CicloPlanejamentoVolta{}, nil
	}

	return ciclos, nil
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
			id, data_viagem, turno, municipio_destino_id, rota_interna_id, veiculo_id, motorista_id,
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
				rota_interna_id = @rota_interna_id,
				veiculo_id = @veiculo_id,
				motorista_id = @motorista_id,
				status = @status,
				expires_at = @expires_at
			WHERE id = @id
			RETURNING
				id, data_viagem, turno, municipio_destino_id, rota_interna_id, veiculo_id, motorista_id,
				status, expires_at, created_at, updated_at
		`

		rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
			"id":              ciclo.ID,
			"data_viagem":     ciclo.DataViagem,
			"turno":           ciclo.Turno,
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
			id, data_viagem, turno, municipio_destino_id, rota_interna_id, veiculo_id, motorista_id,
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
			data_viagem, turno, municipio_destino_id, rota_interna_id, veiculo_id, motorista_id, expires_at
		)
		VALUES (
			@data_viagem, @turno, @municipio_destino_id, @rota_interna_id, @veiculo_id, @motorista_id, @expires_at
		)
		RETURNING
			id, data_viagem, turno, municipio_destino_id, rota_interna_id, veiculo_id, motorista_id,
			status, expires_at, created_at, updated_at
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":          input.DataViagem,
		"turno":                input.Turno,
		"municipio_destino_id": input.MunicipioDestinoID,
		"rota_interna_id":      input.RotaInternaID,
		"veiculo_id":           input.VeiculoID,
		"motorista_id":         input.MotoristaID,
		"expires_at":           input.ExpiresAt,
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
		&ciclo.MunicipioDestinoID,
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

func scanPlanejamentoReserva(row pgx.CollectableRow) (PlanejamentoReserva, error) {
	var reserva PlanejamentoReserva
	err := row.Scan(&reserva.ID, &reserva.DestinoID)
	return reserva, err
}

func isCicloAlreadyAllocated(err error) bool {
	return db.IsUniqueViolation(err, "uq_ciclos_viagem_ativos_veiculo_data_turno") ||
		db.IsUniqueViolation(err, "uq_ciclos_viagem_ativos_motorista_data_turno")
}
