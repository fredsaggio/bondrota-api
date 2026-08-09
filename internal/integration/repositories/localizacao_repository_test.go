//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestLocalizacaoRepository_UpsertsLatestLocationAndChecksAccess(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	tripDate := futureTripDate()
	reservaID := seedReserva(t, ctx, tx, fixture, "ida", tripDate)
	cicloID := seedCiclo(t, ctx, tx, fixture, tripDate)
	viagemID := seedViagem(t, ctx, tx, cicloID, "ida")
	seedViagemReserva(t, ctx, tx, viagemID, reservaID)
	_, err := tx.Exec(ctx, `UPDATE viagens SET status = 'em_andamento' WHERE id = $1`, viagemID)
	require.NoError(t, err)
	store := viagens.NewViagemLocalizacaoStore(tx)
	now := time.Now().UTC().Truncate(time.Microsecond)

	created, err := store.Upsert(ctx, viagens.ViagemLocalizacaoInput{
		ViagemID: viagemID, MotoristaID: fixture.MotoristaID,
		Latitude: -9.65, Longitude: -35.72, VelocidadeKmh: 30,
		DirecaoGraus: 90, PrecisaoMetros: 8, RegistradaEm: now,
	})
	require.NoError(t, err)
	require.Equal(t, 30.0, created.VelocidadeKmh)

	updated, err := store.Upsert(ctx, viagens.ViagemLocalizacaoInput{
		ViagemID: viagemID, MotoristaID: fixture.MotoristaID,
		Latitude: -9.66, Longitude: -35.73, VelocidadeKmh: 42,
		DirecaoGraus: 180, PrecisaoMetros: 5, RegistradaEm: now.Add(time.Minute),
	})
	require.NoError(t, err)
	require.Equal(t, 42.0, updated.VelocidadeKmh)

	got, err := store.GetByViagem(ctx, viagemID)
	require.NoError(t, err)
	require.Equal(t, -9.66, got.Latitude)

	motoristaAllowed, err := store.CanMotoristaAccessViagem(ctx, viagemID, fixture.MotoristaID, true)
	require.NoError(t, err)
	require.True(t, motoristaAllowed)
	clienteAllowed, err := store.CanClienteAccessViagem(ctx, viagemID, fixture.ClienteID)
	require.NoError(t, err)
	require.True(t, clienteAllowed)
}
