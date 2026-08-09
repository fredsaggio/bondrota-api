//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/stretchr/testify/require"
)

func TestMotoristaRepository_CRUD(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := motoristas.NewMotoristaStore(tx)

	created, err := store.Create(ctx, motoristas.MotoristaInput{
		Nome: "Carlos", CPF: "40000000001", Senha: "hash", Telefone: "82999990000",
		DataNasc: time.Date(1985, 5, 20, 0, 0, 0, 0, time.UTC), Turno: motoristas.TurnoIntegral,
		CidadeTrabalho: testCity, Residencia: testCity,
	})
	require.NoError(t, err)

	byCPF, err := store.GetByCPF(ctx, created.CPF)
	require.NoError(t, err)
	require.Equal(t, "hash", byCPF.Senha)

	updated, err := store.Update(ctx, created.ID, func(current *motoristas.Motorista) (bool, error) {
		current.Nome = "Carlos Silva"
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, "Carlos Silva", updated.Nome)

	listed, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, motoristas.ErrNotFound)
}

func TestMotoristaRepository_ListsOnlyAvailableDrivers(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	availableID := seedMotorista(t, ctx, tx, "40000000002", testCity, "IN")
	tripDate := futureTripDate()
	seedCiclo(t, ctx, tx, fixture, tripDate)

	store := motoristas.NewAlocacaoMotoristaStore(tx)
	available, err := store.ListDisponiveisParaAlocacao(ctx, motoristas.MotoristasDisponiveisFiltro{
		Cidade: "maceio", DataViagem: tripDate, Turno: motoristas.TurnoNoturno, Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, available, 1)
	require.Equal(t, availableID, available[0].ID)
}
