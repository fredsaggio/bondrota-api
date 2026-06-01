package admin

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("admin not found")

type Admin struct {
	ID     int64  `json:"id"`
	Email  string `json:"email"`
	Senha  string `json:"-"`
	Cidade string `json:"cidade"`
}

type AdminInput struct {
	Email  string `json:"email"`
	Senha  string `json:"senha"`
	Cidade string `json:"cidade"`
}

type AdminStore interface {
	Create(ctx context.Context, input AdminInput) (*Admin, error)
	Update(ctx context.Context, adminID int64, updateFunc func(*Admin) (bool, error)) (*Admin, error)
	GetByID(ctx context.Context, adminID int64) (*Admin, error)
	GetByEmail(ctx context.Context, email string) (*Admin, error)
	Delete(ctx context.Context, adminID int64) (error)
	List(ctx context.Context) ([]Admin, error)
}
