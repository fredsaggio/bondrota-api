package viagens

import (
	"context"
	"errors"
	"time"
)

type planejamentoService struct {
	cicloStore CicloViagemStore
}

func NewPlanejamentoService(cicloStore CicloViagemStore) PlanejamentoService {
	return &planejamentoService{
		cicloStore: cicloStore,
	}
}

func (s *planejamentoService) Planejar(ctx context.Context, input CicloViagemInput, partidas map[SentidoViagem]time.Time) (*CicloComViagens, error) {
	if err := validatePlanejamentoInput(input, partidas); err != nil {
		return nil, err
	}

	return s.cicloStore.CreateCicloComViagens(ctx, input, partidas)
}

func validatePlanejamentoInput(input CicloViagemInput, partidas map[SentidoViagem]time.Time) error {
	if input.DataViagem.IsZero() {
		return errors.New("data_viagem is required")
	}
	if !isOperationalTurnoViagem(input.Turno) {
		return errors.New("turno must be MT, VT or NT")
	}
	if input.Cidade == "" {
		return errors.New("cidade is required")
	}
	if input.RotaInternaID <= 0 {
		return errors.New("rota_interna_id is required")
	}
	if input.VeiculoID <= 0 {
		return errors.New("veiculo_id is required")
	}
	if input.MotoristaID <= 0 {
		return errors.New("motorista_id is required")
	}
	if input.ExpiresAt.IsZero() {
		return errors.New("expires_at is required")
	}
	if partidas == nil {
		return errors.New("partidas is required")
	}
	if partidas[SentidoIda].IsZero() {
		return errors.New("partida ida is required")
	}
	if partidas[SentidoVolta].IsZero() {
		return errors.New("partida volta is required")
	}
	if !partidas[SentidoVolta].After(partidas[SentidoIda]) {
		return errors.New("partida volta must be after partida ida")
	}

	return nil
}

func isOperationalTurnoViagem(turno TurnoViagem) bool {
	return turno == TurnoMatutino || turno == TurnoVespertino || turno == TurnoNoturno
}
