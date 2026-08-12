//go:build integration

package repositories

import (
	"errors"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/municipios"
	"github.com/stretchr/testify/require"
)

func TestMunicipioRepository_UpsertsAndListsByUF(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := municipios.NewStore(tx)

	err := store.Upsert(ctx, []municipios.Municipio{
		{CodigoIBGE: 2700300, Nome: "Arapiraca", UF: "AL"},
		{CodigoIBGE: testMunicipioID, Nome: "Maceio Atualizada", UF: "AL"},
		{CodigoIBGE: 2611606, Nome: "Recife", UF: "PE"},
	})
	require.NoError(t, err)

	alagoas, err := store.ListByUF(ctx, "al")
	require.NoError(t, err)
	require.Len(t, alagoas, 2)
	require.Equal(t, "Arapiraca", alagoas[0].Nome)
	require.Equal(t, "Maceio Atualizada", alagoas[1].Nome)
}

func TestMunicipioRepository_GetByID(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := municipios.NewStore(tx)

	require.NoError(t, store.Upsert(ctx, []municipios.Municipio{
		{CodigoIBGE: 2611606, Nome: "Recife", UF: "PE"},
	}))

	t.Run("acha municipio de qualquer UF, nao so a do listByUF", func(t *testing.T) {
		found, err := store.GetByID(ctx, 2611606)
		require.NoError(t, err)
		require.Equal(t, "Recife", found.Nome)
		require.Equal(t, "PE", found.UF)
	})

	t.Run("codigo inexistente devolve ErrNotFound", func(t *testing.T) {
		_, err := store.GetByID(ctx, 9999999)
		require.True(t, errors.Is(err, municipios.ErrNotFound))
	})
}
