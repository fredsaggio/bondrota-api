package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type adminStore struct {
	db db.DB
}

func NewAdminStore(db db.DB) *adminStore {
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

func (s *adminStore) Update(ctx context.Context, adminID int64, updateFunc func(*Admin) (bool, error)) (*Admin, error) {
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

func (s *adminStore) GetByID(ctx context.Context, adminID int64) (*Admin, error) {
	const op = "db/adminStore.GetByID"

	admin, err := getAdminByID(ctx, s.db, adminID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return admin, nil
}

func getAdminByID(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, adminID int64) (*Admin, error) {
	const q = `
		SELECT id, email, senha, cidade
		FROM administrador
		WHERE id = @id
	`
	args := pgx.StrictNamedArgs{"id": adminID}

	rows, err := querier.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}

	admin, err := pgx.CollectExactlyOneRow(rows, func(row pgx.CollectableRow) (Admin, error) {
		var admin Admin
		err := row.Scan(&admin.ID, &admin.Email, &admin.Senha, &admin.Cidade)
		return admin, err
	})
	if err != nil {
		return nil, err
	}

	return &admin, nil
}

func (s *adminStore) GetByEmail(ctx context.Context, email string) (*Admin, error) {
	const op = "db/adminStore.GetByEmail"

	admin, err := getAdminByEmail(ctx, s.db, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return admin, nil
}

func getAdminByEmail(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, email string) (*Admin, error) {
	const q = `
		SELECT id, email, senha, cidade
		FROM administrador
		WHERE email = @email
	`
	args := pgx.StrictNamedArgs{"email": email}

	rows, err := querier.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}

	admin, err := pgx.CollectExactlyOneRow(rows, func(row pgx.CollectableRow) (Admin, error) {
		var admin Admin
		err := row.Scan(&admin.ID, &admin.Email, &admin.Senha, &admin.Cidade)
		return admin, err
	})
	if err != nil {
		return nil, err
	}

	return &admin, nil
}

func (s *adminStore) Delete(ctx context.Context, adminID int64) (error) {
	const op = "db/adminStore.Delete"
	
	const q = `
		DELETE FROM administrador
		WHERE id = @id
	`
	args := pgx.StrictNamedArgs{"id": adminID}

	cmdTag, err := s.db.Exec(ctx, q, args)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *adminStore) List(ctx context.Context) ([]Admin, error) {
	const op = "db/adminStore.List"

	const q = `
		SELECT id, email, senha, cidade
		FROM administrador
	`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	admins, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Admin, error) {
		var admin Admin
		err := row.Scan(&admin.ID, &admin.Email, &admin.Senha, &admin.Cidade)
		return admin, err
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return admins, nil
}