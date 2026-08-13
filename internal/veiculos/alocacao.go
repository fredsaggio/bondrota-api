package veiculos

import (
	"fmt"
	"math"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

type PlanoCategoriaVeiculo struct {
	Categoria  CategoriaVeiculo
	Capacidade int16
	Quantidade int
}

func PlanejarCategoriasPorQuantidade(qtdAlunos int) ([]PlanoCategoriaVeiculo, error) {
	if qtdAlunos <= 0 {
		return nil, fmt.Errorf("%w: A quantidade de alunos deve ser maior que zero.", brerror.ErrInvalidInput)
	}

	categorias := []struct {
		categoria  CategoriaVeiculo
		capacidade int16
	}{
		{CategoriaExecutivo, CapacidadeExecutivo},
		{CategoriaEscolar, CapacidadeEscolar},
		{CategoriaCarroSeteLugares, CapacidadeCarroSeteLugares},
	}

	maxExecutivo := qtdAlunos/int16ToInt(CapacidadeExecutivo) + 2
	maxEscolar := qtdAlunos/int16ToInt(CapacidadeEscolar) + 2
	maxCarro := qtdAlunos/int16ToInt(CapacidadeCarroSeteLugares) + 2

	best := []int{0, 0, 0}
	bestVehicles := math.MaxInt
	bestCapacity := math.MaxInt

	for executivo := 0; executivo <= maxExecutivo; executivo++ {
		for escolar := 0; escolar <= maxEscolar; escolar++ {
			for carro := 0; carro <= maxCarro; carro++ {
				counts := []int{executivo, escolar, carro}
				vehicles := executivo + escolar + carro
				if vehicles == 0 {
					continue
				}

				capacity := executivo*int16ToInt(CapacidadeExecutivo) +
					escolar*int16ToInt(CapacidadeEscolar) +
					carro*int16ToInt(CapacidadeCarroSeteLugares)
				if capacity < qtdAlunos {
					continue
				}

				if isBetterPlano(qtdAlunos, counts, vehicles, capacity, best, bestVehicles, bestCapacity) {
					best = counts
					bestVehicles = vehicles
					bestCapacity = capacity
				}
			}
		}
	}

	plano := make([]PlanoCategoriaVeiculo, 0, len(categorias))
	for i, count := range best {
		if count == 0 {
			continue
		}
		plano = append(plano, PlanoCategoriaVeiculo{
			Categoria:  categorias[i].categoria,
			Capacidade: categorias[i].capacidade,
			Quantidade: count,
		})
	}

	return plano, nil
}

func isBetterPlano(qtdAlunos int, candidate []int, candidateVehicles, candidateCapacity int, best []int, bestVehicles, bestCapacity int) bool {
	if candidateVehicles != bestVehicles {
		return candidateVehicles < bestVehicles
	}
	if qtdAlunos > int16ToInt(CapacidadeEscolar) {
		candidateHasExecutivo := candidate[0] > 0
		bestHasExecutivo := best[0] > 0
		if candidateHasExecutivo != bestHasExecutivo {
			return candidateHasExecutivo
		}
	}
	if candidateCapacity != bestCapacity {
		return candidateCapacity < bestCapacity
	}
	if candidate[0] != best[0] {
		return candidate[0] > best[0]
	}
	if candidate[1] != best[1] {
		return candidate[1] > best[1]
	}
	return candidateCapacity < bestCapacity
}

func int16ToInt(value int16) int {
	return int(value)
}
