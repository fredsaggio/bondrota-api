//go:build integration

package repositories

import (
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/destinos"
	"github.com/stretchr/testify/require"
)

func TestDestinoRepository_CRUDAndCityFilter(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := destinos.NewDestinoStore(tx)

	created, err := store.Create(ctx, destinos.DestinoInput{
		Nome: "Campus A", Rua: "Rua A", Cidade: testCity,
		Latitude: -9.6658, Longitude: -35.7353,
	})
	require.NoError(t, err)

	got, err := store.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created, got)

	byCity, err := store.ListByCity(ctx, testCity)
	require.NoError(t, err)
	require.Len(t, byCity, 1)

	updated, err := store.Update(ctx, created.ID, func(current *destinos.Destino) (bool, error) {
		current.Nome = "Campus B"
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, "Campus B", updated.Nome)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, destinos.ErrNotFound)
}
