package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type adminStore struct {
	db *pgxpool.Pool
}

func NewAdminStore(db *pgxpool.Pool) *adminStore {
	return &adminStore{
		db: db,
	}
}

func (s *adminStore) Create(ctx context.Context, input AdminInput) (*Admin, error) {
	const op = "db/adminStore.Create"
	var adminID int64

	const q = `
		INSERT INTO administrador (email, senha, cidade)
		VALUES (@email, @senha, @cidade)
		RETURNING id
	`
	args := pgx.StrictNamedArgs{
		"email":  input.Email,
		"senha":  input.Senha,
		"cidade": input.Cidade,
	}

	err := s.db.QueryRow(ctx, q, args).Scan(&adminID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Admin{
		ID:     adminID,
		Email:  input.Email,
		Cidade: input.Cidade,
	}, nil
}

func (s *adminStore) Update(ctx context.Context, adminID int, updateFunc func(*Admin) (bool, error)) (*Admin, error) {
	const op = "db/adminStore.Update"

	var admin Admin

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		const q = `
			SELECT id, email, senha, cidade
			FROM administrador
			WHERE id = @id
			FOR UPDATE
		`
		args := pgx.StrictNamedArgs{"id": adminID}
		err := tx.QueryRow(ctx, q, args).Scan(&admin.ID, &admin.Email, &admin.Senha, &admin.Cidade)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("select: %w", err)
		}

		updated, err := updateFunc(&admin)
		if err != nil {
			return err
		}

		if !updated {
			return nil
		}

		const updateQuery = `
			UPDATE administrador
			SET email = @email, senha = @senha, cidade = @cidade
			WHERE id = @id
		`
		updateArgs := pgx.StrictNamedArgs{
			"id":     admin.ID,
			"email":  admin.Email,
			"senha":  admin.Senha,
			"cidade": admin.Cidade,
		}
		if _, err := tx.Exec(ctx, updateQuery, updateArgs); err != nil {
			return fmt.Errorf("update: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &admin, nil
}
