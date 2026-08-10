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
	input := viagens.CicloViagemComReservasInput{
		Ciclo: viagens.CicloViagemInput{
			DataViagem: tripDate, Turno: viagens.TurnoNoturno,
			MunicipioDestinoID: testMunicipioID,
			RotaInternaID:      fixture.RotaInternaID, VeiculoID: fixture.VeiculoID,
			MotoristaID: fixture.MotoristaID,
			ExpiresAt:   time.Date(tripDate.Year(), tripDate.Month(), tripDate.Day(), 23, 59, 0, 0, time.UTC),
		},
		ReservaIDsIda:   []int64{idaID},
		ReservaIDsVolta: []int64{voltaID},
	}
	partidas := map[viagens.SentidoViagem]time.Time{
		viagens.SentidoIda:   tripDate.Add(17 * time.Hour),
		viagens.SentidoVolta: tripDate.Add(22 * time.Hour),
	}

	created, err := store.CreateCiclosComViagens(ctx, []viagens.CicloViagemComReservasInput{input}, partidas)
	require.NoError(t, err)
	require.Len(t, created.Ciclos, 1)
	createdCiclo := created.Ciclos[0]
	require.Len(t, createdCiclo.Viagens, 2)
	require.Equal(t, viagens.SentidoIda, createdCiclo.Viagens[0].Sentido)
	require.Equal(t, viagens.SentidoVolta, createdCiclo.Viagens[1].Sentido)

	var linkedIda, linkedVolta int64
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT reserva_id FROM viagem_reservas WHERE viagem_id = $1`, createdCiclo.Viagens[0].ID,
	).Scan(&linkedIda))
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT reserva_id FROM viagem_reservas WHERE viagem_id = $1`, createdCiclo.Viagens[1].ID,
	).Scan(&linkedVolta))
	require.Equal(t, idaID, linkedIda)
	require.Equal(t, voltaID, linkedVolta)

	got, err := store.GetCicloByID(ctx, createdCiclo.Ciclo.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.VeiculoID, got.VeiculoID)
	require.Equal(t, testMunicipioID, got.MunicipioDestinoID)

	updated, err := store.UpdateCiclo(ctx, createdCiclo.Ciclo.ID, func(current *viagens.CicloViagem) (bool, error) {
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
		DataViagem: tripDate, Turno: viagens.TurnoNoturno, MunicipioDestinoID: testMunicipioID,
		RotaInternaID: fixture.RotaInternaID, Sentido: viagens.SentidoIda,
	})
	require.NoError(t, err)
	require.Equal(t, []viagens.PlanejamentoReserva{{ID: wantedID, DestinoID: fixture.DestinoID}}, reservations)
}
