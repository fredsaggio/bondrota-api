//go:build integration

package repositories

import (
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/paradas"
	"github.com/stretchr/testify/require"
)

func TestParadaRepository_CRUD(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := paradas.NewParadaStore(tx)

	created, err := store.Create(ctx, paradas.ParadaInput{
		Nome: "Terminal A", Latitude: -9.65, Longitude: -35.72,
	})
	require.NoError(t, err)

	got, err := store.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created, got)

	listed, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	updated, err := store.Update(ctx, created.ID, func(current *paradas.Parada) (bool, error) {
		current.Nome = "Terminal B"
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, "Terminal B", updated.Nome)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, paradas.ErrNotFound)
}
