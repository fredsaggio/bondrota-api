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

	proximaTentativa := agora.Add(time.Minute)
	failed, err := store.Falhar(ctx, viagens.FalharExecucaoPlanejamentoInput{
		ExecucaoID:         created.ID,
		Mensagem:           "temporary allocation failure",
		FalhouEm:           agora,
		ProximaTentativaEm: proximaTentativa,
	})
	require.NoError(t, err)
	require.Equal(t, viagens.StatusExecucaoFalhou, failed.Status)
	require.Equal(t, "temporary allocation failure", *failed.UltimoErro)
	require.True(t, proximaTentativa.Equal(*failed.ProximaTentativaEm))
	falhas, err := store.ListFalhas(ctx, 10)
	require.NoError(t, err)
	require.Len(t, falhas, 1)
	require.Equal(t, created.ID, falhas[0].ID)

	beforeRetry := input
	beforeRetry.Agora = agora.Add(30 * time.Second)
	beforeRetry.BloqueioExpiraEm = agora.Add(3 * time.Minute)
	notRetried, claimed, err := store.TentarIniciar(ctx, beforeRetry)
	require.NoError(t, err)
	require.False(t, claimed)
	require.Nil(t, notRetried)

	retryInput := input
	retryInput.Agora = agora.Add(time.Minute)
	retryInput.BloqueioExpiraEm = agora.Add(3 * time.Minute)
	retried, claimed, err := store.TentarIniciar(ctx, retryInput)
	require.NoError(t, err)
	require.True(t, claimed)
	require.Equal(t, created.ID, retried.ID)
	require.Equal(t, 2, retried.Tentativas)
	require.Nil(t, retried.UltimoErro)
	require.Nil(t, retried.ProximaTentativaEm)

	completed, err := store.Finalizar(ctx, retried.ID, viagens.StatusExecucaoConcluido)
	require.NoError(t, err)
	require.Equal(t, viagens.StatusExecucaoConcluido, completed.Status)
	require.NotNil(t, completed.FinalizadoEm)
	falhas, err = store.ListFalhas(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, falhas)

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
	_, err = store.Falhar(ctx, viagens.FalharExecucaoPlanejamentoInput{
		ExecucaoID:         999999,
		Mensagem:           "not found",
		FalhouEm:           agora,
		ProximaTentativaEm: agora.Add(time.Minute),
	})
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
