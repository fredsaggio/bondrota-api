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
		INSERT INTO veiculos (placa, modelo, capacidade, cidade_base, status, ar_condicionado, banheiro, persiana, luz_leitura, tomada)
		VALUES (@placa, @modelo, @capacidade, @cidade_base, @status, @ar_condicionado, @banheiro, @persiana, @luz_leitura, @tomada)
		RETURNING id
	`
	args := pgx.StrictNamedArgs{
		"placa":           input.Placa,
		"modelo":          input.Modelo,
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

	veiculo, err := getVeiculoByID(ctx, s.db, id)
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
}, id int64) (*Veiculo, error) {
	const q = `
		SELECT id, placa, modelo, capacidade, cidade_base, status, ar_condicionado, banheiro, persiana, luz_leitura, tomada
		FROM veiculos
		WHERE id = @id
	`
	args := pgx.StrictNamedArgs{"id": id}

	rows, err := querier.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}

	veiculo, err := pgx.CollectExactlyOneRow(rows, func(row pgx.CollectableRow) (Veiculo, error) {
		var v Veiculo
		err := row.Scan(&v.ID, &v.Placa, &v.Modelo, &v.Capacidade, &v.CidadeBase, &v.Status, &v.ArCondicionado, &v.Banheiro, &v.Persiana, &v.LuzLeitura, &v.Tomada)
		return v, err
	})
	if err != nil {
		return nil, err
	}

	return &veiculo, nil
}
