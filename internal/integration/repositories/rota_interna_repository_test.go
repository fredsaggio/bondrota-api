//go:build integration

package repositories

import (
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
	"github.com/stretchr/testify/require"
)

func TestRotaInternaRepository_PersistsOrderedStops(t *testing.T) {
	ctx, tx := beginTestTx(t)
	firstID := seedParada(t, ctx, tx, "Primeira")
	secondID := seedParada(t, ctx, tx, "Segunda")
	store := rotasinternas.NewRotaInternaStore(tx)

	created, err := store.Create(ctx, rotasinternas.CreateRotaInternaInput{
		Paradas: []rotasinternas.ParadaInput{
			{ParadaID: secondID, Ordem: 2},
			{ParadaID: firstID, Ordem: 1},
		},
	})
	require.NoError(t, err)
	require.Len(t, created.Paradas, 2)

	got, err := store.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, []int{got.Paradas[0].Ordem, got.Paradas[1].Ordem})
	require.Equal(t, firstID, got.Paradas[0].ID)

	updated, err := store.UpdateParadas(ctx, created.ID, rotasinternas.UpdateParadasInput{
		Paradas: []rotasinternas.ParadaInput{{ParadaID: secondID, Ordem: 1}},
	})
	require.NoError(t, err)
	require.Len(t, updated.Paradas, 1)
	require.Equal(t, secondID, updated.Paradas[0].ID)

	listed, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, rotasinternas.ErrNotFound)
}
