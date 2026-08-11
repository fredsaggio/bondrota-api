package admin

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("admin not found")

// MinPasswordLen vale para todo caminho que define senha de administrador: o painel
// e o cmd/admin. Mantendo a regra aqui, nenhum dos dois fica mais permissivo que o
// outro sem que a diferenca seja proposital.
const MinPasswordLen = 8

var ErrSenhaFraca = fmt.Errorf("a senha precisa de pelo menos %d caracteres", MinPasswordLen)

func ValidarSenha(senha string) error {
	if len([]rune(senha)) < MinPasswordLen {
		return ErrSenhaFraca
	}
	return nil
}

type Admin struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Senha string `json:"-"`
}

type AdminInput struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type AdminStore interface {
	Create(ctx context.Context, input AdminInput) (*Admin, error)
	Update(ctx context.Context, adminID int64, updateFunc func(*Admin) (bool, error)) (*Admin, error)
	GetByID(ctx context.Context, adminID int64) (*Admin, error)
	GetByEmail(ctx context.Context, email string) (*Admin, error)
	Delete(ctx context.Context, adminID int64) error
	List(ctx context.Context) ([]Admin, error)
}

type AdminService interface {
	Login(ctx context.Context, email, password string) (string, error)
	// ChangePassword troca a senha do proprio admin autenticado e devolve um token
	// novo. O adminID vem sempre do JWT, nunca do corpo ou do path.
	ChangePassword(ctx context.Context, adminID int64, senhaAtual, novaSenha string) (string, error)
	Create(ctx context.Context, input AdminInput) (*Admin, error)
	Update(ctx context.Context, adminID int64, email string) (*Admin, error)
	GetByID(ctx context.Context, adminID int64) (*Admin, error)
	Delete(ctx context.Context, adminID int64) error
	List(ctx context.Context) ([]Admin, error)
}
