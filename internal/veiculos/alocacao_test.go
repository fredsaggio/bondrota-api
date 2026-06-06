package veiculos_test

import (
	"errors"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
)

func TestPlanejarCategoriasPorQuantidade(t *testing.T) {
	tests := []struct {
		name      string
		qtdAlunos int
		want      []veiculos.PlanoCategoriaVeiculo
	}{
		{
			name:      "7 alunos uses carro",
			qtdAlunos: 7,
			want: []veiculos.PlanoCategoriaVeiculo{
				{Categoria: veiculos.CategoriaCarroSeteLugares, Capacidade: veiculos.CapacidadeCarroSeteLugares, Quantidade: 1},
			},
		},
		{
			name:      "10 alunos uses escolar",
			qtdAlunos: 10,
			want: []veiculos.PlanoCategoriaVeiculo{
				{Categoria: veiculos.CategoriaEscolar, Capacidade: veiculos.CapacidadeEscolar, Quantidade: 1},
			},
		},
		{
			name:      "25 alunos uses executivo",
			qtdAlunos: 25,
			want: []veiculos.PlanoCategoriaVeiculo{
				{Categoria: veiculos.CategoriaExecutivo, Capacidade: veiculos.CapacidadeExecutivo, Quantidade: 1},
			},
		},
		{
			name:      "48 alunos uses executivo and carro",
			qtdAlunos: 48,
			want: []veiculos.PlanoCategoriaVeiculo{
				{Categoria: veiculos.CategoriaExecutivo, Capacidade: veiculos.CapacidadeExecutivo, Quantidade: 1},
				{Categoria: veiculos.CategoriaCarroSeteLugares, Capacidade: veiculos.CapacidadeCarroSeteLugares, Quantidade: 1},
			},
		},
		{
			name:      "70 alunos uses executivo and escolar",
			qtdAlunos: 70,
			want: []veiculos.PlanoCategoriaVeiculo{
				{Categoria: veiculos.CategoriaExecutivo, Capacidade: veiculos.CapacidadeExecutivo, Quantidade: 1},
				{Categoria: veiculos.CategoriaEscolar, Capacidade: veiculos.CapacidadeEscolar, Quantidade: 1},
			},
		},
		{
			name:      "92 alunos uses two executivos",
			qtdAlunos: 92,
			want: []veiculos.PlanoCategoriaVeiculo{
				{Categoria: veiculos.CategoriaExecutivo, Capacidade: veiculos.CapacidadeExecutivo, Quantidade: 2},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := veiculos.PlanejarCategoriasPorQuantidade(tc.qtdAlunos)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			assertPlano(t, got, tc.want)
		})
	}
}

func TestPlanejarCategoriasPorQuantidadeValidation(t *testing.T) {
	_, err := veiculos.PlanejarCategoriasPorQuantidade(0)

	if !errors.Is(err, brerror.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestValidateCategoriaCapacidade(t *testing.T) {
	tests := []struct {
		name       string
		categoria  veiculos.CategoriaVeiculo
		capacidade int16
		wantErr    bool
	}{
		{name: "executivo valid", categoria: veiculos.CategoriaExecutivo, capacidade: 46},
		{name: "escolar valid", categoria: veiculos.CategoriaEscolar, capacidade: 24},
		{name: "carro valid", categoria: veiculos.CategoriaCarroSeteLugares, capacidade: 7},
		{name: "executivo invalid", categoria: veiculos.CategoriaExecutivo, capacidade: 45, wantErr: true},
		{name: "unknown category", categoria: "van", capacidade: 15, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := veiculos.ValidateCategoriaCapacidade(tc.categoria, tc.capacidade)
			if tc.wantErr && !errors.Is(err, brerror.ErrInvalidInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

func assertPlano(t *testing.T, got, want []veiculos.PlanoCategoriaVeiculo) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d items, got %d: %+v", len(want), len(got), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("unexpected item %d: want %+v, got %+v", i, want[i], got[i])
		}
	}
}
