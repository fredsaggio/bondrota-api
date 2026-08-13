package viagens

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/publicid"
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

	viagem, err := publicid.Insert(publicid.Viagem, func(publicID string) (*Viagem, error) {
		var viagem Viagem
		err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
			const q = `
			INSERT INTO viagens (public_id, ciclo_viagem_id, sentido)
			VALUES (@public_id, @ciclo_viagem_id, @sentido)
			RETURNING id, public_id, ciclo_viagem_id, sentido, status, created_at, updated_at
		`

			rows, err := tx.Query(ctx, q, pgx.StrictNamedArgs{
				"public_id":       publicID,
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
		return &viagem, err
	}, func(err error) bool {
		return db.IsUniqueViolation(err, "viagens_public_id_key")
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return viagem, nil
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

const (
	defaultViagemListLimit = 50
	maxViagemListLimit     = 200
)

// listViagensColumns e a projecao compartilhada entre a listagem paginada e as
// "proximas viagens" do resumo, para as duas devolverem exatamente a mesma forma.
const listViagensColumns = `
	v.id, v.public_id, v.ciclo_viagem_id, v.sentido, v.status, v.created_at, v.updated_at,
	c.id, c.data_viagem, c.turno, c.municipio_destino_id, c.rota_interna_id,
	c.veiculo_id, c.motorista_id, mot.public_id, c.status, c.expires_at, c.created_at, c.updated_at,
	m.nome, ve.placa
`

const listViagensFrom = `
	FROM viagens v
	JOIN ciclos_viagem c ON c.id = v.ciclo_viagem_id
	JOIN municipios m ON m.codigo_ibge = c.municipio_destino_id
	JOIN veiculos ve ON ve.id = c.veiculo_id
	JOIN motoristas mot ON mot.id = c.motorista_id
`

func (s *viagemStore) ListViagens(ctx context.Context, params ViagemListParams) (ViagemListResult, error) {
	const op = "db/viagemStore.ListViagens"

	limit := params.Limit
	if limit <= 0 {
		limit = defaultViagemListLimit
	}
	if limit > maxViagemListLimit {
		limit = maxViagemListLimit
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

	// A direcao vem de um booleano, nunca de texto do usuario: interpolar aqui
	// mantem a comparacao do cursor alinhada ao ORDER BY e deixa o indice ser
	// usado nos dois sentidos.
	comparador, direcao := "<", "DESC"
	if params.Ascendente {
		comparador, direcao = ">", "ASC"
	}

	// A ordem virou (data_viagem, viagem.id): o cursor precisa de uma chave total
	// e de direcao unica para a comparacao de tupla funcionar. Antes o desempate
	// dentro da data era por turno e sentido.
	q := fmt.Sprintf(`
		SELECT `+listViagensColumns+listViagensFrom+`
		WHERE (@data_inicio::DATE IS NULL OR c.data_viagem >= @data_inicio)
		  AND (@data_fim::DATE IS NULL OR c.data_viagem <= @data_fim)
		  AND (@motorista_id = 0 OR c.motorista_id = @motorista_id)
		  AND (cardinality(@status::TEXT[]) = 0 OR v.status::TEXT = ANY(@status))
		  AND (@busca = '' OR
		       m.nome ILIKE '%%' || @busca || '%%' OR
		       ve.placa ILIKE '%%' || @busca || '%%' OR
		       v.status::TEXT ILIKE '%%' || @busca || '%%' OR
		       c.turno::TEXT ILIKE '%%' || @busca || '%%' OR
		       v.sentido::TEXT ILIKE '%%' || @busca || '%%' OR
		       v.id::TEXT = @busca)
		  AND (@has_cursor = FALSE OR (c.data_viagem, v.id) %s (@cursor_data, @cursor_id))
		ORDER BY c.data_viagem %s, v.id %s
		LIMIT @limit
	`, comparador, direcao, direcao)

	status := make([]string, 0, len(params.Status))
	for _, item := range params.Status {
		status = append(status, string(item))
	}

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_inicio":  params.DataInicio,
		"data_fim":     params.DataFim,
		"motorista_id": params.MotoristaID,
		"status":       status,
		"busca":        strings.TrimSpace(params.Busca),
		"has_cursor":   hasCursor,
		"cursor_data":  cursorData,
		"cursor_id":    cursorID,
		"limit":        limit + 1,
	})
	if err != nil {
		return ViagemListResult{}, fmt.Errorf("%s: %w", op, err)
	}

	items, err := pgx.CollectRows(rows, scanViagemComCicloENomes)
	if err != nil {
		return ViagemListResult{}, fmt.Errorf("%s: %w", op, err)
	}

	result := ViagemListResult{Items: items}
	if len(items) > limit {
		result.Items = items[:limit]
		last := result.Items[len(result.Items)-1]
		result.NextCursor = &ViagemCursor{DataViagem: last.Ciclo.DataViagem, ID: last.Viagem.ID}
		result.HasMore = true
	}
	return result, nil
}

func (s *viagemStore) ResumoViagens(ctx context.Context, hoje time.Time) (ViagemResumo, error) {
	const op = "db/viagemStore.ResumoViagens"

	resumo := ViagemResumo{
		PorStatus: map[StatusViagem]int64{},
		PorTurno:  map[TurnoViagem]int64{},
	}

	statusRows, err := s.db.Query(ctx, `SELECT status, COUNT(*) FROM viagens GROUP BY status`)
	if err != nil {
		return ViagemResumo{}, fmt.Errorf("%s: %w", op, err)
	}
	for statusRows.Next() {
		var (
			status StatusViagem
			total  int64
		)
		if err := statusRows.Scan(&status, &total); err != nil {
			statusRows.Close()
			return ViagemResumo{}, fmt.Errorf("%s: %w", op, err)
		}
		resumo.PorStatus[status] = total
	}
	statusRows.Close()
	if err := statusRows.Err(); err != nil {
		return ViagemResumo{}, fmt.Errorf("%s: %w", op, err)
	}

	turnoRows, err := s.db.Query(ctx, `
		SELECT c.turno, COUNT(*)
		FROM viagens v
		JOIN ciclos_viagem c ON c.id = v.ciclo_viagem_id
		GROUP BY c.turno
	`)
	if err != nil {
		return ViagemResumo{}, fmt.Errorf("%s: %w", op, err)
	}
	for turnoRows.Next() {
		var (
			turno TurnoViagem
			total int64
		)
		if err := turnoRows.Scan(&turno, &total); err != nil {
			turnoRows.Close()
			return ViagemResumo{}, fmt.Errorf("%s: %w", op, err)
		}
		resumo.PorTurno[turno] = total
	}
	turnoRows.Close()
	if err := turnoRows.Err(); err != nil {
		return ViagemResumo{}, fmt.Errorf("%s: %w", op, err)
	}

	err = s.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE v.status = 'em_andamento')
		FROM viagens v
		JOIN ciclos_viagem c ON c.id = v.ciclo_viagem_id
		WHERE c.data_viagem = @hoje
	`, pgx.StrictNamedArgs{"hoje": hoje}).Scan(&resumo.HojeTotal, &resumo.HojeEmAndamento)
	if err != nil {
		return ViagemResumo{}, fmt.Errorf("%s: %w", op, err)
	}

	// Proximas: as mais perto de acontecer primeiro, entao ASC — o oposto da
	// listagem, que mostra o mais recente no topo.
	proximasRows, err := s.db.Query(ctx, `
		SELECT `+listViagensColumns+listViagensFrom+`
		WHERE v.status IN ('programada', 'em_andamento')
		ORDER BY c.data_viagem ASC, v.id ASC
		LIMIT 6
	`)
	if err != nil {
		return ViagemResumo{}, fmt.Errorf("%s: %w", op, err)
	}
	proximas, err := pgx.CollectRows(proximasRows, scanViagemComCicloENomes)
	if err != nil {
		return ViagemResumo{}, fmt.Errorf("%s: %w", op, err)
	}
	resumo.Proximas = proximas

	return resumo, nil
}

func (s *viagemStore) ListViagensByCiclo(ctx context.Context, cicloID int64) ([]Viagem, error) {
	const op = "db/viagemStore.ListViagensByCiclo"

	const q = `
		SELECT
			id, public_id, ciclo_viagem_id, sentido, status, created_at, updated_at
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
		SELECT h.id, h.viagem_id, v.public_id, h.tipo, h.horario, h.created_at, h.updated_at
		FROM viagem_horarios h
		JOIN viagens v ON v.id = h.viagem_id
		WHERE h.viagem_id = @viagem_id
		ORDER BY h.created_at ASC, h.id ASC
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

	viagemHorario, err := pgx.CollectExactlyOneRow(rows, scanViagemHorarioInternal)
	if err != nil {
		if isHorarioViagemAlreadyRegistered(err) {
			return nil, fmt.Errorf("%s: %w", op, brerror.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err := s.db.QueryRow(ctx, `SELECT public_id FROM viagens WHERE id = $1`, viagemID).Scan(&viagemHorario.ViagemPublicID); err != nil {
		return nil, fmt.Errorf("%s: resolve viagem public id: %w", op, err)
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
			return fmt.Errorf("%w: A situação atual da viagem não permite esta mudança.", brerror.ErrAlreadyExists)
		}

		const updateQ = `
			UPDATE viagens
			SET status = @status
			WHERE id = @id
			RETURNING id, public_id, ciclo_viagem_id, sentido, status, created_at, updated_at
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
			RETURNING id, public_id, ciclo_viagem_id, sentido, status, created_at, updated_at
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
			id, public_id, ciclo_viagem_id, sentido, status, created_at, updated_at
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
			v.id, v.public_id, v.ciclo_viagem_id, v.sentido, v.status, v.created_at, v.updated_at,
			c.id, c.data_viagem, c.turno, c.municipio_destino_id, c.rota_interna_id,
			c.veiculo_id, c.motorista_id, m.public_id, c.status, c.expires_at, c.created_at, c.updated_at
		FROM viagens v
		JOIN ciclos_viagem c ON c.id = v.ciclo_viagem_id
		JOIN motoristas m ON m.id = c.motorista_id
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
		&viagem.PublicID,
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
		&horario.ViagemPublicID,
		&horario.Tipo,
		&horario.Horario,
		&horario.CreatedAt,
		&horario.UpdatedAt,
	)
	return horario, err
}

func scanViagemHorarioInternal(row pgx.CollectableRow) (ViagemHorario, error) {
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
		&data.Viagem.PublicID,
		&data.Viagem.CicloViagemID,
		&data.Viagem.Sentido,
		&data.Viagem.Status,
		&data.Viagem.CreatedAt,
		&data.Viagem.UpdatedAt,
		&data.Ciclo.ID,
		&data.Ciclo.DataViagem,
		&data.Ciclo.Turno,
		&data.Ciclo.MunicipioDestinoID,
		&data.Ciclo.RotaInternaID,
		&data.Ciclo.VeiculoID,
		&data.Ciclo.MotoristaID,
		&data.Ciclo.MotoristaPublicID,
		&data.Ciclo.Status,
		&data.Ciclo.ExpiresAt,
		&data.Ciclo.CreatedAt,
		&data.Ciclo.UpdatedAt,
	)
	return data, err
}

func scanViagemComCicloENomes(row pgx.CollectableRow) (ViagemComCicloENomes, error) {
	var item ViagemComCicloENomes
	err := row.Scan(
		&item.Viagem.ID,
		&item.Viagem.PublicID,
		&item.Viagem.CicloViagemID,
		&item.Viagem.Sentido,
		&item.Viagem.Status,
		&item.Viagem.CreatedAt,
		&item.Viagem.UpdatedAt,
		&item.Ciclo.ID,
		&item.Ciclo.DataViagem,
		&item.Ciclo.Turno,
		&item.Ciclo.MunicipioDestinoID,
		&item.Ciclo.RotaInternaID,
		&item.Ciclo.VeiculoID,
		&item.Ciclo.MotoristaID,
		&item.Ciclo.MotoristaPublicID,
		&item.Ciclo.Status,
		&item.Ciclo.ExpiresAt,
		&item.Ciclo.CreatedAt,
		&item.Ciclo.UpdatedAt,
		&item.MunicipioNome,
		&item.VeiculoPlaca,
	)
	return item, err
}

func isViagemAlreadyCreated(err error) bool {
	return db.IsUniqueViolation(err, "uq_viagens_ciclo_sentido")
}

func isHorarioViagemAlreadyRegistered(err error) bool {
	return db.IsUniqueViolation(err, "uq_viagem_horarios_viagem_tipo")
}
