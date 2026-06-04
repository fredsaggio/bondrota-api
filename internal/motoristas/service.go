package motoristas

import (
	"context"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/auth"
)

type motoristaService struct {
	store   MotoristaStore
	authSvc *auth.AuthService
}

func NewMotoristaService(store MotoristaStore, authSvc *auth.AuthService) MotoristaService {
	return &motoristaService{
		store:   store,
		authSvc: authSvc,
	}
}

func (s *motoristaService) Login(ctx context.Context, cpf, senha string) (string, error) {
	const op = "service/motoristaService.Login"

	motorista, err := s.store.GetByCPF(ctx, cpf)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	ok, err := s.authSvc.CheckPassword(motorista.Senha, senha)
	if err != nil || !ok {
		return "", auth.ErrInvalidCredentials
	}

	token, err := s.authSvc.GenerateToken(motorista.ID, "motorista")
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

func (s *motoristaService) Create(ctx context.Context, input MotoristaInput) (*Motorista, error) {
	const op = "service/motoristaService.Create"

	hashed, err := s.authSvc.HashPassword(input.Senha)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	input.Senha = hashed

	return s.store.Create(ctx, input)
}

func (s *motoristaService) GetByID(ctx context.Context, motoristaID int64) (*Motorista, error) {
	return s.store.GetByID(ctx, motoristaID)
}

func (s *motoristaService) List(ctx context.Context) ([]Motorista, error) {
	return s.store.List(ctx)
}

func (s *motoristaService) Update(ctx context.Context, motoristaID int64, updateFunc func(*Motorista) (bool, error)) (*Motorista, error) {
	return s.store.Update(ctx, motoristaID, updateFunc)
}

func (s *motoristaService) Delete(ctx context.Context, motoristaID int64) error {
	return s.store.Delete(ctx, motoristaID)
}