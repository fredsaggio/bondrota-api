//go:build integration

package repositories

import (
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestViagemReservaRepository_AllocatesAndRegistersPresence(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	tripDate := futureTripDate()
	reservaID := seedReserva(t, ctx, tx, fixture, "ida", tripDate)
	cicloID := seedCiclo(t, ctx, tx, fixture, tripDate)
	viagemID := seedViagem(t, ctx, tx, cicloID, "ida")
	store := viagens.NewViagemReservaStore(tx)

	created, err := store.CreateViagemReserva(ctx, viagens.ViagemReservaInput{
		ViagemID: viagemID, ReservaID: reservaID,
	})
	require.NoError(t, err)
	require.Equal(t, viagens.StatusPresencaAguardando, created.StatusPresenca)

	listed, err := store.ListReservasByViagem(ctx, viagemID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, fixture.ClienteID, listed[0].ClienteID)

	updated, err := store.UpdatePresenca(ctx, viagemID, reservaID, func(current *viagens.ViagemReserva) (bool, error) {
		current.StatusPresenca = viagens.StatusPresencaEmbarcou
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, viagens.StatusPresencaEmbarcou, updated.StatusPresenca)

	var confirmations int
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM viagem_reserva_confirmacoes WHERE viagem_reserva_id = $1`, created.ID,
	).Scan(&confirmations))
	require.Equal(t, 1, confirmations)
}
