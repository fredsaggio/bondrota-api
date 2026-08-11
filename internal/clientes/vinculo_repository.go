package clientes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type vinculoStore struct {
	db db.DB
}

func NewVinculoStore(db db.DB) VinculoStore {
	return &vinculoStore{db: db}
}

func (s *vinculoStore) Create(ctx context.Context, input VinculoInput) (*Vinculo, error) {
	const op = "db/vinculoStore.Create"

	var vinculo Vinculo

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const q = `
			INSERT INTO cliente_vinculos (
				cliente_id, tipo, turno, destino_id, rota_interna_id, curso, comprovante, validade
			)
			VALUES (
				@cliente_id, @tipo, @turno, @destino_id, @rota_interna_id, @curso, @comprovante, @validade
			)
			RETURNING id, cliente_id, tipo, turno, destino_id, rota_interna_id, curso, comprovante, validade
		`
		rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
			"cliente_id":      input.ClienteID,
			"tipo":            input.Tipo,
			"turno":           input.Turno,
			"destino_id":      input.DestinoID,
			"rota_interna_id": input.RotaInternaID,
			"curso":           input.Curso,
			"comprovante":     input.Comprovante,
			"validade":        input.Validade,
		})
		if err != nil {
			return fmt.Errorf("insert vinculo: %w", err)
		}

		v, err := pgx.CollectExactlyOneRow(rows, scanVinculo)
		if err != nil {
			return fmt.Errorf("insert vinculo: %w", err)
		}

		horarios, err := replaceHorariosFixos(ctx, tx, v.ID, input.HorariosFixos)
		if err != nil {
			return err
		}
		v.HorariosFixos = horarios
		vinculo = v

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &vinculo, nil
}

func (s *vinculoStore) GetByID(ctx context.Context, vinculoID int64) (*Vinculo, error) {
	const op = "db/vinculoStore.GetByID"

	const q = `
        SELECT
            v.id, v.cliente_id, v.tipo, v.turno, v.destino_id, v.rota_interna_id,
            v.curso, v.comprovante, v.validade,
            h.id, h.vinculo_id, h.dia_semana
        FROM cliente_vinculos v
        LEFT JOIN horarios_fixos h ON h.vinculo_id = v.id
        WHERE v.id = @id
        ORDER BY v.id ASC, h.dia_semana ASC
    `

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"id": vinculoID})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	vinculos, err := collectVinculos(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(vinculos) == 0 {
		return nil, ErrVinculoNotFound
	}

	return &vinculos[0], nil
}

func (s *vinculoStore) List(ctx context.Context) ([]VinculoComCliente, error) {
	const op = "db/vinculoStore.List"

	const q = `
        SELECT
            v.id, v.cliente_id, v.tipo, v.turno, v.destino_id, v.rota_interna_id,
            v.curso, v.comprovante, v.validade,
            h.id, h.vinculo_id, h.dia_semana,
            c.nome
        FROM cliente_vinculos v
        JOIN clientes c ON c.id = v.cliente_id
        LEFT JOIN horarios_fixos h ON h.vinculo_id = v.id
        ORDER BY c.nome ASC, v.id ASC, h.dia_semana ASC
    `

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	vinculos, err := collectVinculosComCliente(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return vinculos, nil
}

func (s *vinculoStore) ListByCliente(ctx context.Context, clienteID int64) ([]Vinculo, error) {
	const op = "db/vinculoStore.ListByCliente"

	const q = `
        SELECT
            v.id, v.cliente_id, v.tipo, v.turno, v.destino_id, v.rota_interna_id,
            v.curso, v.comprovante, v.validade,
            h.id, h.vinculo_id, h.dia_semana
        FROM cliente_vinculos v
        LEFT JOIN horarios_fixos h ON h.vinculo_id = v.id
        WHERE v.cliente_id = @cliente_id
        ORDER BY v.id ASC, h.dia_semana ASC
    `

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"cliente_id": clienteID})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	vinculos, err := collectVinculos(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if vinculos == nil {
		return []Vinculo{}, nil
	}

	return vinculos, nil
}

