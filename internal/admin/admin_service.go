package admin

import (
	"context"

	"github.com/fredsaggio/bondrota-api/internal/auth"
)

type adminService struct {
	store   AdminStore
	authSvc *auth.AuthService
}

func NewAdminService(store AdminStore, authSvc *auth.AuthService) AdminService {
	return &adminService{
		store:   store,
		authSvc: authSvc,
	}
}

func (s *adminService) Login(ctx context.Context, email, password string) (string, error) {
	admin, err := s.store.GetByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	ok, err := s.authSvc.CheckPassword(admin.Senha, password)
	if err != nil || !ok {
		return "", auth.ErrInvalidCredentials
	}

	token, err := s.authSvc.GenerateToken(admin.ID, auth.RoleAdmin)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *adminService) ChangePassword(ctx context.Context, adminID int64, senhaAtual, novaSenha string) (string, error) {
	if err := ValidarSenha(novaSenha); err != nil {
		return "", err
	}

	hashed, err := s.authSvc.HashPassword(novaSenha)
	if err != nil {
		return "", err
	}

	// A conferencia da senha atual acontece dentro do Update porque so ali a linha
	// esta travada (SELECT ... FOR UPDATE) e o hash gravado e o mesmo que foi lido.
	// Checar antes abriria janela para duas trocas simultaneas se perderem.
	_, err = s.store.Update(ctx, adminID, func(a *Admin) (bool, error) {
		ok, err := s.authSvc.CheckPassword(a.Senha, senhaAtual)
		if err != nil || !ok {
			return false, auth.ErrInvalidCredentials
		}
		a.Senha = hashed
		return true, nil
	})
	if err != nil {
		return "", err
	}

	return s.authSvc.GenerateToken(adminID, auth.RoleAdmin)
}

func (s *adminService) Create(ctx context.Context, input AdminInput) (*Admin, error) {
	hashed, err := s.authSvc.HashPassword(input.Senha)
	if err != nil {
		return nil, err
	}
	input.Senha = hashed
	return s.store.Create(ctx, input)
}

func (s *adminService) Update(ctx context.Context, adminID int64, email string) (*Admin, error) {
	return s.store.Update(ctx, adminID, func(a *Admin) (bool, error) {
		var updated bool

		if email != "" && email != a.Email {
			a.Email = email
			updated = true
		}

		return updated, nil
	})
}

func (s *adminService) GetByID(ctx context.Context, adminID int64) (*Admin, error) {
	return s.store.GetByID(ctx, adminID)
}

func (s *adminService) Delete(ctx context.Context, adminID int64) error {
	return s.store.Delete(ctx, adminID)
}

func (s *adminService) List(ctx context.Context) ([]Admin, error) {
	return s.store.List(ctx)
}
