package motoristas

import (
	"context"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type motoristaStore struct {
	db db.DB
}

func NewMotoristaStore(db db.DB) MotoristaStore {
	return &motoristaStore{db: db}
}

func (s *motoristaStore) Create(ctx context.Context, input MotoristaInput) (*Motorista, error) {
	const op = "db/motoristaStore.Create"

	const q = `
		INSERT INTO motoristas (nome, cpf, senha, telefone, data_nasc, turno, cidade_trabalho, residencia, foto)
		VALUES (@nome, @cpf, @senha, @telefone, @data_nasc, @turno, @cidade_trabalho, @residencia, @foto)
		RETURNING id, nome, cpf, telefone, data_nasc, turno, cidade_trabalho, residencia, foto
	`
	args := pgx.StrictNamedArgs{
		"nome":            input.Nome,
		"cpf":             input.CPF,
		"senha":           input.Senha,
		"telefone":        input.Telefone,
		"data_nasc":       input.DataNasc,
		"turno":           input.Turno,
		"cidade_trabalho": input.CidadeTrabalho,
		"residencia":      input.Residencia,
		"foto":            input.Foto,
	}

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	motorista, err := pgx.CollectExactlyOneRow(rows, scanMotorista)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &motorista, nil
}

func scanMotorista(row pgx.CollectableRow) (Motorista, error) {
	var m Motorista
	err := row.Scan(
		&m.ID, &m.Nome, &m.CPF, &m.Telefone,
		&m.DataNasc, &m.Turno, &m.CidadeTrabalho, &m.Residencia, &m.Foto,
	)
	return m, err
}