func (s *vinculoStore) Update(ctx context.Context, vinculoID int64, input VinculoUpdateInput) (*Vinculo, error) {
	const op = "db/vinculoStore.Update"

	var vinculo Vinculo

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const selectQ = `
			SELECT id, cliente_id, tipo, turno, destino_id, rota_interna_id, curso, comprovante, validade
			FROM cliente_vinculos
			WHERE id = @id
			FOR UPDATE
		`
		rows, err := tx.Query(ctx, selectQ, pgx.StrictNamedArgs{"id": vinculoID})
		if err != nil {
			return fmt.Errorf("select vinculo: %w", err)
		}

		v, err := pgx.CollectExactlyOneRow(rows, scanVinculo)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrVinculoNotFound
			}
			return fmt.Errorf("select vinculo: %w", err)
		}

		const updateQ = `
			UPDATE cliente_vinculos
			SET tipo = @tipo,
				turno = @turno,
				destino_id = @destino_id,
				rota_interna_id = @rota_interna_id,
				curso = @curso,
				comprovante = @comprovante,
				validade = @validade
			WHERE id = @id
			RETURNING id, cliente_id, tipo, turno, destino_id, rota_interna_id, curso, comprovante, validade
		`
		rows, err = tx.Query(ctx, updateQ, pgx.StrictNamedArgs{
			"id":              v.ID,
			"tipo":            input.Tipo,
			"turno":           input.Turno,
			"destino_id":      input.DestinoID,
			"rota_interna_id": input.RotaInternaID,
			"curso":           input.Curso,
			"comprovante":     input.Comprovante,
			"validade":        input.Validade,
		})
		if err != nil {
			return fmt.Errorf("update vinculo: %w", err)
		}

		v, err = pgx.CollectExactlyOneRow(rows, scanVinculo)
		if err != nil {
			return fmt.Errorf("update vinculo: %w", err)
		}

		horarios, err := replaceHorariosFixos(ctx, tx, v.ID, input.HorariosFixos)
		if err != nil {
			return err
		}
		v.HorariosFixos = horarios
		vinculo = v

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &vinculo, nil
}

func (s *vinculoStore) Delete(ctx context.Context, vinculoID int64) error {
	const op = "db/vinculoStore.Delete"

	const q = `DELETE FROM cliente_vinculos WHERE id = @id`

	cmdTag, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{"id": vinculoID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrVinculoNotFound
	}

	return nil
}

// vinculoRow representa uma linha do join entre cliente_vinculos e horarios_fixos.
// Como o LEFT JOIN repete o vinculo uma vez por horario fixo, os coletores abaixo
// agrupam as linhas pelo id do vinculo.
type vinculoRow struct {
	vID        int64
	vClienteID int64
	vTipo      TipoConta
	vTurno     TurnoCliente
	vDestinoID int64
	vRotaID    int64
	vCurso     string
	vComp      string
	vValidade  time.Time
	hID        *int64
	hVinculoID *int64
	hDia       *DiaSemana
}

func (r *vinculoRow) scanTargets() []any {
	return []any{
		&r.vID, &r.vClienteID, &r.vTipo, &r.vTurno, &r.vDestinoID, &r.vRotaID,
		&r.vCurso, &r.vComp, &r.vValidade,
		&r.hID, &r.hVinculoID, &r.hDia,
	}
}

func (r *vinculoRow) vinculo() Vinculo {
	return Vinculo{
		ID:            r.vID,
		ClienteID:     r.vClienteID,
		Tipo:          r.vTipo,
		Turno:         r.vTurno,
		DestinoID:     r.vDestinoID,
		RotaInternaID: r.vRotaID,
		Curso:         r.vCurso,
		Comprovante:   r.vComp,
		Validade:      r.vValidade,
		HorariosFixos: []HorarioFixo{},
	}
}

