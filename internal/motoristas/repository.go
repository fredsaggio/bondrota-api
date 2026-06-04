package motoristas

import (
	"context"
	"errors"
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

func (s *motoristaStore) GetByID(ctx context.Context, motoristaID int64) (*Motorista, error) {
	const op = "db/motoristaStore.GetByID"

	const q = `
		SELECT id, nome, cpf, telefone, data_nasc, turno, cidade_trabalho, residencia, foto
		FROM motoristas
		WHERE id = @id
	`
	args := pgx.StrictNamedArgs{"id": motoristaID}

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	motorista, err := pgx.CollectExactlyOneRow(rows, scanMotorista)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &motorista, nil
}

func (s *motoristaStore) List(ctx context.Context) ([]Motorista, error) {
	const op = "db/motoristaStore.List"

	const q = `
		SELECT id, nome, cpf, telefone, data_nasc, turno, cidade_trabalho, residencia, foto
		FROM motoristas
		ORDER BY id DESC
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	motoristas, err := pgx.CollectRows(rows, scanMotorista)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return motoristas, nil
}

func (s *motoristaStore) Update(ctx context.Context, motoristaID int64, updateFunc func(*Motorista) (bool, error)) (*Motorista, error) {
	const op = "db/motoristaStore.Update"

	var motorista Motorista

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const selectQ = `
			SELECT id, nome, cpf, telefone, data_nasc, turno, cidade_trabalho, residencia, foto
			FROM motoristas
			WHERE id = @id
			FOR UPDATE
		`
		rows, err := tx.Query(ctx, selectQ, pgx.StrictNamedArgs{"id": motoristaID})
		if err != nil {
			return fmt.Errorf("select: %w", err)
		}

		m, err := pgx.CollectExactlyOneRow(rows, scanMotorista)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("select: %w", err)
		}

		motorista = m

		changed, err := updateFunc(&motorista)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}

		const updateQ = `
			UPDATE motoristas
			SET nome = @nome, cpf = @cpf, telefone = @telefone,
			    data_nasc = @data_nasc, turno = @turno,
			    cidade_trabalho = @cidade_trabalho, residencia = @residencia,
			    foto = @foto
			WHERE id = @id
		`
		_, err = tx.Exec(ctx, updateQ, pgx.StrictNamedArgs{
			"id":              motorista.ID,
			"nome":            motorista.Nome,
			"cpf":             motorista.CPF,
			"telefone":        motorista.Telefone,
			"data_nasc":       motorista.DataNasc,
			"turno":           motorista.Turno,
			"cidade_trabalho": motorista.CidadeTrabalho,
			"residencia":      motorista.Residencia,
			"foto":            motorista.Foto,
		})
		if err != nil {
			return fmt.Errorf("update: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &motorista, nil
}

func (s *motoristaStore) Delete(ctx context.Context, motoristaID int64) error {
	const op = "db/motoristaStore.Delete"

	const q = `DELETE FROM motoristas WHERE id = @id`

	cmdTag, err := s.db.Exec(ctx, q, pgx.StrictNamedArgs{"id": motoristaID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *motoristaStore) GetByCPF(ctx context.Context, cpf string) (*Motorista, error) {
	const op = "db/motoristaStore.GetByCPF"

	const q = `
		SELECT id, nome, cpf, senha, telefone, data_nasc, turno, cidade_trabalho, residencia, foto
		FROM motoristas
		WHERE cpf = @cpf
	`
	args := pgx.StrictNamedArgs{"cpf": cpf}

	rows, err := s.db.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	m, err := pgx.CollectExactlyOneRow(rows, scanMotoristaComSenha)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &m, nil
}

func scanMotoristaComSenha(row pgx.CollectableRow) (Motorista, error) {
	var m Motorista
	err := row.Scan(
		&m.ID, &m.Nome, &m.CPF, &m.Senha, &m.Telefone,
		&m.DataNasc, &m.Turno, &m.CidadeTrabalho, &m.Residencia, &m.Foto,
	)
	return m, err
}

func scanMotorista(row pgx.CollectableRow) (Motorista, error) {
	var m Motorista
	err := row.Scan(
		&m.ID, &m.Nome, &m.CPF, &m.Telefone,
		&m.DataNasc, &m.Turno, &m.CidadeTrabalho, &m.Residencia, &m.Foto,
	)
	return m, err
}
