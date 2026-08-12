package municipios

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type store struct {
	db db.DB
}

func NewStore(db db.DB) Store {
	return &store{db: db}
}

func (s *store) ListByUF(ctx context.Context, uf string) ([]Municipio, error) {
	const op = "db/municipioStore.ListByUF"
	const q = `
		SELECT codigo_ibge, nome, uf, ativo
		FROM municipios
		WHERE uf = @uf AND ativo = TRUE
		ORDER BY nome, codigo_ibge
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"uf": strings.ToUpper(strings.TrimSpace(uf))})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result, err := pgx.CollectRows(rows, scanMunicipio)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if result == nil {
		return []Municipio{}, nil
	}
	return result, nil
}

func (s *store) GetByID(ctx context.Context, codigoIBGE int64) (*Municipio, error) {
	const op = "db/municipioStore.GetByID"
	const q = `
		SELECT codigo_ibge, nome, uf, ativo
		FROM municipios
		WHERE codigo_ibge = @codigo_ibge
	`

	// Sem filtro de "ativo": um registro existente (motorista, destino, horário)
	// pode apontar para um município que a última reimportação do IBGE desativou.
	// A busca por id resolve o nome de qualquer jeito, em vez de tratar como
	// inexistente.
	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"codigo_ibge": codigoIBGE})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	municipio, err := pgx.CollectExactlyOneRow(rows, scanMunicipio)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &municipio, nil
}

func (s *store) Upsert(ctx context.Context, items []Municipio) error {
	const op = "db/municipioStore.Upsert"
	if len(items) == 0 {
		return nil
	}

	codigos := make([]int64, len(items))
	nomes := make([]string, len(items))
	ufs := make([]string, len(items))
	for i, municipio := range items {
		codigos[i] = municipio.CodigoIBGE
		nomes[i] = strings.TrimSpace(municipio.Nome)
		ufs[i] = strings.ToUpper(strings.TrimSpace(municipio.UF))
	}

	const q = `
		INSERT INTO municipios (codigo_ibge, nome, uf, ativo)
		SELECT codigo_ibge, nome, uf, TRUE
		FROM UNNEST(@codigos::BIGINT[], @nomes::TEXT[], @ufs::TEXT[])
			AS imported(codigo_ibge, nome, uf)
		ON CONFLICT (codigo_ibge) DO UPDATE
		SET nome = EXCLUDED.nome,
			uf = EXCLUDED.uf,
			ativo = TRUE
	`

	if _, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{
		"codigos": codigos,
		"nomes":   nomes,
		"ufs":     ufs,
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func scanMunicipio(row pgx.CollectableRow) (Municipio, error) {
	var municipio Municipio
	err := row.Scan(&municipio.CodigoIBGE, &municipio.Nome, &municipio.UF, &municipio.Ativo)
	return municipio, err
}
