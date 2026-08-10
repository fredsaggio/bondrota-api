//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestExecucaoPlanejamentoRepository_ControlsIdempotencyAndRetry(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := viagens.NewExecucaoPlanejamentoStore(tx)
	agora := time.Date(2026, time.August, 12, 16, 30, 0, 0, time.FixedZone("America/Maceio", -3*60*60))
	input := viagens.IniciarExecucaoPlanejamentoInput{
		Chave: viagens.ChaveExecucaoPlanejamento{
			DataViagem:         agora,
			Turno:              viagens.TurnoNoturno,
			MunicipioDestinoID: testMunicipioID,
			RotaInternaID:      fixture.RotaInternaID,
			Sentido:            viagens.SentidoIda,
		},
		PartidaEm:        agora.Add(30 * time.Minute),
		FechamentoEm:     agora,
		Agora:            agora,
		BloqueioExpiraEm: agora.Add(2 * time.Minute),
	}

	created, claimed, err := store.TentarIniciar(ctx, input)
	require.NoError(t, err)
	require.True(t, claimed)
	require.Equal(t, viagens.StatusExecucaoProcessando, created.Status)
	require.Equal(t, 1, created.Tentativas)
	require.NotNil(t, created.BloqueioExpiraEm)

	duplicate, claimed, err := store.TentarIniciar(ctx, input)
	require.NoError(t, err)
	require.False(t, claimed)
	require.Nil(t, duplicate)

	failed, err := store.Falhar(ctx, created.ID, "temporary allocation failure")
	require.NoError(t, err)
	require.Equal(t, viagens.StatusExecucaoFalhou, failed.Status)
	require.Equal(t, "temporary allocation failure", *failed.UltimoErro)

	retryInput := input
	retryInput.Agora = agora.Add(time.Minute)
	retryInput.BloqueioExpiraEm = agora.Add(3 * time.Minute)
	retried, claimed, err := store.TentarIniciar(ctx, retryInput)
	require.NoError(t, err)
	require.True(t, claimed)
	require.Equal(t, created.ID, retried.ID)
	require.Equal(t, 2, retried.Tentativas)
	require.Nil(t, retried.UltimoErro)

	completed, err := store.Finalizar(ctx, retried.ID, viagens.StatusExecucaoConcluido)
	require.NoError(t, err)
	require.Equal(t, viagens.StatusExecucaoConcluido, completed.Status)
	require.NotNil(t, completed.FinalizadoEm)

	afterCompletion, claimed, err := store.TentarIniciar(ctx, retryInput)
	require.NoError(t, err)
	require.False(t, claimed)
	require.Nil(t, afterCompletion)

	stored, err := store.GetByChave(ctx, input.Chave)
	require.NoError(t, err)
	require.Equal(t, viagens.StatusExecucaoConcluido, stored.Status)
	require.Equal(t, 2, stored.Tentativas)

	_, err = store.Finalizar(ctx, completed.ID, viagens.StatusExecucaoSemDemanda)
	require.ErrorIs(t, err, viagens.ErrExecucaoNaoProcessando)
	_, err = store.Falhar(ctx, 999999, "not found")
	require.ErrorIs(t, err, brerror.ErrNotFound)
}

func TestExecucaoPlanejamentoRepository_ReclaimsExpiredProcessing(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := viagens.NewExecucaoPlanejamentoStore(tx)
	agora := time.Date(2026, time.August, 12, 21, 30, 0, 0, time.FixedZone("America/Maceio", -3*60*60))
	input := viagens.IniciarExecucaoPlanejamentoInput{
		Chave: viagens.ChaveExecucaoPlanejamento{
			DataViagem:         agora,
			Turno:              viagens.TurnoNoturno,
			MunicipioDestinoID: testMunicipioID,
			RotaInternaID:      fixture.RotaInternaID,
			Sentido:            viagens.SentidoVolta,
		},
		PartidaEm:        agora.Add(30 * time.Minute),
		FechamentoEm:     agora,
		Agora:            agora,
		BloqueioExpiraEm: agora.Add(time.Minute),
	}

	first, claimed, err := store.TentarIniciar(ctx, input)
	require.NoError(t, err)
	require.True(t, claimed)

	input.Agora = agora.Add(2 * time.Minute)
	input.BloqueioExpiraEm = agora.Add(4 * time.Minute)
	reclaimed, claimed, err := store.TentarIniciar(ctx, input)
	require.NoError(t, err)
	require.True(t, claimed)
	require.Equal(t, first.ID, reclaimed.ID)
	require.Equal(t, 2, reclaimed.Tentativas)

	withoutDemand, err := store.Finalizar(ctx, reclaimed.ID, viagens.StatusExecucaoSemDemanda)
	require.NoError(t, err)
	require.Equal(t, viagens.StatusExecucaoSemDemanda, withoutDemand.Status)
}
