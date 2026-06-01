package admin

import (
	"context"
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
