package veiculos

import (
	"context"
	"fmt"
	"strings"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

type AlocacaoService struct {
	store AlocacaoVeiculoStore
}

func NewAlocacaoService(store AlocacaoVeiculoStore) *AlocacaoService {
	return &AlocacaoService{store: store}
}

func (s *AlocacaoService) Alocar(ctx context.Context, input AlocarVeiculosInput) (*AlocacaoVeiculos, error) {
	if err := validateAlocarVeiculosInput(input); err != nil {
		return nil, err
	}

	plano, err := PlanejarCategoriasPorQuantidade(input.QuantidadeAlunos)
	if err != nil {
		return nil, err
	}

	disponiveis, err := s.store.ListDisponiveisParaAlocacao(ctx, VeiculosDisponiveisFiltro{
		DataViagem: input.DataViagem,
		Turno:      input.Turno,
		Categorias: categoriasParaBusca(plano),
	})
	if err != nil {
		return nil, err
	}

	selecionados, err := selecionarVeiculosDoPlano(plano, disponiveis)
	if err != nil {
		return nil, err
	}

	return &AlocacaoVeiculos{
		Plano:           plano,
		Veiculos:        selecionados,
		CapacidadeTotal: calcularCapacidadeTotal(selecionados),
	}, nil
}

func validateAlocarVeiculosInput(input AlocarVeiculosInput) error {
	if input.DataViagem.IsZero() {
		return fmt.Errorf("%w: Informe a data da viagem.", brerror.ErrInvalidInput)
	}
	if !isTurnoOperacional(input.Turno) {
		return fmt.Errorf("%w: Selecione um turno válido: matutino, vespertino ou noturno.", brerror.ErrInvalidInput)
	}
	if input.QuantidadeAlunos <= 0 {
		return fmt.Errorf("%w: A quantidade de alunos deve ser maior que zero.", brerror.ErrInvalidInput)
	}
	return nil
}

func isTurnoOperacional(turno string) bool {
	switch strings.TrimSpace(turno) {
	case "MT", "VT", "NT":
		return true
	default:
		return false
	}
}

func categoriasParaBusca(plano []PlanoCategoriaVeiculo) []CategoriaVeiculo {
	seen := make(map[CategoriaVeiculo]bool)
	categorias := make([]CategoriaVeiculo, 0, 3)
	for _, item := range plano {
		for _, categoria := range categoriasComFallback(item.Categoria) {
			if seen[categoria] {
				continue
			}
			seen[categoria] = true
			categorias = append(categorias, categoria)
		}
	}
	return categorias
}

func selecionarVeiculosDoPlano(plano []PlanoCategoriaVeiculo, disponiveis []Veiculo) ([]Veiculo, error) {
	selecionados := make([]Veiculo, 0)
	usados := make(map[int64]bool)

	for _, item := range plano {
		quantidadeSelecionada := 0
		for _, categoria := range categoriasComFallback(item.Categoria) {
			for _, veiculo := range disponiveis {
				if usados[veiculo.ID] || veiculo.Categoria != categoria {
					continue
				}

				selecionados = append(selecionados, veiculo)
				usados[veiculo.ID] = true
				quantidadeSelecionada++

				if quantidadeSelecionada == item.Quantidade {
					break
				}
			}
			if quantidadeSelecionada == item.Quantidade {
				break
			}
		}

		if quantidadeSelecionada < item.Quantidade {
			// A categoria fica so no log: para quem usa o painel, o que importa
			// e que faltou veiculo, nao o nome interno da categoria.
			return nil, fmt.Errorf("%w: Não há veículos suficientes disponíveis.", brerror.ErrNotFound)
		}
	}

	return selecionados, nil
}

func categoriasComFallback(categoria CategoriaVeiculo) []CategoriaVeiculo {
	switch categoria {
	case CategoriaCarroSeteLugares:
		return []CategoriaVeiculo{CategoriaCarroSeteLugares, CategoriaEscolar, CategoriaExecutivo}
	case CategoriaEscolar:
		return []CategoriaVeiculo{CategoriaEscolar, CategoriaExecutivo}
	case CategoriaExecutivo:
		return []CategoriaVeiculo{CategoriaExecutivo}
	default:
		return []CategoriaVeiculo{categoria}
	}
}

func calcularCapacidadeTotal(veiculos []Veiculo) int {
	total := 0
	for _, veiculo := range veiculos {
		total += int(veiculo.Capacidade)
	}
	return total
}
