//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestAgendadorPlanejamentoRepository_ListsOutboundDemandAndExistingReturns(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	tripDate := futureTripDate()
	seedReserva(t, ctx, tx, fixture, "ida", tripDate)
	seedReserva(t, ctx, tx, fixture, "ida", tripDate.AddDate(0, 0, 1))

	horarioStore := viagens.NewHorarioTurnoViagemStore(tx)
	_, err := horarioStore.Create(ctx, viagens.HorarioTurnoViagemInput{
		MunicipioDestinoID: testMunicipioID,
		Turno:              viagens.TurnoNoturno,
		HorarioIda:         17 * time.Hour,
		HorarioVolta:       22 * time.Hour,
	})
	require.NoError(t, err)

	cicloID := seedCiclo(t, ctx, tx, fixture, tripDate)
	seedViagem(t, ctx, tx, cicloID, "ida")

	store := viagens.NewAgendadorPlanejamentoStore(tx)
	candidatos, err := store.ListCandidatos(ctx, tripDate, tripDate)
	require.NoError(t, err)
	require.Equal(t, []viagens.CandidatoPlanejamento{
		{
			Chave: viagens.ChaveExecucaoPlanejamento{
				DataViagem:         tripDate,
				Turno:              viagens.TurnoNoturno,
				MunicipioDestinoID: testMunicipioID,
				RotaInternaID:      fixture.RotaInternaID,
				Sentido:            viagens.SentidoIda,
			},
			HorarioPartida: 17 * time.Hour,
		},
		{
			Chave: viagens.ChaveExecucaoPlanejamento{
				DataViagem:         tripDate,
				Turno:              viagens.TurnoNoturno,
				MunicipioDestinoID: testMunicipioID,
				RotaInternaID:      fixture.RotaInternaID,
				Sentido:            viagens.SentidoVolta,
			},
			HorarioPartida: 22 * time.Hour,
		},
	}, candidatos)

	semCandidatos, err := store.ListCandidatos(ctx, tripDate.AddDate(0, 0, 2), tripDate.AddDate(0, 0, 2))
	require.NoError(t, err)
	require.Empty(t, semCandidatos)
}
