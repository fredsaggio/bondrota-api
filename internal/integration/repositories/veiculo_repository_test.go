//go:build integration

package repositories

import (
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/veiculos"
	"github.com/stretchr/testify/require"
)

func TestVeiculoRepository_CRUD(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := veiculos.NewVeiculoStore(tx)
	input := veiculos.VeiculoInput{
		Placa: "INT1001", Modelo: "Van", Categoria: veiculos.CategoriaCarroSeteLugares,
		Capacidade: 7, Status: veiculos.StatusAtivo, ArCondicionado: true,
	}

	created, err := store.Create(ctx, input)
	require.NoError(t, err)

	got, err := store.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created, got)

	updated, err := store.Update(ctx, created.ID, func(current *veiculos.Veiculo) (bool, error) {
		current.Status = veiculos.StatusManutencao
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, veiculos.StatusManutencao, updated.Status)

	listed, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, veiculos.ErrNotFound)
}

func TestVeiculoRepository_ListsOnlyAvailableVehicles(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	availableID := seedVeiculo(t, ctx, tx, "INT1002", "ativo")
	seedVeiculo(t, ctx, tx, "INT1003", "manutencao")
	tripDate := futureTripDate()
	seedCiclo(t, ctx, tx, fixture, tripDate)

	store := veiculos.NewAlocacaoVeiculoStore(tx)
	available, err := store.ListDisponiveisParaAlocacao(ctx, veiculos.VeiculosDisponiveisFiltro{
		DataViagem: tripDate, Turno: "NT",
		Categorias: []veiculos.CategoriaVeiculo{veiculos.CategoriaCarroSeteLugares},
	})
	require.NoError(t, err)
	require.Len(t, available, 1)
	require.Equal(t, availableID, available[0].ID)
}
