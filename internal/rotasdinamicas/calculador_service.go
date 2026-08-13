package rotasdinamicas

import (
	"context"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/geo"
)

const (
	sentidoIda   = "ida"
	sentidoVolta = "volta"
)

type calculadorRotaDinamicaService struct {
	store      CalculadorRotaDinamicaStore
	rotaSvc    RotaDinamicaService
	roteador   geo.Roteador
	otimizador *geo.OtimizadorRota
}

func NewCalculadorRotaDinamicaService(
	store CalculadorRotaDinamicaStore,
	rotaSvc RotaDinamicaService,
	roteador geo.Roteador,
	otimizador *geo.OtimizadorRota,
) CalculadorRotaDinamicaService {
	if otimizador == nil {
		otimizador = geo.NewOtimizadorRota()
	}

	return &calculadorRotaDinamicaService{
		store:      store,
		rotaSvc:    rotaSvc,
		roteador:   roteador,
		otimizador: otimizador,
	}
}

func (s *calculadorRotaDinamicaService) Calcular(ctx context.Context, viagemID int64) (*RotaDinamicaComDestinos, error) {
	if viagemID <= 0 {
		return nil, invalidInput("Selecione a viagem.")
	}
	if s.roteador == nil {
		return nil, fmt.Errorf("%w: roteador is required", brerror.ErrInvalidInput)
	}

	dados, err := s.store.GetDadosCalculo(ctx, viagemID)
	if err != nil {
		return nil, err
	}
	if err := validateDadosCalculo(dados); err != nil {
		return nil, err
	}

	input, coordenadas, err := s.montarInputRota(ctx, dados)
	if err != nil {
		return nil, err
	}

	rotaCalculada, err := s.roteador.CalcularRota(ctx, coordenadas)
	if err != nil {
		return nil, err
	}

	input.DistanciaMetros = rotaCalculada.DistanciaMetros
	input.DuracaoSegundos = rotaCalculada.DuracaoSegundos
	input.Geometry = rotaCalculada.Geometry

	return s.rotaSvc.Create(ctx, input)
}

func (s *calculadorRotaDinamicaService) montarInputRota(ctx context.Context, dados *DadosCalculoRota) (RotaDinamicaInput, []geo.Coordenada, error) {
	switch dados.Sentido {
	case sentidoIda:
		return s.montarInputIda(ctx, dados)
	case sentidoVolta:
		return s.montarInputVolta(ctx, dados)
	default:
		return RotaDinamicaInput{}, nil, invalidInput("Selecione o sentido: ida ou volta.")
	}
}

func (s *calculadorRotaDinamicaService) montarInputIda(ctx context.Context, dados *DadosCalculoRota) (RotaDinamicaInput, []geo.Coordenada, error) {
	origem := dados.Paradas[0]
	ultimaParada := dados.Paradas[len(dados.Paradas)-1]
	pontos := toPontosRoteirizacao(dados.Destinos)
	coordenadasOtimizacao := make([]geo.Coordenada, 0, len(pontos)+1)
	coordenadasOtimizacao = append(coordenadasOtimizacao, coordenadaPonto(ultimaParada))
	for _, ponto := range pontos {
		coordenadasOtimizacao = append(coordenadasOtimizacao, ponto.Coordenada)
	}

	matriz, err := s.roteador.CalcularMatriz(ctx, coordenadasOtimizacao)
	if err != nil {
		return RotaDinamicaInput{}, nil, err
	}

	result, err := s.otimizador.OrdenarDestinosPorMatriz(geo.OtimizacaoRotaMatrizInput{
		Destinos:            pontos,
		CustosEntreDestinos: recortarCustosDestinos(matriz.DuracoesSegundos, 1, len(pontos)),
		CustosOrigem:        custosOrigem(matriz.DuracoesSegundos, 0, 1, len(pontos)),
	})
	if err != nil {
		return RotaDinamicaInput{}, nil, err
	}

	ordenados := result.Destinos
	destinoFinal := ordenados[len(ordenados)-1]
	coordenadas := make([]geo.Coordenada, 0, len(dados.Paradas)+len(ordenados))
	coordenadas = appendCoordenadasParadas(coordenadas, dados.Paradas, false)
	for _, destino := range ordenados {
		coordenadas = append(coordenadas, destino.Coordenada)
	}

	return RotaDinamicaInput{
		ViagemID:     dados.ViagemID,
		Origem:       pontoRotaFromParada(origem),
		DestinoFinal: pontoRotaFromRoteirizacao(destinoFinal),
		ExpiresAt:    dados.ExpiresAt,
		Destinos:     toRotaDinamicaDestinoInputs(ordenados),
	}, coordenadas, nil
}

