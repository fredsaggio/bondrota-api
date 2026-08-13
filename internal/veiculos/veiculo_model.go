package veiculos

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

var ErrNotFound = errors.New("Veículo não encontrado.")

type StatusVeiculo string
type CategoriaVeiculo string

const (
	StatusAtivo      StatusVeiculo = "ativo"
	StatusInativo    StatusVeiculo = "inativo"
	StatusManutencao StatusVeiculo = "manutencao"

	CategoriaExecutivo        CategoriaVeiculo = "executivo"
	CategoriaEscolar          CategoriaVeiculo = "escolar"
	CategoriaCarroSeteLugares CategoriaVeiculo = "carro_7_lugares"

	CapacidadeExecutivo        int16 = 46
	CapacidadeEscolar          int16 = 24
	CapacidadeCarroSeteLugares int16 = 7
)

type Veiculo struct {
	ID             int64            `json:"id"`
	Placa          string           `json:"placa"`
	Modelo         string           `json:"modelo"`
	Categoria      CategoriaVeiculo `json:"categoria"`
	Capacidade     int16            `json:"capacidade"`
	Status         StatusVeiculo    `json:"status"`
	ArCondicionado bool             `json:"ar_condicionado"`
	Banheiro       bool             `json:"banheiro"`
	Persiana       bool             `json:"persiana"`
	LuzLeitura     bool             `json:"luz_leitura"`
	Tomada         bool             `json:"tomada"`
}

type VeiculoInput struct {
	Placa          string           `json:"placa"`
	Modelo         string           `json:"modelo"`
	Categoria      CategoriaVeiculo `json:"categoria"`
	Capacidade     int16            `json:"capacidade"`
	Status         StatusVeiculo    `json:"status"`
	ArCondicionado bool             `json:"ar_condicionado"`
	Banheiro       bool             `json:"banheiro"`
	Persiana       bool             `json:"persiana"`
	LuzLeitura     bool             `json:"luz_leitura"`
	Tomada         bool             `json:"tomada"`
}

type VeiculosDisponiveisFiltro struct {
	DataViagem time.Time
	Turno      string
	Categorias []CategoriaVeiculo
}

type AlocarVeiculosInput struct {
	DataViagem       time.Time
	Turno            string
	QuantidadeAlunos int
}

type AlocacaoVeiculos struct {
	Plano           []PlanoCategoriaVeiculo
	Veiculos        []Veiculo
	CapacidadeTotal int
}

type VeiculoStore interface {
	Create(ctx context.Context, input VeiculoInput) (*Veiculo, error)
	GetByID(ctx context.Context, id int64) (*Veiculo, error)
	List(ctx context.Context) ([]Veiculo, error)
	Update(ctx context.Context, id int64, updateFunc func(*Veiculo) (bool, error)) (*Veiculo, error)
	Delete(ctx context.Context, id int64) error
}

type AlocacaoVeiculoStore interface {
	ListDisponiveisParaAlocacao(ctx context.Context, filtro VeiculosDisponiveisFiltro) ([]Veiculo, error)
}

func CapacidadeDaCategoria(categoria CategoriaVeiculo) (int16, bool) {
	switch categoria {
	case CategoriaExecutivo:
		return CapacidadeExecutivo, true
	case CategoriaEscolar:
		return CapacidadeEscolar, true
	case CategoriaCarroSeteLugares:
		return CapacidadeCarroSeteLugares, true
	default:
		return 0, false
	}
}

func ValidateCategoriaCapacidade(categoria CategoriaVeiculo, capacidade int16) error {
	expected, ok := CapacidadeDaCategoria(categoria)
	if !ok {
		return fmt.Errorf("%w: Selecione uma categoria válida.", brerror.ErrInvalidInput)
	}
	if capacidade != expected {
		return fmt.Errorf("%w: A capacidade deve ser de %d lugares para esta categoria.", brerror.ErrInvalidInput, expected)
	}
	return nil
}
