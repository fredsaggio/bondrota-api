package clientes

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

var nonDigits = regexp.MustCompile(`\D`)

func contemLetra(value string) bool {
	for _, char := range value {
		if unicode.IsLetter(char) {
			return true
		}
	}
	return false
}

type clienteStore struct {
	db db.DB
}

func NewClienteStore(db db.DB) ClienteStore {
	return &clienteStore{db: db}
}

func (s *clienteStore) Create(ctx context.Context, input ClienteInput) (*Cliente, error) {
	const op = "db/clienteStore.Create"

	const q = `
		INSERT INTO clientes (nome, cpf, senha, telefone, data_nasc, foto)
		VALUES (@nome, @cpf, @senha, @telefone, @data_nasc, @foto)
		RETURNING id, nome, cpf, telefone, data_nasc, foto
	`
	args := pgx.StrictNamedArgs{
		"nome":      input.Nome,
		"cpf":       input.CPF,
		"senha":     input.Senha,
		"telefone":  input.Telefone,
		"data_nasc": input.DataNasc,
		"foto":      input.Foto,
	}

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	c, err := pgx.CollectExactlyOneRow(rows, scanCliente)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &c, nil
}

func (s *clienteStore) GetByID(ctx context.Context, clienteID int64) (*ClienteComVinculos, error) {
	const op = "db/clienteStore.GetByID"

	const q = `
		SELECT
			c.id, c.nome, c.cpf, c.telefone, c.data_nasc, c.foto,
			v.id, v.cliente_id, v.tipo, v.turno, v.destino_id, v.rota_interna_id,
			v.curso, v.comprovante, v.validade,
			h.id, h.vinculo_id, h.dia_semana
		FROM clientes c
		LEFT JOIN cliente_vinculos v ON v.cliente_id = c.id
		LEFT JOIN horarios_fixos h ON h.vinculo_id = v.id
		WHERE c.id = @id
		ORDER BY v.id ASC, h.dia_semana ASC
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"id": clienteID})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result, err := collectClienteComVinculos(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if result == nil {
		return nil, ErrNotFound
	}

	return result, nil
}

func (s *clienteStore) GetByCPF(ctx context.Context, cpf string) (*Cliente, error) {
	const op = "db/clienteStore.GetByCPF"

	const q = `
		SELECT id, nome, cpf, senha, telefone, data_nasc, foto
		FROM clientes
		WHERE cpf = @cpf
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"cpf": cpf})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	c, err := pgx.CollectExactlyOneRow(rows, scanClienteComSenha)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &c, nil
}

const (
	defaultClienteListLimit = 50
	maxClienteListLimit     = 200
)

func (s *clienteStore) List(ctx context.Context, params ClienteListParams) (ClienteListResult, error) {
	const op = "db/clienteStore.List"

	limit := params.Limit
	if limit <= 0 {
		limit = defaultClienteListLimit
	}
	if limit > maxClienteListLimit {
		limit = maxClienteListLimit
	}

	// A busca por CPF ignora pontuacao: o cadastro guarda so digitos, mas quem
	// digita costuma colar o documento formatado. So vale para termos sem letra:
	// senao "Cliente 13" viraria uma busca por CPF contendo "13" e traria junto
	// todo mundo cujo documento tem esses digitos em qualquer posicao.
	busca := strings.TrimSpace(params.Busca)
	digitos := ""
	if !contemLetra(busca) {
		digitos = nonDigits.ReplaceAllString(busca, "")
	}

	const q = `
		SELECT id, nome, cpf, telefone, data_nasc, foto
		FROM clientes
		WHERE (@cursor_id = 0 OR id < @cursor_id)
		  AND (@busca = '' OR
		       nome ILIKE '%' || @busca || '%' OR
		       telefone ILIKE '%' || @busca || '%' OR
		       (@digitos <> '' AND cpf LIKE '%' || @digitos || '%'))
		ORDER BY id DESC
		LIMIT @limit
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"cursor_id": params.CursorID,
		"busca":     busca,
		"digitos":   digitos,
		"limit":     limit + 1,
	})
	if err != nil {
		return ClienteListResult{}, fmt.Errorf("%s: %w", op, err)
	}

	items, err := pgx.CollectRows(rows, scanCliente)
	if err != nil {
		return ClienteListResult{}, fmt.Errorf("%s: %w", op, err)
	}

	result := ClienteListResult{Items: items}
	if len(items) > limit {
		result.Items = items[:limit]
		result.NextCursorID = result.Items[len(result.Items)-1].ID
		result.HasMore = true
	}
	return result, nil
}

func (s *clienteStore) Resumo(ctx context.Context) (ClienteResumo, error) {
	const op = "db/clienteStore.Resumo"

	var resumo ClienteResumo
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM clientes`).Scan(&resumo.Total); err != nil {
		return ClienteResumo{}, fmt.Errorf("%s: %w", op, err)
	}
	return resumo, nil
}

