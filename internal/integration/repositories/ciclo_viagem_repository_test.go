//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestCicloViagemRepository_PlansOutboundAndReturnSeparately(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	secondFixture := fixture
	secondFixture.ClienteID = seedCliente(t, ctx, tx, "20000000002")
	secondFixture.VinculoID = seedVinculo(t, ctx, tx, secondFixture.ClienteID, fixture.DestinoID, fixture.RotaInternaID)

	tripDate := futureTripDate()
	idaID := seedReserva(t, ctx, tx, fixture, "ida", tripDate)
	voltaID := seedReserva(t, ctx, tx, fixture, "volta", tripDate)
	secondIdaID := seedReserva(t, ctx, tx, secondFixture, "ida", tripDate)
	secondVoltaID := seedReserva(t, ctx, tx, secondFixture, "volta", tripDate)
	store := viagens.NewCicloViagemStore(tx)

	createdIda, err := store.CreatePlanejamentoIda(ctx, []viagens.CicloIdaComReservasInput{{
		Ciclo: viagens.CicloViagemInput{
			DataViagem:         tripDate,
			Turno:              viagens.TurnoNoturno,
			MunicipioDestinoID: testMunicipioID,
			RotaInternaID:      fixture.RotaInternaID,
			VeiculoID:          fixture.VeiculoID,
			MotoristaID:        fixture.MotoristaID,
			ExpiresAt:          time.Date(tripDate.Year(), tripDate.Month()+3, tripDate.Day(), 0, 0, 0, 0, time.UTC),
		},
		ReservaIDs: []int64{idaID, secondIdaID},
	}}, tripDate.Add(17*time.Hour))
	require.NoError(t, err)
	require.Equal(t, viagens.SentidoIda, createdIda.Sentido)
	require.Len(t, createdIda.Ciclos, 1)
	require.Len(t, createdIda.Ciclos[0].Viagens, 1)
	require.Equal(t, viagens.SentidoIda, createdIda.Ciclos[0].Viagens[0].Sentido)

	filtroVolta := viagens.PlanejamentoReservasFiltro{
		DataViagem:         tripDate,
		Turno:              viagens.TurnoNoturno,
		MunicipioDestinoID: testMunicipioID,
		RotaInternaID:      fixture.RotaInternaID,
		Sentido:            viagens.SentidoVolta,
	}

	reservasVolta, err := store.ListReservasElegiveisParaVolta(ctx, filtroVolta)
	require.NoError(t, err)
	require.Empty(t, reservasVolta)

	_, err = tx.Exec(ctx, `
		UPDATE viagem_reservas
		SET status_presenca = 'embarcou'
		WHERE viagem_id = $1 AND reserva_id = $2`, createdIda.Ciclos[0].Viagens[0].ID, idaID)
	require.NoError(t, err)

	reservasVolta, err = store.ListReservasElegiveisParaVolta(ctx, filtroVolta)
	require.NoError(t, err)
	require.Equal(t, []viagens.PlanejamentoReserva{{ID: voltaID, DestinoID: fixture.DestinoID}}, reservasVolta)

	ciclosVolta, err := store.ListCiclosParaPlanejamentoVolta(ctx, filtroVolta)
	require.NoError(t, err)
	require.Len(t, ciclosVolta, 1)
	require.Equal(t, createdIda.Ciclos[0].Ciclo.ID, ciclosVolta[0].Ciclo.ID)
	require.Equal(t, 7, ciclosVolta[0].Capacidade)

	createdVolta, err := store.CreatePlanejamentoVolta(ctx, []viagens.CicloVoltaComReservasInput{{
		Ciclo:      ciclosVolta[0].Ciclo,
		ReservaIDs: []int64{voltaID},
	}}, tripDate.Add(22*time.Hour))
	require.NoError(t, err)
	require.Equal(t, viagens.SentidoVolta, createdVolta.Sentido)
	require.Len(t, createdVolta.Ciclos, 1)
	require.Equal(t, createdIda.Ciclos[0].Ciclo.ID, createdVolta.Ciclos[0].Ciclo.ID)
	require.Equal(t, fixture.VeiculoID, createdVolta.Ciclos[0].Ciclo.VeiculoID)
	require.Equal(t, fixture.MotoristaID, createdVolta.Ciclos[0].Ciclo.MotoristaID)
	require.Len(t, createdVolta.Ciclos[0].Viagens, 1)
	require.Equal(t, viagens.SentidoVolta, createdVolta.Ciclos[0].Viagens[0].Sentido)

	var linkedVoltaID int64
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT reserva_id FROM viagem_reservas WHERE viagem_id = $1`, createdVolta.Ciclos[0].Viagens[0].ID,
	).Scan(&linkedVoltaID))
	require.Equal(t, voltaID, linkedVoltaID)

	var secondLinkedCount int
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM viagem_reservas WHERE reserva_id = $1`, secondVoltaID,
	).Scan(&secondLinkedCount))
	require.Zero(t, secondLinkedCount)

	_, err = store.CreatePlanejamentoVolta(ctx, []viagens.CicloVoltaComReservasInput{{
		Ciclo: ciclosVolta[0].Ciclo,
	}}, tripDate.Add(22*time.Hour))
	require.ErrorIs(t, err, brerror.ErrAlreadyExists)

	got, err := store.GetCicloByID(ctx, createdIda.Ciclos[0].Ciclo.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.VeiculoID, got.VeiculoID)
	require.Equal(t, testMunicipioID, got.MunicipioDestinoID)
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
