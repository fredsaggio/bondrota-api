package viagens

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
)

type planejamentoService struct {
	cicloStore        CicloViagemStore
	veiculoAlocador   VeiculoAlocador
	motoristaAlocador MotoristaAlocador
}

func NewPlanejamentoService(cicloStore CicloViagemStore, veiculoAlocador VeiculoAlocador, motoristaAlocador MotoristaAlocador) PlanejamentoService {
	return &planejamentoService{
		cicloStore:        cicloStore,
		veiculoAlocador:   veiculoAlocador,
		motoristaAlocador: motoristaAlocador,
	}
}

func (s *planejamentoService) Planejar(ctx context.Context, input PlanejamentoViagensInput, partidas map[SentidoViagem]time.Time) (*PlanejamentoViagens, error) {
	if err := validatePlanejamentoInput(input, partidas); err != nil {
		return nil, err
	}

	reservasIda, err := s.cicloStore.ListReservaIDsConfirmadasParaPlanejamento(ctx, PlanejamentoReservasFiltro{
		DataViagem:    input.DataViagem,
		Turno:         input.Turno,
		Cidade:        input.Cidade,
		RotaInternaID: input.RotaInternaID,
		Sentido:       SentidoIda,
	})
	if err != nil {
		return nil, err
	}

	reservasVolta, err := s.cicloStore.ListReservaIDsConfirmadasParaPlanejamento(ctx, PlanejamentoReservasFiltro{
		DataViagem:    input.DataViagem,
		Turno:         input.Turno,
		Cidade:        input.Cidade,
		RotaInternaID: input.RotaInternaID,
		Sentido:       SentidoVolta,
	})
	if err != nil {
		return nil, err
	}

	qtdAlunos := maxInt(len(reservasIda), len(reservasVolta))
	if qtdAlunos == 0 {
		return nil, fmt.Errorf("%w: no confirmed reservations found for planejamento", brerror.ErrNotFound)
	}

	alocacaoVeiculos, err := s.veiculoAlocador.Alocar(ctx, veiculos.AlocarVeiculosInput{
		Cidade:           input.Cidade,
		DataViagem:       input.DataViagem,
		Turno:            string(input.Turno),
		QuantidadeAlunos: qtdAlunos,
	})
	if err != nil {
		return nil, err
	}

	alocacaoMotoristas, err := s.motoristaAlocador.Alocar(ctx, motoristas.AlocarMotoristasInput{
		Cidade:     input.Cidade,
		DataViagem: input.DataViagem,
		Turno:      motoristas.Turno(input.Turno),
		Quantidade: len(alocacaoVeiculos.Veiculos),
	})
	if err != nil {
		return nil, err
	}

	ciclosInput := montarCiclosComReservasInput(input, alocacaoVeiculos.Veiculos, alocacaoMotoristas, reservasIda, reservasVolta)
	planejamento, err := s.cicloStore.CreateCiclosComViagens(ctx, ciclosInput, partidas)
	if err != nil {
		return nil, err
	}

	planejamento.QuantidadeReservasIda = len(reservasIda)
	planejamento.QuantidadeReservasVolta = len(reservasVolta)
	planejamento.CapacidadeTotal = alocacaoVeiculos.CapacidadeTotal

	return planejamento, nil
}

func validatePlanejamentoInput(input PlanejamentoViagensInput, partidas map[SentidoViagem]time.Time) error {
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

func montarCiclosComReservasInput(input PlanejamentoViagensInput, veiculosAlocados []veiculos.Veiculo, motoristasAlocados []motoristas.Motorista, reservasIda, reservasVolta []int64) []CicloViagemComReservasInput {
	result := make([]CicloViagemComReservasInput, 0, len(veiculosAlocados))
	reservasIdaPorVeiculo := distribuirReservasPorCapacidade(reservasIda, veiculosAlocados)
	reservasVoltaPorVeiculo := distribuirReservasPorCapacidade(reservasVolta, veiculosAlocados)

	for i, veiculo := range veiculosAlocados {
		result = append(result, CicloViagemComReservasInput{
			Ciclo: CicloViagemInput{
				DataViagem:    input.DataViagem,
				Turno:         input.Turno,
				Cidade:        input.Cidade,
				RotaInternaID: input.RotaInternaID,
				VeiculoID:     veiculo.ID,
				MotoristaID:   motoristasAlocados[i].ID,
				ExpiresAt:     input.ExpiresAt,
			},
			ReservaIDsIda:   reservasIdaPorVeiculo[i],
			ReservaIDsVolta: reservasVoltaPorVeiculo[i],
		})
	}

	return result
}

func distribuirReservasPorCapacidade(reservaIDs []int64, veiculosAlocados []veiculos.Veiculo) [][]int64 {
	result := make([][]int64, len(veiculosAlocados))
	offset := 0

	for i, veiculo := range veiculosAlocados {
		if offset >= len(reservaIDs) {
			result[i] = []int64{}
			continue
		}

		limit := offset + int(veiculo.Capacidade)
		if limit > len(reservaIDs) {
			limit = len(reservaIDs)
		}
		result[i] = append([]int64(nil), reservaIDs[offset:limit]...)
		offset = limit
	}

	return result
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isOperationalTurnoViagem(turno TurnoViagem) bool {
	return turno == TurnoMatutino || turno == TurnoVespertino || turno == TurnoNoturno
}
