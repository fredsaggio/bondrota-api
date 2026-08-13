package clientes

import (
	"context"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/auth"
)

type clienteService struct {
	store   ClienteStore
	authSvc *auth.AuthService
}

func NewClienteService(store ClienteStore, authSvc *auth.AuthService) ClienteService {
	return &clienteService{
		store:   store,
		authSvc: authSvc,
	}
}

func (s *clienteService) Login(ctx context.Context, cpf, senha string) (string, error) {
	const op = "service/clienteService.Login"

	cliente, err := s.store.GetByCPF(ctx, cpf)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	ok, err := s.authSvc.CheckPassword(cliente.Senha, senha)
	if err != nil || !ok {
		return "", auth.ErrInvalidCredentials
	}

	token, err := s.authSvc.GenerateToken(cliente.PublicID, auth.RoleCliente)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

func (s *clienteService) Create(ctx context.Context, input ClienteInput) (*Cliente, error) {
	const op = "service/clienteService.Create"

	hashed, err := s.authSvc.HashPassword(input.Senha)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	input.Senha = hashed

	return s.store.Create(ctx, input)
}

func (s *clienteService) GetByID(ctx context.Context, clienteID int64) (*ClienteComVinculos, error) {
	return s.store.GetByID(ctx, clienteID)
}

func (s *clienteService) List(ctx context.Context, params ClienteListParams) (ClienteListResult, error) {
	return s.store.List(ctx, params)
}

func (s *clienteService) Resumo(ctx context.Context) (ClienteResumo, error) {
	return s.store.Resumo(ctx)
}

func (s *clienteService) Update(ctx context.Context, clienteID int64, updateFunc func(*Cliente) (bool, error)) (*Cliente, error) {
	return s.store.Update(ctx, clienteID, updateFunc)
}

func (s *clienteService) Delete(ctx context.Context, clienteID int64) error {
	return s.store.Delete(ctx, clienteID)
}
