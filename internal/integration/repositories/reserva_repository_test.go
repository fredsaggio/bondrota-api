//go:build integration

package repositories

import (
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/reservas"
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