func (s *clienteStore) Update(ctx context.Context, clienteID int64, updateFunc func(*Cliente) (bool, error)) (*Cliente, error) {
	const op = "db/clienteStore.Update"

	var cliente Cliente

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const selectQ = `
			SELECT id, nome, cpf, telefone, data_nasc, foto
			FROM clientes
			WHERE id = @id
			FOR UPDATE
		`
		rows, err := tx.Query(ctx, selectQ, pgx.StrictNamedArgs{"id": clienteID})
		if err != nil {
			return fmt.Errorf("select: %w", err)
		}

		c, err := pgx.CollectExactlyOneRow(rows, scanCliente)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("select: %w", err)
		}
		cliente = c

		changed, err := updateFunc(&cliente)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const updateQ = `
			UPDATE clientes
			SET nome = @nome, telefone = @telefone, data_nasc = @data_nasc, foto = @foto
			WHERE id = @id
		`
		_, err = tx.Exec(ctx, updateQ, pgx.StrictNamedArgs{
			"id":        cliente.ID,
			"nome":      cliente.Nome,
			"telefone":  cliente.Telefone,
			"data_nasc": cliente.DataNasc,
			"foto":      cliente.Foto,
		})
		if err != nil {
			return fmt.Errorf("update: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &cliente, nil
}

func (s *clienteStore) Delete(ctx context.Context, clienteID int64) error {
	const op = "db/clienteStore.Delete"

	const q = `DELETE FROM clientes WHERE id = @id`

	cmdTag, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{"id": clienteID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func collectClienteComVinculos(rows pgx.Rows) (*ClienteComVinculos, error) {
	defer rows.Close()

	var result *ClienteComVinculos
	vinculoIndex := map[int64]int{}

	for rows.Next() {
		var (
			cID        int64
			cNome      string
			cCPF       string
			cTelefone  string
			cDataNasc  time.Time
			cFoto      string
			vID        *int64
			vClienteID *int64
			vTipo      *TipoConta
			vTurno     *TurnoCliente
			vDestinoID *int64
			vRotaID    *int64
			vCurso     *string
			vComp      *string
			vValidade  *time.Time
			hID        *int64
			hVinculoID *int64
			hDia       *DiaSemana
		)

		if err := rows.Scan(
			&cID, &cNome, &cCPF, &cTelefone, &cDataNasc, &cFoto,
			&vID, &vClienteID, &vTipo, &vTurno, &vDestinoID, &vRotaID,
			&vCurso, &vComp, &vValidade,
			&hID, &hVinculoID, &hDia,
		); err != nil {
			return nil, err
		}

		if result == nil {
			result = &ClienteComVinculos{
				Cliente: Cliente{
					ID:       cID,
					Nome:     cNome,
					CPF:      cCPF,
					Telefone: cTelefone,
					DataNasc: cDataNasc,
					Foto:     cFoto,
				},
				Vinculos: []Vinculo{},
			}
		}

		if vID != nil {
			if _, ok := vinculoIndex[*vID]; !ok {
				result.Vinculos = append(result.Vinculos, Vinculo{
					ID:            *vID,
					ClienteID:     *vClienteID,
					Tipo:          *vTipo,
					Turno:         *vTurno,
					DestinoID:     *vDestinoID,
					RotaInternaID: *vRotaID,
					Curso:         *vCurso,
					Comprovante:   *vComp,
					Validade:      *vValidade,
					HorariosFixos: []HorarioFixo{},
				})
				vinculoIndex[*vID] = len(result.Vinculos) - 1
			}

			if hID != nil {
				i := vinculoIndex[*vID]
				result.Vinculos[i].HorariosFixos = append(result.Vinculos[i].HorariosFixos, HorarioFixo{
					ID:        *hID,
					VinculoID: *hVinculoID,
					DiaSemana: *hDia,
				})
			}
		}
	}

	return result, rows.Err()
}

func scanCliente(row pgx.CollectableRow) (Cliente, error) {
	var c Cliente
	err := row.Scan(&c.ID, &c.Nome, &c.CPF, &c.Telefone, &c.DataNasc, &c.Foto)
	return c, err
}

func scanClienteComSenha(row pgx.CollectableRow) (Cliente, error) {
	var c Cliente
	err := row.Scan(&c.ID, &c.Nome, &c.CPF, &c.Senha, &c.Telefone, &c.DataNasc, &c.Foto)
	return c, err
}
