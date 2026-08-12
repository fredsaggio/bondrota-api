package reservas

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

	var reserva Reserva
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := bloquearSePlanejamentoIniciado(ctx, tx, input.DataViagem, input.Turno, input.DestinoID, input.RotaInternaID, input.Sentido); err != nil {
			return err
		}

		rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
			"cliente_id":      input.ClienteID,
			"vinculo_id":      input.VinculoID,
			"data_viagem":     input.DataViagem,
			"turno":           input.Turno,
			"destino_id":      input.DestinoID,
			"rota_interna_id": input.RotaInternaID,
			"sentido":         input.Sentido,
		})
		if err != nil {
			return err
		}

		reserva, err = pgx.CollectExactlyOneRow(rows, scanReserva)
		return err
	})
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

func (s *reservaStore) GetHorarioPartida(ctx context.Context, destinoID int64, turno TurnoReserva, sentido SentidoReserva) (time.Duration, error) {
	const op = "db/reservaStore.GetHorarioPartida"

	const q = `
		SELECT EXTRACT(EPOCH FROM CASE @sentido::reserva_sentido
			WHEN 'ida' THEN h.horario_ida
			WHEN 'volta' THEN h.horario_volta
		END)::BIGINT
		FROM destinos d
		JOIN horarios_turno_viagem h
			ON h.municipio_destino_id = d.municipio_id
			AND h.turno = @turno::turno_cliente
		WHERE d.id = @destino_id
	`

	var segundos int64
	err := s.db.QueryRow(ctx, q, pgx.StrictNamedArgs{
		"destino_id": destinoID,
		"turno":      turno,
		"sentido":    sentido,
	}).Scan(&segundos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrHorarioNaoConfigurado
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return time.Duration(segundos) * time.Second, nil
}

// defaultReservaListLimit e o tamanho de pagina quando o consumidor nao pede um
// explicitamente. maxReservaListLimit evita que um limit absurdo (ou um bug no
// consumidor) force a query a devolver a tabela inteira de uma vez.
const (
	defaultReservaListLimit = 50
	maxReservaListLimit     = 200
)

func (s *reservaStore) List(ctx context.Context, params ReservaListParams) (ReservaListResult, error) {
	const op = "db/reservaStore.List"

	limit := params.Limit
	if limit <= 0 {
		limit = defaultReservaListLimit
	}
	if limit > maxReservaListLimit {
		limit = maxReservaListLimit
	}

	var (
		hasCursor  bool
		cursorData time.Time
		cursorID   int64
	)
	if params.Cursor != nil {
		hasCursor = true
		cursorData = params.Cursor.DataViagem
		cursorID = params.Cursor.ID
	}

	// Busca um a mais que o limite para saber se ha proxima pagina sem uma
	// segunda query. A data fica fora do "busca": o filtro de intervalo abaixo
	// (data_inicio/data_fim) e o unico jeito de restringir por data.
	const q = `
		SELECT
			r.id, r.cliente_id, r.vinculo_id, r.data_viagem, r.turno, r.destino_id,
			r.rota_interna_id, r.sentido, r.status, r.created_at, r.updated_at,
			c.nome, d.nome
		FROM reservas r
		JOIN clientes c ON c.id = r.cliente_id
		JOIN destinos d ON d.id = r.destino_id
		WHERE (@data_inicio::DATE IS NULL OR r.data_viagem >= @data_inicio)
		  AND (@data_fim::DATE IS NULL OR r.data_viagem <= @data_fim)
		  AND (@busca = '' OR
		       c.nome ILIKE '%' || @busca || '%' OR
		       d.nome ILIKE '%' || @busca || '%' OR
		       r.status::TEXT ILIKE '%' || @busca || '%' OR
		       r.turno::TEXT ILIKE '%' || @busca || '%' OR
		       r.sentido::TEXT ILIKE '%' || @busca || '%' OR
		       r.id::TEXT = @busca)
		  AND (@has_cursor = FALSE OR (r.data_viagem, r.id) < (@cursor_data, @cursor_id))
		ORDER BY r.data_viagem DESC, r.id DESC
		LIMIT @limit
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_inicio": params.DataInicio,
		"data_fim":    params.DataFim,
		"busca":       strings.TrimSpace(params.Busca),
		"has_cursor":  hasCursor,
		"cursor_data": cursorData,
		"cursor_id":   cursorID,
		"limit":       limit + 1,
	})
	if err != nil {
		return ReservaListResult{}, fmt.Errorf("%s: %w", op, err)
	}

	items, err := pgx.CollectRows(rows, scanReservaComNomes)
	if err != nil {
		return ReservaListResult{}, fmt.Errorf("%s: %w", op, err)
	}

	result := ReservaListResult{Items: items}
	if len(items) > limit {
		result.Items = items[:limit]
		last := result.Items[len(result.Items)-1]
		result.NextCursor = &ReservaCursor{DataViagem: last.DataViagem, ID: last.ID}
		result.HasMore = true
	}
	return result, nil
}

// Resumo agrega no banco em vez de contar linhas na aplicacao: o custo nao cresce
// com o tamanho da tabela e a resposta tem sempre no maximo um punhado de linhas.
func (s *reservaStore) Resumo(ctx context.Context) (ReservaResumo, error) {
	const op = "db/reservaStore.Resumo"

	const q = `
		SELECT turno, COUNT(*)
		FROM reservas
		WHERE status = 'confirmada'
		GROUP BY turno
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return ReservaResumo{}, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	resumo := ReservaResumo{ConfirmadasPorTurno: map[TurnoReserva]int64{}}
	for rows.Next() {
		var (
			turno TurnoReserva
			total int64
		)
		if err := rows.Scan(&turno, &total); err != nil {
			return ReservaResumo{}, fmt.Errorf("%s: %w", op, err)
		}
		resumo.ConfirmadasPorTurno[turno] = total
		resumo.ConfirmadasTotal += total
	}
	if err := rows.Err(); err != nil {
		return ReservaResumo{}, fmt.Errorf("%s: %w", op, err)
	}

	return resumo, nil
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
		if reserva.Status == StatusConfirmada {
			if err := bloquearSePlanejamentoIniciado(ctx, tx, reserva.DataViagem, reserva.Turno, reserva.DestinoID, reserva.RotaInternaID, reserva.Sentido); err != nil {
				return err
			}
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

func bloquearSePlanejamentoIniciado(
	ctx context.Context,
	tx pgx.Tx,
	dataViagem time.Time,
	turno TurnoReserva,
	destinoID int64,
	rotaInternaID int64,
	sentido SentidoReserva,
) error {
	var municipioDestinoID int64
	err := tx.QueryRow(ctx, `
		SELECT municipio_id
		FROM destinos
		WHERE id = @destino_id
		FOR SHARE
	`, pgx.StrictNamedArgs{"destino_id": destinoID}).Scan(&municipioDestinoID)
	if err != nil {
		return fmt.Errorf("select municipio destino: %w", err)
	}

	_, err = tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(
			planejamento_advisory_lock_key(
				@data_viagem::DATE,
				@turno::TEXT,
				@municipio_destino_id,
				@rota_interna_id,
				@sentido::TEXT
			)
		)
	`, pgx.StrictNamedArgs{
		"data_viagem":          dataViagem,
		"turno":                turno,
		"municipio_destino_id": municipioDestinoID,
		"rota_interna_id":      rotaInternaID,
		"sentido":              sentido,
	})
	if err != nil {
		return fmt.Errorf("lock planejamento: %w", err)
	}

	var planejamentoIniciado bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM execucoes_planejamento
			WHERE data_viagem = @data_viagem
				AND turno = @turno
				AND municipio_destino_id = @municipio_destino_id
				AND rota_interna_id = @rota_interna_id
				AND sentido::TEXT = @sentido::TEXT
		)
	`, pgx.StrictNamedArgs{
		"data_viagem":          dataViagem,
		"turno":                turno,
		"municipio_destino_id": municipioDestinoID,
		"rota_interna_id":      rotaInternaID,
		"sentido":              sentido,
	}).Scan(&planejamentoIniciado)
	if err != nil {
		return fmt.Errorf("check planejamento: %w", err)
	}
	if planejamentoIniciado {
		return ErrPrazoReservaEncerrado
	}

	return nil
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

func scanReservaComNomes(row pgx.CollectableRow) (ReservaComNomes, error) {
	var item ReservaComNomes
	err := row.Scan(
		&item.ID,
		&item.ClienteID,
		&item.VinculoID,
		&item.DataViagem,
		&item.Turno,
		&item.DestinoID,
		&item.RotaInternaID,
		&item.Sentido,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ClienteNome,
		&item.DestinoNome,
	)
	return item, err
}
