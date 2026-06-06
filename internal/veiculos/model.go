package veiculos

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

var ErrNotFound = errors.New("vehicle not found")

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
	CidadeBase     string           `json:"cidade_base"`
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
	CidadeBase     string           `json:"cidade_base"`
	Status         StatusVeiculo    `json:"status"`
	ArCondicionado bool             `json:"ar_condicionado"`
	Banheiro       bool             `json:"banheiro"`
	Persiana       bool             `json:"persiana"`
	LuzLeitura     bool             `json:"luz_leitura"`
	Tomada         bool             `json:"tomada"`
}

type VeiculoStore interface {
	Create(ctx context.Context, input VeiculoInput) (*Veiculo, error)
	GetByID(ctx context.Context, id int64) (*Veiculo, error)
	List(ctx context.Context) ([]Veiculo, error)
	Update(ctx context.Context, id int64, updateFunc func(*Veiculo) (bool, error)) (*Veiculo, error)
	Delete(ctx context.Context, id int64) error
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
		return fmt.Errorf("%w: categoria must be executivo, escolar or carro_7_lugares", brerror.ErrInvalidInput)
	}
	if capacidade != expected {
		return fmt.Errorf("%w: capacidade for categoria %s must be %d", brerror.ErrInvalidInput, categoria, expected)
	}
	return nil
}
