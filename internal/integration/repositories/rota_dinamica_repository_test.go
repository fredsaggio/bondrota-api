//go:build integration

package repositories

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
	"github.com/stretchr/testify/require"
)

func TestRotaDinamicaRepository_PersistsOrderedDestinations(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	secondDestinoID := seedDestino(t, ctx, tx, "IFAL", testMunicipioID)
	tripDate := futureTripDate()
	cicloID := seedCiclo(t, ctx, tx, fixture, tripDate)
	viagemID := seedViagem(t, ctx, tx, cicloID, "ida")
	store := rotasdinamicas.NewRotaDinamicaStore(tx)
	expiresAt := time.Now().UTC().Add(2 * time.Hour)

	created, err := store.Create(ctx, rotasdinamicas.RotaDinamicaInput{
		ViagemID: viagemID, Provider: "osrm",
		Origem:          rotasdinamicas.PontoRota{Nome: "Terminal", Latitude: -9.65, Longitude: -35.72},
		DestinoFinal:    rotasdinamicas.PontoRota{Nome: "IFAL", Latitude: -9.67, Longitude: -35.74},
		DistanciaMetros: 15000, DuracaoSegundos: 1800,
		Geometry:  json.RawMessage(`{"type":"LineString","coordinates":[]}`),
		ExpiresAt: expiresAt,
		Destinos: []rotasdinamicas.RotaDinamicaDestinoInput{
			{DestinoID: fixture.DestinoID, Ordem: 1},
			{DestinoID: secondDestinoID, Ordem: 2},
		},
	})
	require.NoError(t, err)
	require.Len(t, created.Destinos, 2)

	got, err := store.GetByViagem(ctx, viagemID)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, []int{got.Destinos[0].Ordem, got.Destinos[1].Ordem})

	cycleExpiry, err := store.GetExpiresAtByViagem(ctx, viagemID)
	require.NoError(t, err)
	require.True(t, cycleExpiry.After(tripDate))

	require.NoError(t, store.DeleteByViagem(ctx, viagemID))
	_, err = store.GetByViagem(ctx, viagemID)
	require.ErrorIs(t, err, brerror.ErrNotFound)
}
