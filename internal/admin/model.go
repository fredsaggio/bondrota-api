package admin

import "context"

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
	Update(ctx context.Context, adminID int, updateFunc func(*Admin) (bool, error)) (*Admin, error)
	GetByID(ctx context.Context, adminID int) (*Admin, error)
	Delete(ctx context.Context, adminID int) (*Admin, error)
	List(ctx context.Context) ([]Admin, error)
}
