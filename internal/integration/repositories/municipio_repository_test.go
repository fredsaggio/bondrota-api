//go:build integration

package repositories

import (
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
