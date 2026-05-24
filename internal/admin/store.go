package admin

import "context"

type AdminStore interface {
	Create(ctx context.Context, input AdminInput) (*Admin, error)
	Update(ctx context.Context, adminID int, updateFunc func(*Admin) (bool, error)) (*Admin, error)
	GetByID(ctx context.Context, adminID int) (*Admin, error)
	Delete(ctx context.Context, adminID int) (*Admin, error)
	List(ctx context.Context) ([]Admin, error)
}
