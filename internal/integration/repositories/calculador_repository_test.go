//go:build integration

package repositories

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
	"github.com/stretchr/testify/require"
)

func TestCalculadorRepository_LoadsRouteInputsAndPendingTrips(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	tripDate := futureTripDate()
	reservaID := seedReserva(t, ctx, tx, fixture, "ida", tripDate)
	cicloID := seedCiclo(t, ctx, tx, fixture, tripDate)
	viagemID := seedViagem(t, ctx, tx, cicloID, "ida")
	seedViagemReserva(t, ctx, tx, viagemID, reservaID)
	now := time.Now().UTC()
	partida := now.Add(90 * time.Minute)
	_, err := tx.Exec(ctx, `
		INSERT INTO viagem_horarios (viagem_id, tipo, horario)
		VALUES ($1, 'partida_prevista', $2)`, viagemID, partida)
	require.NoError(t, err)
	store := rotasdinamicas.NewCalculadorRotaDinamicaStore(tx)

	dados, err := store.GetDadosCalculo(ctx, viagemID)
	require.NoError(t, err)
	require.Equal(t, "ida", dados.Sentido)
	require.Len(t, dados.Paradas, 1)
	require.Equal(t, fixture.ParadaID, dados.Paradas[0].ID)
	require.Len(t, dados.Destinos, 1)
	require.Equal(t, fixture.DestinoID, dados.Destinos[0].ID)

	pending, err := store.ListViagensPendentesCalculo(ctx, now, 2*time.Hour, 30*time.Minute)
	require.NoError(t, err)
	require.Equal(t, []int64{viagemID}, pending)

	_, err = rotasdinamicas.NewRotaDinamicaStore(tx).Create(ctx, rotasdinamicas.RotaDinamicaInput{
		ViagemID: viagemID, Provider: "osrm",
		Origem:          rotasdinamicas.PontoRota{Nome: "Terminal", Latitude: -9.65, Longitude: -35.72},
		DestinoFinal:    rotasdinamicas.PontoRota{Nome: "UFAL", Latitude: -9.66, Longitude: -35.73},
		DistanciaMetros: 1000, DuracaoSegundos: 300,
		Geometry:  json.RawMessage(`{"type":"LineString","coordinates":[]}`),
		ExpiresAt: now.Add(2 * time.Hour),
		Destinos:  []rotasdinamicas.RotaDinamicaDestinoInput{{DestinoID: fixture.DestinoID, Ordem: 1}},
	})
	require.NoError(t, err)
	require.NoError(t, store.DeleteRotasPorReservaAntesDoBloqueio(ctx, reservaID, now, 30*time.Minute))

	var routeCount int
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM rotas_dinamicas WHERE viagem_id = $1`, viagemID,
	).Scan(&routeCount))
	require.Zero(t, routeCount)
}
