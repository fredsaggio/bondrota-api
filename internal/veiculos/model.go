package veiculos

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("vehicle not found")

type StatusVeiculo string

const (
	StatusAtivo      StatusVeiculo = "ativo"
	StatusInativo    StatusVeiculo = "inativo"
	StatusManutencao StatusVeiculo = "manutencao"
)

type Veiculo struct {
	ID             int64         `json:"id"`
	Placa          string        `json:"placa"`
	Modelo         string        `json:"modelo"`
	Capacidade     int16         `json:"capacidade"`
	CidadeBase     string        `json:"cidade_base"`
	Status         StatusVeiculo `json:"status"`
	ArCondicionado bool          `json:"ar_condicionado"`
	Banheiro       bool          `json:"banheiro"`
	Persiana       bool          `json:"persiana"`
	LuzLeitura     bool          `json:"luz_leitura"`
	Tomada         bool          `json:"tomada"`
}

type VeiculoInput struct {
	Placa          string        `json:"placa"`
	Modelo         string        `json:"modelo"`
	Capacidade     int16         `json:"capacidade"`
	CidadeBase     string        `json:"cidade_base"`
	Status         StatusVeiculo `json:"status"`
	ArCondicionado bool          `json:"ar_condicionado"`
	Banheiro       bool          `json:"banheiro"`
	Persiana       bool          `json:"persiana"`
	LuzLeitura     bool          `json:"luz_leitura"`
	Tomada         bool          `json:"tomada"`
}

type VeiculoStore interface {
	Create(ctx context.Context, input VeiculoInput) (*Veiculo, error)
	GetByID(ctx context.Context, id int64) (*Veiculo, error)
	List(ctx context.Context) ([]Veiculo, error)
	Update(ctx context.Context, id int64, updateFunc func(*Veiculo) (bool, error)) (*Veiculo, error)
	Delete(ctx context.Context, id int64) (*Veiculo, error)
}