func (r *vinculoRow) horarioFixo() (HorarioFixo, bool) {
	if r.hID == nil {
		return HorarioFixo{}, false
	}
	return HorarioFixo{ID: *r.hID, VinculoID: *r.hVinculoID, DiaSemana: *r.hDia}, true
}

func collectVinculos(rows pgx.Rows) ([]Vinculo, error) {
	defer rows.Close()

	var vinculos []Vinculo
	index := map[int64]int{}

	for rows.Next() {
		var row vinculoRow
		if err := rows.Scan(row.scanTargets()...); err != nil {
			return nil, err
		}

		i, ok := index[row.vID]
		if !ok {
			vinculos = append(vinculos, row.vinculo())
			i = len(vinculos) - 1
			index[row.vID] = i
		}

		if horario, ok := row.horarioFixo(); ok {
			vinculos[i].HorariosFixos = append(vinculos[i].HorariosFixos, horario)
		}
	}

	return vinculos, rows.Err()
}

func collectVinculosComCliente(rows pgx.Rows) ([]VinculoComCliente, error) {
	defer rows.Close()

	vinculos := []VinculoComCliente{}
	index := map[int64]int{}

	for rows.Next() {
		var (
			row         vinculoRow
			clienteNome string
		)
		if err := rows.Scan(append(row.scanTargets(), &clienteNome)...); err != nil {
			return nil, err
		}

		i, ok := index[row.vID]
		if !ok {
			vinculos = append(vinculos, VinculoComCliente{Vinculo: row.vinculo(), ClienteNome: clienteNome})
			i = len(vinculos) - 1
			index[row.vID] = i
		}

		if horario, ok := row.horarioFixo(); ok {
			vinculos[i].HorariosFixos = append(vinculos[i].HorariosFixos, horario)
		}
	}

	return vinculos, rows.Err()
}

func scanVinculo(row pgx.CollectableRow) (Vinculo, error) {
	var v Vinculo
	err := row.Scan(
		&v.ID,
		&v.ClienteID,
		&v.Tipo,
		&v.Turno,
		&v.DestinoID,
		&v.RotaInternaID,
		&v.Curso,
		&v.Comprovante,
		&v.Validade,
	)
	v.HorariosFixos = []HorarioFixo{}
	return v, err
}

func replaceHorariosFixos(ctx context.Context, tx pgx.Tx, vinculoID int64, dias []DiaSemana) ([]HorarioFixo, error) {
	const deleteQ = `DELETE FROM horarios_fixos WHERE vinculo_id = @vinculo_id`
	if _, err := tx.Exec(ctx, deleteQ, pgx.StrictNamedArgs{"vinculo_id": vinculoID}); err != nil {
		return nil, fmt.Errorf("delete horarios fixos: %w", err)
	}

	const insertQ = `
		INSERT INTO horarios_fixos (vinculo_id, dia_semana)
		VALUES (@vinculo_id, @dia_semana)
		RETURNING id, vinculo_id, dia_semana
	`

	batch := &pgx.Batch{}
	for _, dia := range dias {
		batch.Queue(insertQ, pgx.StrictNamedArgs{
			"vinculo_id": vinculoID,
			"dia_semana": dia,
		})
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	horarios := make([]HorarioFixo, 0, len(dias))
	for range dias {
		rows, err := results.Query()
		if err != nil {
			return nil, fmt.Errorf("insert horario fixo: %w", err)
		}

		h, err := pgx.CollectExactlyOneRow(rows, scanHorarioFixo)
		if err != nil {
			return nil, fmt.Errorf("insert horario fixo: %w", err)
		}
		horarios = append(horarios, h)
	}

	return horarios, nil
}

func scanHorarioFixo(row pgx.CollectableRow) (HorarioFixo, error) {
	var h HorarioFixo
	err := row.Scan(&h.ID, &h.VinculoID, &h.DiaSemana)
	return h, err
}
