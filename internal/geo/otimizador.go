package geo

import (
	"errors"
	"fmt"
	"math"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

const defaultLimiteHeldKarp = 12

type PontoRoteirizacao struct {
	ID         int64
	Nome       string
	Coordenada Coordenada
}

type OtimizacaoRotaMatrizInput struct {
	Destinos            []PontoRoteirizacao
	CustosEntreDestinos [][]float64
	CustosOrigem        []float64
	CustosDestinoFinal  []float64
}

type OtimizacaoRotaResult struct {
	Destinos      []PontoRoteirizacao
	CustoEstimado float64
}

type OtimizadorRota struct {
	limiteHeldKarp int
}

func NewOtimizadorRota() *OtimizadorRota {
	return &OtimizadorRota{limiteHeldKarp: defaultLimiteHeldKarp}
}

func (o *OtimizadorRota) OrdenarDestinosPorMatriz(input OtimizacaoRotaMatrizInput) (*OtimizacaoRotaResult, error) {
	if o == nil {
		o = NewOtimizadorRota()
	}
	if err := validateOtimizacaoMatrizInput(input); err != nil {
		return nil, err
	}

	var (
		ordem []int
		custo float64
	)
	if len(input.Destinos) <= o.limiteHeldKarp {
		ordem, custo = ordenarHeldKarp(input)
	} else {
		ordem, custo = ordenarHeuristicaMatriz(input)
	}

	ordenados := make([]PontoRoteirizacao, len(ordem))
	for i, index := range ordem {
		ordenados[i] = input.Destinos[index]
	}

	return &OtimizacaoRotaResult{
		Destinos:      ordenados,
		CustoEstimado: custo,
	}, nil
}

func ordenarHeldKarp(input OtimizacaoRotaMatrizInput) ([]int, float64) {
	n := len(input.Destinos)
	states := 1 << n
	// dp[mask, destino] guarda o menor custo para visitar mask e terminar em destino.
	dp := make([]float64, states*n)
	parent := make([]int, states*n)
	for i := range dp {
		dp[i] = math.Inf(1)
		parent[i] = -1
	}

	for destino := 0; destino < n; destino++ {
		mask := 1 << destino
		dp[mask*n+destino] = custoOrigem(input.CustosOrigem, destino)
	}

	for mask := 1; mask < states; mask++ {
		for atual := 0; atual < n; atual++ {
			if mask&(1<<atual) == 0 {
				continue
			}

			maskAnterior := mask &^ (1 << atual)
			if maskAnterior == 0 {
				continue
			}

			stateIndex := mask*n + atual
			for anterior := 0; anterior < n; anterior++ {
				if maskAnterior&(1<<anterior) == 0 {
					continue
				}

				candidate := dp[maskAnterior*n+anterior] + input.CustosEntreDestinos[anterior][atual]
				if candidate < dp[stateIndex] {
					dp[stateIndex] = candidate
					parent[stateIndex] = anterior
				}
			}
		}
	}

	fullMask := states - 1
	ultimo := 0
	melhorCusto := math.Inf(1)
	for destino := 0; destino < n; destino++ {
		candidate := dp[fullMask*n+destino] + custoDestinoFinal(input.CustosDestinoFinal, destino)
		if candidate < melhorCusto {
			melhorCusto = candidate
			ultimo = destino
		}
	}

	ordem := make([]int, n)
	mask := fullMask
	for pos := n - 1; pos >= 0; pos-- {
		ordem[pos] = ultimo
		anterior := parent[mask*n+ultimo]
		mask &^= 1 << ultimo
		ultimo = anterior
	}

	return ordem, melhorCusto
}

func ordenarHeuristicaMatriz(input OtimizacaoRotaMatrizInput) ([]int, float64) {
	n := len(input.Destinos)
	var melhor []int
	melhorCusto := math.Inf(1)

	for inicio := 0; inicio < n; inicio++ {
		ordem := ordenarVizinhoMaisProximoMatriz(input.CustosEntreDestinos, inicio)
		ordem = melhorar2OptMatriz(input, ordem)
		custo := custoTotalMatriz(input, ordem)
		if custo < melhorCusto {
			melhor = ordem
			melhorCusto = custo
		}
	}

	return melhor, melhorCusto
}

func ordenarVizinhoMaisProximoMatriz(custos [][]float64, inicio int) []int {
	n := len(custos)
	visitado := make([]bool, n)
	ordem := make([]int, 0, n)
	atual := inicio
	visitado[atual] = true
	ordem = append(ordem, atual)

	for len(ordem) < n {
		proximo := -1
		melhorCusto := math.Inf(1)
		for candidato := 0; candidato < n; candidato++ {
			if visitado[candidato] || custos[atual][candidato] >= melhorCusto {
				continue
			}
			proximo = candidato
			melhorCusto = custos[atual][candidato]
		}

		visitado[proximo] = true
		ordem = append(ordem, proximo)
		atual = proximo
	}

	return ordem
}

func melhorar2OptMatriz(input OtimizacaoRotaMatrizInput, ordem []int) []int {
	melhor := append([]int(nil), ordem...)
	melhorCusto := custoTotalMatriz(input, melhor)
	improved := true

	for improved {
		improved = false
		for i := 0; i < len(melhor)-1; i++ {
			for j := i + 1; j < len(melhor); j++ {
				candidate := append([]int(nil), melhor...)
				reverseIndices(candidate[i : j+1])
				custo := custoTotalMatriz(input, candidate)
				if custo < melhorCusto {
					melhor = candidate
					melhorCusto = custo
					improved = true
				}
			}
		}
	}

	return melhor
}

func custoTotalMatriz(input OtimizacaoRotaMatrizInput, ordem []int) float64 {
	total := custoOrigem(input.CustosOrigem, ordem[0])
	for i := 1; i < len(ordem); i++ {
		total += input.CustosEntreDestinos[ordem[i-1]][ordem[i]]
	}
	total += custoDestinoFinal(input.CustosDestinoFinal, ordem[len(ordem)-1])
	return total
}

func custoOrigem(custos []float64, destino int) float64 {
	if custos == nil {
		return 0
	}
	return custos[destino]
}

func custoDestinoFinal(custos []float64, origem int) float64 {
	if custos == nil {
		return 0
	}
	return custos[origem]
}

func validateOtimizacaoMatrizInput(input OtimizacaoRotaMatrizInput) error {
	if err := validatePontos(input.Destinos); err != nil {
		return err
	}

	n := len(input.Destinos)
	if len(input.CustosEntreDestinos) != n {
		return fmt.Errorf("%w: custos_entre_destinos must be a square matrix", brerror.ErrInvalidInput)
	}
	for _, row := range input.CustosEntreDestinos {
		if len(row) != n {
			return fmt.Errorf("%w: custos_entre_destinos must be a square matrix", brerror.ErrInvalidInput)
		}
		if err := validateCustos(row); err != nil {
			return err
		}
	}
	if input.CustosOrigem != nil && len(input.CustosOrigem) != n {
		return fmt.Errorf("%w: custos_origem must match destinos", brerror.ErrInvalidInput)
	}
	if input.CustosDestinoFinal != nil && len(input.CustosDestinoFinal) != n {
		return fmt.Errorf("%w: custos_destino_final must match destinos", brerror.ErrInvalidInput)
	}
	if err := validateCustos(input.CustosOrigem); err != nil {
		return err
	}
	return validateCustos(input.CustosDestinoFinal)
}

func validatePontos(destinos []PontoRoteirizacao) error {
	if len(destinos) == 0 {
		return fmt.Errorf("%w: destinos is required", brerror.ErrInvalidInput)
	}

	seen := make(map[int64]struct{}, len(destinos))
	for _, destino := range destinos {
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

func validateCustos(custos []float64) error {
	for _, custo := range custos {
		if custo < 0 || math.IsNaN(custo) || math.IsInf(custo, 0) {
			return fmt.Errorf("%w: route costs must be finite and non-negative", brerror.ErrInvalidInput)
		}
	}
	return nil
}

func validateCoordenada(coordenada Coordenada) error {
	if coordenada.Latitude == 0 && coordenada.Longitude == 0 {
		return errors.New("Marque a localização no mapa.")
	}
	if coordenada.Latitude < -90 || coordenada.Latitude > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if coordenada.Longitude < -180 || coordenada.Longitude > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

func reverseIndices(indices []int) {
	for i, j := 0, len(indices)-1; i < j; i, j = i+1, j-1 {
		indices[i], indices[j] = indices[j], indices[i]
	}
}
