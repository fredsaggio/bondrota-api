package veiculos_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
)

type fakeAlocacaoVeiculoStore struct {
	listDisponiveisFn func(ctx context.Context, filtro veiculos.VeiculosDisponiveisFiltro) ([]veiculos.Veiculo, error)
}

func (s fakeAlocacaoVeiculoStore) ListDisponiveisParaAlocacao(ctx context.Context, filtro veiculos.VeiculosDisponiveisFiltro) ([]veiculos.Veiculo, error) {
	return s.listDisponiveisFn(ctx, filtro)
}

func TestAlocacaoService_Alocar(t *testing.T) {
	dataViagem := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)

	t.Run("selects real vehicles from planned categories", func(t *testing.T) {
		svc := veiculos.NewAlocacaoService(fakeAlocacaoVeiculoStore{
			listDisponiveisFn: func(_ context.Context, filtro veiculos.VeiculosDisponiveisFiltro) ([]veiculos.Veiculo, error) {
				if filtro.Turno != "NT" {
					t.Fatalf("unexpected turno: %s", filtro.Turno)
				}
				if !filtro.DataViagem.Equal(dataViagem) {
					t.Fatalf("unexpected data_viagem: %v", filtro.DataViagem)
				}
				assertCategorias(t, filtro.Categorias, []veiculos.CategoriaVeiculo{
					veiculos.CategoriaExecutivo,
					veiculos.CategoriaCarroSeteLugares,
					veiculos.CategoriaEscolar,
				})
				return []veiculos.Veiculo{
					{ID: 1, Categoria: veiculos.CategoriaExecutivo, Capacidade: veiculos.CapacidadeExecutivo},
					{ID: 2, Categoria: veiculos.CategoriaCarroSeteLugares, Capacidade: veiculos.CapacidadeCarroSeteLugares},
				}, nil
			},
		})

		alocacao, err := svc.Alocar(context.Background(), veiculos.AlocarVeiculosInput{
			DataViagem:       dataViagem,
			Turno:            "NT",
			QuantidadeAlunos: 48,
		})

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(alocacao.Veiculos) != 2 {
			t.Fatalf("expected 2 vehicles, got %d", len(alocacao.Veiculos))
		}
		if alocacao.CapacidadeTotal != 53 {
			t.Fatalf("expected capacity 53, got %d", alocacao.CapacidadeTotal)
		}
		if alocacao.Veiculos[0].Categoria != veiculos.CategoriaExecutivo {
			t.Fatalf("expected first vehicle executivo, got %s", alocacao.Veiculos[0].Categoria)
		}
		if alocacao.Veiculos[1].Categoria != veiculos.CategoriaCarroSeteLugares {
			t.Fatalf("expected second vehicle carro, got %s", alocacao.Veiculos[1].Categoria)
		}
	})

	t.Run("fallbacks to smallest larger vehicle when ideal category is unavailable", func(t *testing.T) {
		svc := veiculos.NewAlocacaoService(fakeAlocacaoVeiculoStore{
			listDisponiveisFn: func(_ context.Context, filtro veiculos.VeiculosDisponiveisFiltro) ([]veiculos.Veiculo, error) {
				assertCategorias(t, filtro.Categorias, []veiculos.CategoriaVeiculo{
					veiculos.CategoriaCarroSeteLugares,
					veiculos.CategoriaEscolar,
					veiculos.CategoriaExecutivo,
				})
				return []veiculos.Veiculo{
					{ID: 10, Categoria: veiculos.CategoriaExecutivo, Capacidade: veiculos.CapacidadeExecutivo},
					{ID: 11, Categoria: veiculos.CategoriaEscolar, Capacidade: veiculos.CapacidadeEscolar},
				}, nil
			},
		})

		alocacao, err := svc.Alocar(context.Background(), veiculos.AlocarVeiculosInput{
			DataViagem:       dataViagem,
			Turno:            "NT",
			QuantidadeAlunos: 1,
		})

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(alocacao.Veiculos) != 1 {
			t.Fatalf("expected 1 vehicle, got %d", len(alocacao.Veiculos))
		}
		if alocacao.Veiculos[0].ID != 11 {
			t.Fatalf("expected escolar fallback vehicle 11, got %+v", alocacao.Veiculos[0])
		}
		if alocacao.CapacidadeTotal != int(veiculos.CapacidadeEscolar) {
			t.Fatalf("expected escolar capacity, got %d", alocacao.CapacidadeTotal)
		}
	})

	t.Run("returns not found when category capacity is unavailable", func(t *testing.T) {
		svc := veiculos.NewAlocacaoService(fakeAlocacaoVeiculoStore{
			listDisponiveisFn: func(_ context.Context, _ veiculos.VeiculosDisponiveisFiltro) ([]veiculos.Veiculo, error) {
				return []veiculos.Veiculo{
					{ID: 1, Categoria: veiculos.CategoriaExecutivo, Capacidade: veiculos.CapacidadeExecutivo},
				}, nil
			},
		})

		_, err := svc.Alocar(context.Background(), veiculos.AlocarVeiculosInput{
			DataViagem:       dataViagem,
			Turno:            "NT",
			QuantidadeAlunos: 48,
		})

		if !errors.Is(err, brerror.ErrNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})

	t.Run("validates input before calling store", func(t *testing.T) {
		svc := veiculos.NewAlocacaoService(fakeAlocacaoVeiculoStore{
			listDisponiveisFn: func(_ context.Context, _ veiculos.VeiculosDisponiveisFiltro) ([]veiculos.Veiculo, error) {
				t.Fatal("store should not be called")
				return nil, nil
			},
		})

		_, err := svc.Alocar(context.Background(), veiculos.AlocarVeiculosInput{})

		if !errors.Is(err, brerror.ErrInvalidInput) {
			t.Fatalf("expected invalid input, got %v", err)
		}
	})
}

func assertCategorias(t *testing.T, got, want []veiculos.CategoriaVeiculo) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d categorias, got %d: %+v", len(want), len(got), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("unexpected categoria %d: want %s, got %s", i, want[i], got[i])
		}
	}
}
