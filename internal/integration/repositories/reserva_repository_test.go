//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/reservas"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestReservaRepository_CRUDAndVinculoSnapshot(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := reservas.NewReservaStore(tx)
	tripDate := futureTripDate()

	created, err := store.Create(ctx, reservas.ReservaInput{
		ClienteID: fixture.ClienteID, VinculoID: fixture.VinculoID, DataViagem: tripDate,
		Turno: reservas.TurnoNoturno, DestinoID: fixture.DestinoID,
		RotaInternaID: fixture.RotaInternaID, Sentido: reservas.SentidoIda,
	})
	require.NoError(t, err)
	require.Equal(t, reservas.StatusConfirmada, created.Status)

	snapshot, err := store.GetVinculoSnapshot(ctx, fixture.VinculoID)
	require.NoError(t, err)
	require.Equal(t, fixture.ClienteID, snapshot.ClienteID)

	byCliente, err := store.ListByCliente(ctx, fixture.ClienteID)
	require.NoError(t, err)
	require.Len(t, byCliente, 1)
	byVinculo, err := store.ListByVinculo(ctx, fixture.ClienteID, fixture.VinculoID)
	require.NoError(t, err)
	require.Len(t, byVinculo, 1)

	updated, err := store.Update(ctx, created.ID, func(current *reservas.Reserva) (bool, error) {
		current.Status = reservas.StatusCancelada
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, reservas.StatusCancelada, updated.Status)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, reservas.ErrReservaNotFound)
}

func TestReservaRepository_GetHorarioPartidaPorSentido(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := reservas.NewReservaStore(tx)

	_, err := store.GetHorarioPartida(ctx, fixture.DestinoID, reservas.TurnoNoturno, reservas.SentidoIda)
	require.ErrorIs(t, err, reservas.ErrHorarioNaoConfigurado)

	_, err = tx.Exec(ctx, `
		INSERT INTO horarios_turno_viagem (
			municipio_destino_id, turno, horario_ida, horario_volta
		) VALUES ($1, 'NT', '17:00', '22:00')
	`, testMunicipioID)
	require.NoError(t, err)

	horarioIda, err := store.GetHorarioPartida(ctx, fixture.DestinoID, reservas.TurnoNoturno, reservas.SentidoIda)
	require.NoError(t, err)
	require.Equal(t, 17*time.Hour, horarioIda)

	horarioVolta, err := store.GetHorarioPartida(ctx, fixture.DestinoID, reservas.TurnoNoturno, reservas.SentidoVolta)
	require.NoError(t, err)
	require.Equal(t, 22*time.Hour, horarioVolta)
}

func TestReservaRepository_EnforcesOneActiveReservationPerDirection(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := reservas.NewReservaStore(tx)
	input := reservas.ReservaInput{
		ClienteID: fixture.ClienteID, VinculoID: fixture.VinculoID, DataViagem: futureTripDate(),
		Turno: reservas.TurnoNoturno, DestinoID: fixture.DestinoID,
		RotaInternaID: fixture.RotaInternaID, Sentido: reservas.SentidoIda,
	}

	_, err := store.Create(ctx, input)
	require.NoError(t, err)
	_, err = store.Create(ctx, input)
	require.Error(t, err)
}

func TestReservaRepository_RejectsConfirmedReservationAfterPlanningStarts(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	reservaStore := reservas.NewReservaStore(tx)
	execucaoStore := viagens.NewExecucaoPlanejamentoStore(tx)
	dataViagem := futureTripDate()
	agora := time.Date(dataViagem.Year(), dataViagem.Month(), dataViagem.Day(), 16, 30, 0, 0, time.UTC)

	_, adquirida, err := execucaoStore.TentarIniciar(ctx, viagens.IniciarExecucaoPlanejamentoInput{
		Chave: viagens.ChaveExecucaoPlanejamento{
			DataViagem:         dataViagem,
			Turno:              viagens.TurnoNoturno,
			MunicipioDestinoID: testMunicipioID,
			RotaInternaID:      fixture.RotaInternaID,
			Sentido:            viagens.SentidoIda,
		},
		PartidaEm:        agora.Add(30 * time.Minute),
		FechamentoEm:     agora,
		Agora:            agora,
		BloqueioExpiraEm: agora.Add(2 * time.Minute),
	})
	require.NoError(t, err)
	require.True(t, adquirida)

	_, err = reservaStore.Create(ctx, reservas.ReservaInput{
		ClienteID:     fixture.ClienteID,
		VinculoID:     fixture.VinculoID,
		DataViagem:    dataViagem,
		Turno:         reservas.TurnoNoturno,
		DestinoID:     fixture.DestinoID,
		RotaInternaID: fixture.RotaInternaID,
		Sentido:       reservas.SentidoIda,
	})
	require.ErrorIs(t, err, reservas.ErrPrazoReservaEncerrado)
}

func TestReservaRepository_RejectsReactivationAfterPlanningStarts(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	reservaStore := reservas.NewReservaStore(tx)
	execucaoStore := viagens.NewExecucaoPlanejamentoStore(tx)
	dataViagem := futureTripDate()

	created, err := reservaStore.Create(ctx, reservas.ReservaInput{
		ClienteID:     fixture.ClienteID,
		VinculoID:     fixture.VinculoID,
		DataViagem:    dataViagem,
		Turno:         reservas.TurnoNoturno,
		DestinoID:     fixture.DestinoID,
		RotaInternaID: fixture.RotaInternaID,
		Sentido:       reservas.SentidoVolta,
	})
	require.NoError(t, err)

	_, err = reservaStore.Update(ctx, created.ID, func(current *reservas.Reserva) (bool, error) {
		current.Status = reservas.StatusCancelada
		return true, nil
	})
	require.NoError(t, err)

	agora := time.Date(dataViagem.Year(), dataViagem.Month(), dataViagem.Day(), 21, 30, 0, 0, time.UTC)
	_, adquirida, err := execucaoStore.TentarIniciar(ctx, viagens.IniciarExecucaoPlanejamentoInput{
		Chave: viagens.ChaveExecucaoPlanejamento{
			DataViagem:         dataViagem,
			Turno:              viagens.TurnoNoturno,
			MunicipioDestinoID: testMunicipioID,
			RotaInternaID:      fixture.RotaInternaID,
			Sentido:            viagens.SentidoVolta,
		},
		PartidaEm:        agora.Add(30 * time.Minute),
		FechamentoEm:     agora,
		Agora:            agora,
		BloqueioExpiraEm: agora.Add(2 * time.Minute),
	})
	require.NoError(t, err)
	require.True(t, adquirida)

	_, err = reservaStore.Update(ctx, created.ID, func(current *reservas.Reserva) (bool, error) {
		current.Status = reservas.StatusConfirmada
		return true, nil
	})
	require.ErrorIs(t, err, reservas.ErrPrazoReservaEncerrado)

	stored, err := reservaStore.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, reservas.StatusCancelada, stored.Status)
}
