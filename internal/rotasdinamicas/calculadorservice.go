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
		return nil, invalidInput("viagem_id is required")
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

	input, coordenadas, err := s.montarInputRota(dados)
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

func (s *calculadorRotaDinamicaService) montarInputRota(dados *DadosCalculoRota) (RotaDinamicaInput, []geo.Coordenada, error) {
	switch dados.Sentido {
	case sentidoIda:
		return s.montarInputIda(dados)
	case sentidoVolta:
		return s.montarInputVolta(dados)
	default:
		return RotaDinamicaInput{}, nil, invalidInput("sentido da viagem invalido")
	}
}

func (s *calculadorRotaDinamicaService) montarInputIda(dados *DadosCalculoRota) (RotaDinamicaInput, []geo.Coordenada, error) {
	origem := dados.Paradas[len(dados.Paradas)-1]

	result, err := s.otimizador.OrdenarDestinos(geo.OtimizacaoRotaInput{
		Origem:   coordenadaPonto(origem),
		Destinos: toPontosRoteirizacao(dados.Destinos),
	})
	if err != nil {
		return RotaDinamicaInput{}, nil, err
	}

	ordenados := result.Destinos
	destinoFinal := ordenados[len(ordenados)-1]
	coordenadas := make([]geo.Coordenada, 0, len(ordenados)+1)
	coordenadas = append(coordenadas, coordenadaPonto(origem))
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

func (s *calculadorRotaDinamicaService) montarInputVolta(dados *DadosCalculoRota) (RotaDinamicaInput, []geo.Coordenada, error) {
	destinoFinal := dados.Paradas[0]
	ordenados, err := s.ordenarVolta(dados.Destinos, coordenadaPonto(destinoFinal))
	if err != nil {
		return RotaDinamicaInput{}, nil, err
	}

	origem := ordenados[0]
	coordenadas := make([]geo.Coordenada, 0, len(ordenados)+1)
	for _, destino := range ordenados {
		coordenadas = append(coordenadas, destino.Coordenada)
	}
	coordenadas = append(coordenadas, coordenadaPonto(destinoFinal))

	return RotaDinamicaInput{
		ViagemID:     dados.ViagemID,
		Origem:       pontoRotaFromRoteirizacao(origem),
		DestinoFinal: pontoRotaFromParada(destinoFinal),
		ExpiresAt:    dados.ExpiresAt,
		Destinos:     toRotaDinamicaDestinoInputs(ordenados),
	}, coordenadas, nil
}

func (s *calculadorRotaDinamicaService) ordenarVolta(destinos []DestinoCalculoRota, destinoFinal geo.Coordenada) ([]geo.PontoRoteirizacao, error) {
	pontos := toPontosRoteirizacao(destinos)
	if len(pontos) == 1 {
		return pontos, nil
	}

	var best []geo.PontoRoteirizacao
	bestDist := int(^uint(0) >> 1)

	for i, origem := range pontos {
		restantes := make([]geo.PontoRoteirizacao, 0, len(pontos)-1)
		restantes = append(restantes, pontos[:i]...)
		restantes = append(restantes, pontos[i+1:]...)

		result, err := s.otimizador.OrdenarDestinos(geo.OtimizacaoRotaInput{
			Origem:           origem.Coordenada,
			Destinos:         restantes,
			DestinoFinal:     destinoFinal,
			UsarDestinoFinal: true,
		})
		if err != nil {
			return nil, err
		}

		ordenados := append([]geo.PontoRoteirizacao{origem}, result.Destinos...)
		if result.DistanciaEstimadaMetros < bestDist {
			bestDist = result.DistanciaEstimadaMetros
			best = ordenados
		}
	}

	return best, nil
}

func validateDadosCalculo(dados *DadosCalculoRota) error {
	if dados == nil {
		return invalidInput("dados de calculo is required")
	}
	if dados.ViagemID <= 0 {
		return invalidInput("viagem_id is required")
	}
	if dados.ExpiresAt.IsZero() {
		return invalidInput("expires_at is required")
	}
	if len(dados.Paradas) == 0 {
		return invalidInput("rota interna precisa ter paradas")
	}
	if len(dados.Destinos) == 0 {
		return invalidInput("viagem precisa ter reservas com destinos")
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
