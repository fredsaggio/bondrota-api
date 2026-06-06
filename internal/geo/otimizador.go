package geo

import (
	"errors"
	"fmt"
	"math"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

const defaultLimiteForcaBruta = 8

type PontoRoteirizacao struct {
	ID         int64
	Nome       string
	Coordenada Coordenada
}

type OtimizacaoRotaInput struct {
	Origem           Coordenada
	Destinos         []PontoRoteirizacao
	DestinoFinal     Coordenada
	UsarDestinoFinal bool
}

type OtimizacaoRotaResult struct {
	Destinos                []PontoRoteirizacao
	DistanciaEstimadaMetros int
}

type OtimizadorRota struct {
	limiteForcaBruta int
}

func NewOtimizadorRota() *OtimizadorRota {
	return &OtimizadorRota{limiteForcaBruta: defaultLimiteForcaBruta}
}

func (o *OtimizadorRota) OrdenarDestinos(input OtimizacaoRotaInput) (*OtimizacaoRotaResult, error) {
	if o == nil {
		o = NewOtimizadorRota()
	}
	if err := validateOtimizacaoInput(input); err != nil {
		return nil, err
	}

	destinos := clonePontos(input.Destinos)
	if len(destinos) <= 1 {
		return &OtimizacaoRotaResult{
			Destinos:                destinos,
			DistanciaEstimadaMetros: int(math.Round(distanciaTotal(input.Origem, destinos, input.DestinoFinal, input.UsarDestinoFinal))),
		}, nil
	}

	var ordenados []PontoRoteirizacao
	if len(destinos) <= o.limiteForcaBruta {
		ordenados = ordenarForcaBruta(input.Origem, destinos, input.DestinoFinal, input.UsarDestinoFinal)
	} else {
		ordenados = ordenarVizinhoMaisProximo(input.Origem, destinos)
		ordenados = melhorar2Opt(input.Origem, ordenados, input.DestinoFinal, input.UsarDestinoFinal)
	}

	return &OtimizacaoRotaResult{
		Destinos:                ordenados,
		DistanciaEstimadaMetros: int(math.Round(distanciaTotal(input.Origem, ordenados, input.DestinoFinal, input.UsarDestinoFinal))),
	}, nil
}

func validateOtimizacaoInput(input OtimizacaoRotaInput) error {
	if err := validateCoordenada(input.Origem); err != nil {
		return fmt.Errorf("%w: origem %v", brerror.ErrInvalidInput, err)
	}
	if input.UsarDestinoFinal {
		if err := validateCoordenada(input.DestinoFinal); err != nil {
			return fmt.Errorf("%w: destino_final %v", brerror.ErrInvalidInput, err)
		}
	}
	if len(input.Destinos) == 0 {
		return fmt.Errorf("%w: destinos is required", brerror.ErrInvalidInput)
	}

	seen := make(map[int64]struct{}, len(input.Destinos))
	for _, destino := range input.Destinos {
		if destino.ID <= 0 {
			return fmt.Errorf("%w: destino id is required", brerror.ErrInvalidInput)
		}
		if _, ok := seen[destino.ID]; ok {
			return fmt.Errorf("%w: destino id duplicated", brerror.ErrInvalidInput)
		}
		seen[destino.ID] = struct{}{}

		if err := validateCoordenada(destino.Coordenada); err != nil {
			return fmt.Errorf("%w: destino %d %v", brerror.ErrInvalidInput, destino.ID, err)
		}
	}

	return nil
}

func validateCoordenada(coordenada Coordenada) error {
	if coordenada.Latitude == 0 && coordenada.Longitude == 0 {
		return errors.New("coordinates are required")
	}
	if coordenada.Latitude < -90 || coordenada.Latitude > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if coordenada.Longitude < -180 || coordenada.Longitude > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

func ordenarForcaBruta(origem Coordenada, destinos []PontoRoteirizacao, destinoFinal Coordenada, usarDestinoFinal bool) []PontoRoteirizacao {
	best := clonePontos(destinos)
	bestDist := math.Inf(1)
	perm := clonePontos(destinos)

	var walk func(int)
	walk = func(pos int) {
		if pos == len(perm) {
			dist := distanciaTotal(origem, perm, destinoFinal, usarDestinoFinal)
			if dist < bestDist {
				bestDist = dist
				best = clonePontos(perm)
			}
			return
		}

		for i := pos; i < len(perm); i++ {
			perm[pos], perm[i] = perm[i], perm[pos]
			walk(pos + 1)
			perm[pos], perm[i] = perm[i], perm[pos]
		}
	}
	walk(0)

	return best
}

func ordenarVizinhoMaisProximo(origem Coordenada, destinos []PontoRoteirizacao) []PontoRoteirizacao {
	pendentes := clonePontos(destinos)
	ordenados := make([]PontoRoteirizacao, 0, len(destinos))
	atual := origem

	for len(pendentes) > 0 {
		bestIndex := 0
		bestDist := distanciaMetros(atual, pendentes[0].Coordenada)
		for i := 1; i < len(pendentes); i++ {
			dist := distanciaMetros(atual, pendentes[i].Coordenada)
			if dist < bestDist {
				bestDist = dist
				bestIndex = i
			}
		}

		escolhido := pendentes[bestIndex]
		ordenados = append(ordenados, escolhido)
		atual = escolhido.Coordenada
		pendentes = append(pendentes[:bestIndex], pendentes[bestIndex+1:]...)
	}

	return ordenados
}

func melhorar2Opt(origem Coordenada, destinos []PontoRoteirizacao, destinoFinal Coordenada, usarDestinoFinal bool) []PontoRoteirizacao {
	best := clonePontos(destinos)
	bestDist := distanciaTotal(origem, best, destinoFinal, usarDestinoFinal)
	improved := true

	for improved {
		improved = false
		for i := 0; i < len(best)-1; i++ {
			for j := i + 1; j < len(best); j++ {
				candidate := clonePontos(best)
				reversePontos(candidate[i : j+1])
				dist := distanciaTotal(origem, candidate, destinoFinal, usarDestinoFinal)
				if dist < bestDist {
					best = candidate
					bestDist = dist
					improved = true
				}
			}
		}
	}

	return best
}

func distanciaTotal(origem Coordenada, destinos []PontoRoteirizacao, destinoFinal Coordenada, usarDestinoFinal bool) float64 {
	total := 0.0
	atual := origem
	for _, destino := range destinos {
		total += distanciaMetros(atual, destino.Coordenada)
		atual = destino.Coordenada
	}
	if usarDestinoFinal {
		total += distanciaMetros(atual, destinoFinal)
	}
	return total
}

func distanciaMetros(a, b Coordenada) float64 {
	const earthRadiusMeters = 6371000.0

	lat1 := grausParaRadianos(a.Latitude)
	lat2 := grausParaRadianos(b.Latitude)
	dLat := grausParaRadianos(b.Latitude - a.Latitude)
	dLon := grausParaRadianos(b.Longitude - a.Longitude)

	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

func grausParaRadianos(value float64) float64 {
	return value * math.Pi / 180
}

func reversePontos(pontos []PontoRoteirizacao) {
	for i, j := 0, len(pontos)-1; i < j; i, j = i+1, j-1 {
		pontos[i], pontos[j] = pontos[j], pontos[i]
	}
}

func clonePontos(pontos []PontoRoteirizacao) []PontoRoteirizacao {
	clone := make([]PontoRoteirizacao, len(pontos))
	copy(clone, pontos)
	return clone
}
