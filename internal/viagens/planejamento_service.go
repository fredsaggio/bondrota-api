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
	location          *time.Location
}

func NewPlanejamentoService(cicloStore CicloViagemStore, horarioStore HorarioTurnoViagemStore, veiculoAlocador VeiculoAlocador, motoristaAlocador MotoristaAlocador, config PlanejamentoServiceConfig) PlanejamentoService {
	if config.Location == nil {
		config.Location = time.UTC
	}
	return &planejamentoService{
		cicloStore:        cicloStore,
		horarioStore:      horarioStore,
		veiculoAlocador:   veiculoAlocador,
		motoristaAlocador: motoristaAlocador,
		location:          config.Location,
	}
}

func (s *planejamentoService) Planejar(ctx context.Context, input PlanejamentoViagensInput) (*PlanejamentoViagens, error) {
	if err := validatePlanejamentoInput(input); err != nil {
		return nil, err
	}
	input.ExpiresAt = calcularExpiresAtPlanejamento(input.DataViagem, s.location)

	horarioTurno, err := s.horarioStore.GetByMunicipioDestinoTurno(ctx, input.MunicipioDestinoID, input.Turno)
	if err != nil {
		return nil, err
	}
	partida := montarPartidaPlanejamento(input.DataViagem, horarioTurno, input.Sentido, s.location)

	if input.Sentido == SentidoIda {
		return s.planejarIda(ctx, input, partida)
	}
	return s.planejarVolta(ctx, input, partida)
}

func (s *planejamentoService) planejarIda(ctx context.Context, input PlanejamentoViagensInput, partida time.Time) (*PlanejamentoViagens, error) {
	reservas, err := s.cicloStore.ListReservasConfirmadasParaPlanejamento(ctx, PlanejamentoReservasFiltro{
		DataViagem:         input.DataViagem,
		Turno:              input.Turno,
		MunicipioDestinoID: input.MunicipioDestinoID,
		RotaInternaID:      input.RotaInternaID,
		Sentido:            SentidoIda,
	})
	if err != nil {
		return nil, err
	}
	if len(reservas) == 0 {
		return nil, ErrSemDemandaPlanejamento
	}

	alocacaoVeiculos, err := s.veiculoAlocador.Alocar(ctx, veiculos.AlocarVeiculosInput{
		DataViagem:       input.DataViagem,
		Turno:            string(input.Turno),
		QuantidadeAlunos: len(reservas),
	})
	if err != nil {
		return nil, err
	}

	alocacaoMotoristas, err := s.motoristaAlocador.Alocar(ctx, motoristas.AlocarMotoristasInput{
		MunicipioTrabalhoID: input.MunicipioDestinoID,
		DataViagem:          input.DataViagem,
		Turno:               motoristas.Turno(input.Turno),
		Quantidade:          len(alocacaoVeiculos.Veiculos),
	})
	if err != nil {
		return nil, err
	}

	ciclosInput := montarCiclosIdaComReservasInput(input, alocacaoVeiculos.Veiculos, alocacaoMotoristas, reservas)
	planejamento, err := s.cicloStore.CreatePlanejamentoIda(ctx, ciclosInput, partida)
	if err != nil {
		return nil, err
	}

	planejamento.Sentido = SentidoIda
	planejamento.QuantidadeReservas = len(reservas)
	planejamento.CapacidadeTotal = alocacaoVeiculos.CapacidadeTotal

	return planejamento, nil
}

func (s *planejamentoService) planejarVolta(ctx context.Context, input PlanejamentoViagensInput, partida time.Time) (*PlanejamentoViagens, error) {
	filtro := PlanejamentoReservasFiltro{
		DataViagem:         input.DataViagem,
		Turno:              input.Turno,
		MunicipioDestinoID: input.MunicipioDestinoID,
		RotaInternaID:      input.RotaInternaID,
		Sentido:            SentidoVolta,
	}

	ciclos, err := s.cicloStore.ListCiclosParaPlanejamentoVolta(ctx, filtro)
	if err != nil {
		return nil, err
	}
	if len(ciclos) == 0 {
		return nil, ErrSemDemandaPlanejamento
	}

	reservas, err := s.cicloStore.ListReservasElegiveisParaVolta(ctx, filtro)
	if err != nil {
		return nil, err
	}

	inputs, capacidadeTotal, err := montarCiclosVoltaComReservasInput(ciclos, reservas)
	if err != nil {
		return nil, err
	}
	planejamento, err := s.cicloStore.CreatePlanejamentoVolta(ctx, inputs, partida)
	if err != nil {
		return nil, err
	}

	planejamento.Sentido = SentidoVolta
	planejamento.QuantidadeReservas = len(reservas)
	planejamento.CapacidadeTotal = capacidadeTotal
	return planejamento, nil
}

