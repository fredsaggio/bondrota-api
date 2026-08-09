//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestCicloViagemRepository_CreatesCycleWithTripsAndReservations(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	tripDate := futureTripDate()
	idaID := seedReserva(t, ctx, tx, fixture, "ida", tripDate)
	voltaID := seedReserva(t, ctx, tx, fixture, "volta", tripDate)
	store := viagens.NewCicloViagemStore(tx)
	input := viagens.CicloViagemInput{
		DataViagem: tripDate, Turno: viagens.TurnoNoturno, Cidade: testCity,
		RotaInternaID: fixture.RotaInternaID, VeiculoID: fixture.VeiculoID,
		MotoristaID: fixture.MotoristaID,
		ExpiresAt:   time.Date(tripDate.Year(), tripDate.Month(), tripDate.Day(), 23, 59, 0, 0, time.UTC),
	}
	partidas := map[viagens.SentidoViagem]time.Time{
		viagens.SentidoIda:   tripDate.Add(17 * time.Hour),
		viagens.SentidoVolta: tripDate.Add(22 * time.Hour),
	}

	created, err := store.CreateCicloComViagens(ctx, input, partidas)
	require.NoError(t, err)
	require.Len(t, created.Viagens, 2)
	require.Equal(t, viagens.SentidoIda, created.Viagens[0].Sentido)
	require.Equal(t, viagens.SentidoVolta, created.Viagens[1].Sentido)

	var linkedIda, linkedVolta int64
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT reserva_id FROM viagem_reservas WHERE viagem_id = $1`, created.Viagens[0].ID,
	).Scan(&linkedIda))
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT reserva_id FROM viagem_reservas WHERE viagem_id = $1`, created.Viagens[1].ID,
	).Scan(&linkedVolta))
	require.Equal(t, idaID, linkedIda)
	require.Equal(t, voltaID, linkedVolta)

	got, err := store.GetCicloByID(ctx, created.Ciclo.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.VeiculoID, got.VeiculoID)

	updated, err := store.UpdateCiclo(ctx, created.Ciclo.ID, func(current *viagens.CicloViagem) (bool, error) {
		current.Status = viagens.StatusCicloEmAndamento
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, viagens.StatusCicloEmAndamento, updated.Status)
}

func TestCicloViagemRepository_FiltersPlanningReservations(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	tripDate := futureTripDate()
	wantedID := seedReserva(t, ctx, tx, fixture, "ida", tripDate)
	seedReserva(t, ctx, tx, fixture, "volta", tripDate)
	store := viagens.NewCicloViagemStore(tx)

	reservations, err := store.ListReservasConfirmadasParaPlanejamento(ctx, viagens.PlanejamentoReservasFiltro{
		DataViagem: tripDate, Turno: viagens.TurnoNoturno, Cidade: testCity,
		RotaInternaID: fixture.RotaInternaID, Sentido: viagens.SentidoIda,
	})
	require.NoError(t, err)
	require.Equal(t, []viagens.PlanejamentoReserva{{ID: wantedID, DestinoID: fixture.DestinoID}}, reservations)
}
