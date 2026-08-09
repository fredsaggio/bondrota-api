//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestViagemRepository_CreatesListsAndTransitionsTrip(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	tripDate := futureTripDate()
	cicloID := seedCiclo(t, ctx, tx, fixture, tripDate)
	store := viagens.NewViagemStore(tx)
	partida := tripDate.Add(17 * time.Hour)

	created, err := store.CreateViagem(ctx, viagens.ViagemInput{
		CicloViagemID: cicloID, Sentido: viagens.SentidoIda, PartidaPrevista: partida,
	})
	require.NoError(t, err)
	require.Equal(t, viagens.StatusViagemProgramada, created.Status)

	got, err := store.GetViagemByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, cicloID, got.Ciclo.ID)

	byCycle, err := store.ListViagensByCiclo(ctx, cicloID)
	require.NoError(t, err)
	require.Len(t, byCycle, 1)
	hours, err := store.ListHorariosByViagem(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, hours, 1)
	require.Equal(t, viagens.TipoHorarioPartidaPrevista, hours[0].Tipo)

	startedAt := partida.Add(10 * time.Minute)
	started, err := store.AtualizarStatusERegistrarHorarioViagem(
		ctx, created.ID, viagens.StatusViagemProgramada, viagens.StatusViagemEmAndamento,
		viagens.TipoHorarioInicioReal, startedAt,
	)
	require.NoError(t, err)
	require.Equal(t, viagens.StatusViagemEmAndamento, started.Status)

	hours, err = store.ListHorariosByViagem(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, hours, 2)
}