func (s *calculadorRotaDinamicaService) montarInputVolta(ctx context.Context, dados *DadosCalculoRota) (RotaDinamicaInput, []geo.Coordenada, error) {
	destinoFinal := dados.Paradas[0]
	entradaRotaInterna := dados.Paradas[len(dados.Paradas)-1]
	pontos := toPontosRoteirizacao(dados.Destinos)
	coordenadasOtimizacao := make([]geo.Coordenada, 0, len(pontos)+1)
	for _, ponto := range pontos {
		coordenadasOtimizacao = append(coordenadasOtimizacao, ponto.Coordenada)
	}
	coordenadasOtimizacao = append(coordenadasOtimizacao, coordenadaPonto(entradaRotaInterna))

	matriz, err := s.roteador.CalcularMatriz(ctx, coordenadasOtimizacao)
	if err != nil {
		return RotaDinamicaInput{}, nil, err
	}

	result, err := s.otimizador.OrdenarDestinosPorMatriz(geo.OtimizacaoRotaMatrizInput{
		Destinos:            pontos,
		CustosEntreDestinos: recortarCustosDestinos(matriz.DuracoesSegundos, 0, len(pontos)),
		CustosDestinoFinal:  custosDestinoFinal(matriz.DuracoesSegundos, len(pontos), len(pontos)),
	})
	if err != nil {
		return RotaDinamicaInput{}, nil, err
	}

	ordenados := result.Destinos
	origem := ordenados[0]
	coordenadas := make([]geo.Coordenada, 0, len(ordenados)+len(dados.Paradas))
	for _, destino := range ordenados {
		coordenadas = append(coordenadas, destino.Coordenada)
	}
	coordenadas = appendCoordenadasParadas(coordenadas, dados.Paradas, true)

	return RotaDinamicaInput{
		ViagemID:     dados.ViagemID,
		Origem:       pontoRotaFromRoteirizacao(origem),
		DestinoFinal: pontoRotaFromParada(destinoFinal),
		ExpiresAt:    dados.ExpiresAt,
		Destinos:     toRotaDinamicaDestinoInputs(ordenados),
	}, coordenadas, nil
}

func validateDadosCalculo(dados *DadosCalculoRota) error {
	if dados == nil {
		return invalidInput("Não foi possível calcular a rota agora. Tente novamente.")
	}
	if dados.ViagemID <= 0 {
		return invalidInput("Selecione a viagem.")
	}
	if dados.ExpiresAt.IsZero() {
		return invalidInput("expires_at is required")
	}
	if len(dados.Paradas) == 0 {
		return invalidInput("A rota interna precisa ter paradas.")
	}
	if len(dados.Destinos) == 0 {
		return invalidInput("Esta viagem não tem reservas com destino para calcular a rota.")
	}
	return nil
}

func toPontosRoteirizacao(destinos []DestinoCalculoRota) []geo.PontoRoteirizacao {
	pontos := make([]geo.PontoRoteirizacao, 0, len(destinos))
	for _, destino := range destinos {
		pontos = append(pontos, geo.PontoRoteirizacao{
			ID:   destino.ID,
			Nome: destino.Nome,
			Coordenada: geo.Coordenada{
				Latitude:  destino.Latitude,
				Longitude: destino.Longitude,
			},
		})
	}
	return pontos
}

func toRotaDinamicaDestinoInputs(destinos []geo.PontoRoteirizacao) []RotaDinamicaDestinoInput {
	inputs := make([]RotaDinamicaDestinoInput, 0, len(destinos))
	for i, destino := range destinos {
		inputs = append(inputs, RotaDinamicaDestinoInput{
			DestinoID: destino.ID,
			Ordem:     i + 1,
		})
	}
	return inputs
}

func pontoRotaFromParada(parada PontoCalculoRota) PontoRota {
	return PontoRota{
		Nome:      parada.Nome,
		Latitude:  parada.Latitude,
		Longitude: parada.Longitude,
	}
}

func pontoRotaFromRoteirizacao(ponto geo.PontoRoteirizacao) PontoRota {
	return PontoRota{
		Nome:      ponto.Nome,
		Latitude:  ponto.Coordenada.Latitude,
		Longitude: ponto.Coordenada.Longitude,
	}
}

func coordenadaPonto(ponto PontoCalculoRota) geo.Coordenada {
	return geo.Coordenada{
		Latitude:  ponto.Latitude,
		Longitude: ponto.Longitude,
	}
}

func appendCoordenadasParadas(coordenadas []geo.Coordenada, paradas []PontoCalculoRota, reverse bool) []geo.Coordenada {
	if reverse {
		for i := len(paradas) - 1; i >= 0; i-- {
			coordenadas = append(coordenadas, coordenadaPonto(paradas[i]))
		}
		return coordenadas
	}

	for _, parada := range paradas {
		coordenadas = append(coordenadas, coordenadaPonto(parada))
	}
	return coordenadas
}

func recortarCustosDestinos(matriz [][]float64, inicio, quantidade int) [][]float64 {
	custos := make([][]float64, quantidade)
	for i := 0; i < quantidade; i++ {
		custos[i] = append([]float64(nil), matriz[inicio+i][inicio:inicio+quantidade]...)
	}
	return custos
}

func custosOrigem(matriz [][]float64, origem, inicioDestinos, quantidade int) []float64 {
	return append([]float64(nil), matriz[origem][inicioDestinos:inicioDestinos+quantidade]...)
}

func custosDestinoFinal(matriz [][]float64, destinoFinal, quantidade int) []float64 {
	custos := make([]float64, quantidade)
	for i := 0; i < quantidade; i++ {
		custos[i] = matriz[i][destinoFinal]
	}
	return custos
}