func validatePlanejamentoInput(input PlanejamentoViagensInput) error {
	if input.DataViagem.IsZero() {
		return errors.New("Informe a data da viagem.")
	}
	if !isOperationalTurnoViagem(input.Turno) {
		return errors.New("turno must be MT, VT or NT")
	}
	if input.MunicipioDestinoID <= 0 {
		return errors.New("municipio_destino_id is required")
	}
	if input.RotaInternaID <= 0 {
		return errors.New("Selecione a rota interna.")
	}
	if input.Sentido != SentidoIda && input.Sentido != SentidoVolta {
		return errors.New("Selecione o sentido: ida ou volta.")
	}

	return nil
}

func calcularExpiresAtPlanejamento(dataViagem time.Time, location *time.Location) time.Time {
	dataLocal := time.Date(dataViagem.Year(), dataViagem.Month(), dataViagem.Day(), 0, 0, 0, 0, location)
	return dataLocal.AddDate(0, 3, 0)
}

func montarPartidaPlanejamento(dataViagem time.Time, horario *HorarioTurnoViagem, sentido SentidoViagem, location *time.Location) time.Time {
	horarioPartida := horario.HorarioIda
	if sentido == SentidoVolta {
		horarioPartida = horario.HorarioVolta
	}
	return combinarDataHorario(dataViagem, horarioPartida, location)
}

func combinarDataHorario(data time.Time, horario time.Duration, location *time.Location) time.Time {
	hours := int(horario / time.Hour)
	horario -= time.Duration(hours) * time.Hour
	minutes := int(horario / time.Minute)
	horario -= time.Duration(minutes) * time.Minute
	seconds := int(horario / time.Second)
	nanoseconds := int(horario - time.Duration(seconds)*time.Second)

	return time.Date(data.Year(), data.Month(), data.Day(), hours, minutes, seconds, nanoseconds, location)
}

func montarCiclosIdaComReservasInput(input PlanejamentoViagensInput, veiculosAlocados []veiculos.Veiculo, motoristasAlocados []motoristas.Motorista, reservas []PlanejamentoReserva) []CicloIdaComReservasInput {
	result := make([]CicloIdaComReservasInput, 0, len(veiculosAlocados))
	reservasPorVeiculo := distribuirReservasPorDestinoECapacidade(reservas, veiculosAlocados)

	for i, veiculo := range veiculosAlocados {
		result = append(result, CicloIdaComReservasInput{
			Ciclo: CicloViagemInput{
				DataViagem:         input.DataViagem,
				Turno:              input.Turno,
				MunicipioDestinoID: input.MunicipioDestinoID,
				RotaInternaID:      input.RotaInternaID,
				VeiculoID:          veiculo.ID,
				MotoristaID:        motoristasAlocados[i].ID,
				ExpiresAt:          input.ExpiresAt,
			},
			ReservaIDs: reservasPorVeiculo[i],
		})
	}

	return result
}

func montarCiclosVoltaComReservasInput(ciclos []CicloPlanejamentoVolta, reservas []PlanejamentoReserva) ([]CicloVoltaComReservasInput, int, error) {
	veiculosAlocados := make([]veiculos.Veiculo, 0, len(ciclos))
	capacidadeTotal := 0
	for _, ciclo := range ciclos {
		veiculosAlocados = append(veiculosAlocados, veiculos.Veiculo{
			ID:         ciclo.Ciclo.VeiculoID,
			Capacidade: int16(ciclo.Capacidade),
		})
		capacidadeTotal += ciclo.Capacidade
	}

	reservasPorVeiculo := distribuirReservasPorDestinoECapacidade(reservas, veiculosAlocados)
	quantidadeAlocada := 0
	inputs := make([]CicloVoltaComReservasInput, 0, len(ciclos))
	for i, ciclo := range ciclos {
		quantidadeAlocada += len(reservasPorVeiculo[i])
		inputs = append(inputs, CicloVoltaComReservasInput{
			Ciclo:      ciclo.Ciclo,
			ReservaIDs: reservasPorVeiculo[i],
		})
	}
	if quantidadeAlocada != len(reservas) {
		return nil, 0, fmt.Errorf("%w: As reservas de volta excedem a capacidade dos veículos da ida.", brerror.ErrInvalidInput)
	}

	return inputs, capacidadeTotal, nil
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

func isOperationalTurnoViagem(turno TurnoViagem) bool {
	return turno == TurnoMatutino || turno == TurnoVespertino || turno == TurnoNoturno
}
