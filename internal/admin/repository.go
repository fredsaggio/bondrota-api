package admin

import "github.com/jackc/pgx/v5/pgxpool"

type adminStore struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *adminStore {
    return &adminStore{
        db: db,
    }
}