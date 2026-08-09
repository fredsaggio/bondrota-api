package viagens

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
)

type planejamentoService struct {
	cicloStore        CicloViagemStore
	horarioStore      HorarioTurnoViagemStore
	veiculoAlocador   VeiculoAlocador
	motoristaAlocador MotoristaAlocador
}

func NewPlanejamentoService(cicloStore CicloViagemStore, horarioStore HorarioTurnoViagemStore, veiculoAlocador VeiculoAlocador, motoristaAlocador MotoristaAlocador) PlanejamentoService {
	return &planejamentoService{
		cicloStore:        cicloStore,
		horarioStore:      horarioStore,
		veiculoAlocador:   veiculoAlocador,
		motoristaAlocador: motoristaAlocador,
	}
}

func (s *planejamentoService) Planejar(ctx context.Context, input PlanejamentoViagensInput) (*PlanejamentoViagens, error) {
	if err := validatePlanejamentoInput(input); err != nil {
		return nil, err
	}
	input.ExpiresAt = calcularExpiresAtPlanejamento(input.DataViagem)

	horarioTurno, err := s.horarioStore.GetByCidadeTurno(ctx, input.Cidade, input.Turno)
	if err != nil {
		return nil, err
	}
	partidas := montarPartidasPlanejamento(input.DataViagem, horarioTurno)

	reservasIda, err := s.cicloStore.ListReservasConfirmadasParaPlanejamento(ctx, PlanejamentoReservasFiltro{
		DataViagem:    input.DataViagem,
		Turno:         input.Turno,
		Cidade:        input.Cidade,
		RotaInternaID: input.RotaInternaID,
		Sentido:       SentidoIda,
	})
	if err != nil {
		return nil, err
	}

	reservasVolta, err := s.cicloStore.ListReservasConfirmadasParaPlanejamento(ctx, PlanejamentoReservasFiltro{
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

func validatePlanejamentoInput(input PlanejamentoViagensInput) error {
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

	return nil
}

func calcularExpiresAtPlanejamento(dataViagem time.Time) time.Time {
	return dataViagem.AddDate(0, 3, 0)
}

func montarPartidasPlanejamento(dataViagem time.Time, horario *HorarioTurnoViagem) map[SentidoViagem]time.Time {
	return map[SentidoViagem]time.Time{
		SentidoIda:   combinarDataHorario(dataViagem, horario.HorarioIda),
		SentidoVolta: combinarDataHorario(dataViagem, horario.HorarioVolta),
	}
}

func combinarDataHorario(data time.Time, horario time.Duration) time.Time {
	hours := int(horario / time.Hour)
	horario -= time.Duration(hours) * time.Hour
	minutes := int(horario / time.Minute)
	horario -= time.Duration(minutes) * time.Minute
	seconds := int(horario / time.Second)
	nanoseconds := int(horario - time.Duration(seconds)*time.Second)

	return time.Date(data.Year(), data.Month(), data.Day(), hours, minutes, seconds, nanoseconds, data.Location())
}

func montarCiclosComReservasInput(input PlanejamentoViagensInput, veiculosAlocados []veiculos.Veiculo, motoristasAlocados []motoristas.Motorista, reservasIda, reservasVolta []PlanejamentoReserva) []CicloViagemComReservasInput {
	result := make([]CicloViagemComReservasInput, 0, len(veiculosAlocados))
	reservasIdaPorVeiculo := distribuirReservasPorDestinoECapacidade(reservasIda, veiculosAlocados)
	reservasVoltaPorVeiculo := distribuirReservasPorDestinoECapacidade(reservasVolta, veiculosAlocados)

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

func distribuirReservasPorDestinoECapacidade(reservas []PlanejamentoReserva, veiculosAlocados []veiculos.Veiculo) [][]int64 {
	result := make([][]int64, len(veiculosAlocados))
	capacidadesRestantes := make([]int, len(veiculosAlocados))
	for i, veiculo := range veiculosAlocados {
		capacidadesRestantes[i] = int(veiculo.Capacidade)
	}

	grupos := agruparReservasPorDestino(reservas)
	for _, grupo := range grupos {
		if alocarGrupoInteiroPorDestino(result, capacidadesRestantes, grupo) {
			continue
		}
		alocarGrupoDivididoPorCapacidade(result, capacidadesRestantes, grupo)
	}

	for i := range result {
		if result[i] == nil {
			result[i] = []int64{}
		}
	}

	return result
}

type grupoReservasDestino struct {
	destinoID int64
	reservas  []PlanejamentoReserva
}

func agruparReservasPorDestino(reservas []PlanejamentoReserva) []grupoReservasDestino {
	porDestino := make(map[int64][]PlanejamentoReserva)
	for _, reserva := range reservas {
		porDestino[reserva.DestinoID] = append(porDestino[reserva.DestinoID], reserva)
	}

	grupos := make([]grupoReservasDestino, 0, len(porDestino))
	for destinoID, reservasDestino := range porDestino {
		grupos = append(grupos, grupoReservasDestino{
			destinoID: destinoID,
			reservas:  reservasDestino,
		})
	}

	sort.Slice(grupos, func(i, j int) bool {
		if len(grupos[i].reservas) != len(grupos[j].reservas) {
			return len(grupos[i].reservas) > len(grupos[j].reservas)
		}
		return grupos[i].destinoID < grupos[j].destinoID
	})

	return grupos
}

func alocarGrupoInteiroPorDestino(result [][]int64, capacidadesRestantes []int, grupo grupoReservasDestino) bool {
	veiculoIndex := -1
	menorSobra := 0
	quantidade := len(grupo.reservas)

	for i, capacidadeRestante := range capacidadesRestantes {
		if capacidadeRestante < quantidade {
			continue
		}

		sobra := capacidadeRestante - quantidade
		if veiculoIndex == -1 || sobra < menorSobra {
			veiculoIndex = i
			menorSobra = sobra
		}
	}

	if veiculoIndex == -1 {
		return false
	}

	for _, reserva := range grupo.reservas {
		result[veiculoIndex] = append(result[veiculoIndex], reserva.ID)
	}
	capacidadesRestantes[veiculoIndex] -= quantidade

	return true
}

func alocarGrupoDivididoPorCapacidade(result [][]int64, capacidadesRestantes []int, grupo grupoReservasDestino) {
	reservaIndex := 0
	for reservaIndex < len(grupo.reservas) {
		veiculoIndex := veiculoComMaiorCapacidadeRestante(capacidadesRestantes)
		if veiculoIndex == -1 {
			return
		}

		for capacidadesRestantes[veiculoIndex] > 0 && reservaIndex < len(grupo.reservas) {
			result[veiculoIndex] = append(result[veiculoIndex], grupo.reservas[reservaIndex].ID)
			capacidadesRestantes[veiculoIndex]--
			reservaIndex++
		}
	}
}

func veiculoComMaiorCapacidadeRestante(capacidadesRestantes []int) int {
	veiculoIndex := -1
	maiorCapacidade := 0

	for i, capacidadeRestante := range capacidadesRestantes {
		if capacidadeRestante > maiorCapacidade {
			veiculoIndex = i
			maiorCapacidade = capacidadeRestante
		}
	}

	return veiculoIndex
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
