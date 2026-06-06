package veiculos

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type veiculoStore struct {
	db db.DB
}

func NewVeiculoStore(db db.DB) VeiculoStore {
	return &veiculoStore{db: db}
}

func (s *veiculoStore) Create(ctx context.Context, input VeiculoInput) (*Veiculo, error) {
	const op = "db/veiculoStore.Create"

	const q = `
		INSERT INTO veiculos (placa, modelo, categoria, capacidade, cidade_base, status, ar_condicionado, banheiro, persiana, luz_leitura, tomada)
		VALUES (@placa, @modelo, @categoria, @capacidade, @cidade_base, @status, @ar_condicionado, @banheiro, @persiana, @luz_leitura, @tomada)
		RETURNING id
	`
	args := pgx.StrictNamedArgs{
		"placa":           input.Placa,
		"modelo":          input.Modelo,
		"categoria":       input.Categoria,
		"capacidade":      input.Capacidade,
		"cidade_base":     input.CidadeBase,
		"status":          input.Status,
		"ar_condicionado": input.ArCondicionado,
		"banheiro":        input.Banheiro,
		"persiana":        input.Persiana,
		"luz_leitura":     input.LuzLeitura,
		"tomada":          input.Tomada,
	}

	var vehicleID int64
	err := s.db.QueryRow(ctx, q, args).Scan(&vehicleID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Veiculo{
		ID:             vehicleID,
		Placa:          input.Placa,
		Modelo:         input.Modelo,
		Categoria:      input.Categoria,
		Capacidade:     input.Capacidade,
		CidadeBase:     input.CidadeBase,
		Status:         input.Status,
		ArCondicionado: input.ArCondicionado,
		Banheiro:       input.Banheiro,
		Persiana:       input.Persiana,
		LuzLeitura:     input.LuzLeitura,
		Tomada:         input.Tomada,
	}, nil
}

func (s *veiculoStore) GetByID(ctx context.Context, id int64) (*Veiculo, error) {
	const op = "db/veiculoStore.GetByID"

	veiculo, err := getVeiculoByID(ctx, s.db, id, false)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return veiculo, nil
}

func getVeiculoByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, id int64, forUpdate bool) (*Veiculo, error) {
	q := `
		SELECT id, placa, modelo, categoria, capacidade, cidade_base, status, ar_condicionado, banheiro, persiana, luz_leitura, tomada
		FROM veiculos
		WHERE id = @id
	`
	if forUpdate {
		q += " FOR UPDATE"
	}

	args := pgx.StrictNamedArgs{"id": id}

	rows, err := querier.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}

	veiculo, err := pgx.CollectExactlyOneRow(rows, scanVeiculo)
	if err != nil {
		return nil, err
	}

	return &veiculo, nil
}

func (s *veiculoStore) List(ctx context.Context) ([]Veiculo, error) {
	const op = "db/veiculoStore.List"

	const q = `
		SELECT id, placa, modelo, categoria, capacidade, cidade_base, status, ar_condicionado, banheiro, persiana, luz_leitura, tomada
		FROM veiculos
		ORDER BY id DESC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	veiculos, err := pgx.CollectRows(rows, scanVeiculo)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return veiculos, nil
}

func (s *veiculoStore) Update(ctx context.Context, id int64, updateFunc func(*Veiculo) (bool, error)) (*Veiculo, error) {
	const op = "db/veiculoStore.Update"

	var veiculo Veiculo

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		v, err := getVeiculoByIDForUpdate(ctx, tx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("select: %w", err)
		}
		veiculo = *v

		changed, err := updateFunc(&veiculo)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const updateQ = `
			UPDATE veiculos
			SET placa = @placa, modelo = @modelo, categoria = @categoria, capacidade = @capacidade, cidade_base = @cidade_base,
			    status = @status, ar_condicionado = @ar_condicionado, banheiro = @banheiro,
			    persiana = @persiana, luz_leitura = @luz_leitura, tomada = @tomada
			WHERE id = @id
		`
		updateArgs := pgx.StrictNamedArgs{
			"id":              veiculo.ID,
			"placa":           veiculo.Placa,
			"modelo":          veiculo.Modelo,
			"categoria":       veiculo.Categoria,
			"capacidade":      veiculo.Capacidade,
			"cidade_base":     veiculo.CidadeBase,
			"status":          veiculo.Status,
			"ar_condicionado": veiculo.ArCondicionado,
			"banheiro":        veiculo.Banheiro,
			"persiana":        veiculo.Persiana,
			"luz_leitura":     veiculo.LuzLeitura,
			"tomada":          veiculo.Tomada,
		}

		if _, err := tx.Exec(ctx, updateQ, updateArgs); err != nil {
			return fmt.Errorf("update: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &veiculo, nil
}

func getVeiculoByIDForUpdate(ctx context.Context, tx pgx.Tx, id int64) (*Veiculo, error) {
	const q = `
		SELECT id, placa, modelo, categoria, capacidade, cidade_base, status, ar_condicionado, banheiro, persiana, luz_leitura, tomada
		FROM veiculos
		WHERE id = @id
		FOR UPDATE
	`
	args := pgx.StrictNamedArgs{"id": id}

	rows, err := tx.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}

	veiculo, err := pgx.CollectExactlyOneRow(rows, scanVeiculo)
	if err != nil {
		return nil, err
	}

	return &veiculo, nil
}

func (s *veiculoStore) Delete(ctx context.Context, id int64) error {
	const op = "db/veiculoStore.Delete"

	const q = `
		DELETE FROM veiculos
		WHERE id = @id
	`
	args := pgx.StrictNamedArgs{"id": id}

	cmdTag, err := s.db.Exec(ctx, q, args)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func scanVeiculo(row pgx.CollectableRow) (Veiculo, error) {
	var v Veiculo
	err := row.Scan(&v.ID, &v.Placa, &v.Modelo, &v.Categoria, &v.Capacidade, &v.CidadeBase, &v.Status, &v.ArCondicionado, &v.Banheiro, &v.Persiana, &v.LuzLeitura, &v.Tomada)
	return v, err
}
