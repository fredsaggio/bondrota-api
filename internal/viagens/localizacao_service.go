package viagens

import (
	"context"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

type viagemLocalizacaoService struct {
	store ViagemLocalizacaoStore
}

func NewViagemLocalizacaoService(store ViagemLocalizacaoStore) ViagemLocalizacaoService {
	return &viagemLocalizacaoService{store: store}
}

func (s *viagemLocalizacaoService) Atualizar(ctx context.Context, actor ViagemLocalizacaoActor, input ViagemLocalizacaoInput) (*ViagemLocalizacao, error) {
	if err := validateViagemLocalizacaoInput(input); err != nil {
		return nil, err
	}
	if actor.UserID <= 0 {
		return nil, brerror.ErrUnauthenticated
	}

	switch actor.Role {
	case auth.RoleAdmin:
		if input.MotoristaID <= 0 {
			return nil, fmt.Errorf("%w: Motorista não encontrado.", brerror.ErrInvalidInput)
		}
	case auth.RoleMotorista:
		allowed, err := s.store.CanMotoristaAccessViagem(ctx, input.ViagemID, actor.UserID, true)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, brerror.ErrForbidden
		}
		input.MotoristaID = actor.UserID
	default:
		return nil, brerror.ErrForbidden
	}

	if input.RegistradaEm.IsZero() {
		input.RegistradaEm = time.Now().UTC()
	}

	return s.store.Upsert(ctx, input)
}

func (s *viagemLocalizacaoService) GetByViagem(ctx context.Context, actor ViagemLocalizacaoActor, viagemID int64) (*ViagemLocalizacao, error) {
	if viagemID <= 0 {
		return nil, fmt.Errorf("%w: Viagem não encontrada.", brerror.ErrInvalidInput)
	}
	if actor.UserID <= 0 {
		return nil, brerror.ErrUnauthenticated
	}

	switch actor.Role {
	case auth.RoleAdmin:
	case auth.RoleMotorista:
		allowed, err := s.store.CanMotoristaAccessViagem(ctx, viagemID, actor.UserID, false)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, brerror.ErrForbidden
		}
	case auth.RoleCliente:
		allowed, err := s.store.CanClienteAccessViagem(ctx, viagemID, actor.UserID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, brerror.ErrForbidden
		}
	default:
		return nil, brerror.ErrForbidden
	}

	return s.store.GetByViagem(ctx, viagemID)
}

func validateViagemLocalizacaoInput(input ViagemLocalizacaoInput) error {
	if input.ViagemID <= 0 {
		return fmt.Errorf("%w: Viagem não encontrada.", brerror.ErrInvalidInput)
	}
	if input.Latitude < -90 || input.Latitude > 90 {
		return fmt.Errorf("%w: Localização inválida.", brerror.ErrInvalidInput)
	}
	if input.Longitude < -180 || input.Longitude > 180 {
		return fmt.Errorf("%w: Localização inválida.", brerror.ErrInvalidInput)
	}
	if input.VelocidadeKmh < 0 {
		return fmt.Errorf("%w: Velocidade inválida.", brerror.ErrInvalidInput)
	}
	if input.DirecaoGraus < 0 || input.DirecaoGraus > 360 {
		return fmt.Errorf("%w: Direção inválida.", brerror.ErrInvalidInput)
	}
	if input.PrecisaoMetros < 0 {
		return fmt.Errorf("%w: Precisão de localização inválida.", brerror.ErrInvalidInput)
	}
	return nil
}
