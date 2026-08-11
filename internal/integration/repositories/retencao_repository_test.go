//go:build integration

package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/retencao"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func contar(t *testing.T, ctx context.Context, tx pgx.Tx, tabela string) int {
	t.Helper()
	var total int
	require.NoError(t, tx.QueryRow(ctx, "SELECT count(*) FROM "+tabela).Scan(&total))
	return total
}

// A limpeza precisa remover, em cascata, tudo que pende de um ciclo antigo, e
// preservar integralmente o que ainda esta dentro da janela de retencao.
func TestRetencaoRepository_RemoveCascataAntigaEPreservaRecente(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := retencao.NewStore(tx)

	antiga := time.Date(2020, 1, 10, 0, 0, 0, 0, time.UTC)
	recente := time.Date(2030, 1, 10, 0, 0, 0, 0, time.UTC)
	corte := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Cenario antigo completo: ciclo -> viagem -> viagem_reserva -> reserva.
	cicloAntigo := seedCiclo(t, ctx, tx, fixture, antiga)
	viagemAntiga := seedViagem(t, ctx, tx, cicloAntigo, "ida")
	reservaAntiga := seedReserva(t, ctx, tx, fixture, "ida", antiga)
	seedViagemReserva(t, ctx, tx, viagemAntiga, reservaAntiga)
	_, err := tx.Exec(ctx, `
		INSERT INTO viagem_horarios (viagem_id, tipo, horario)
		VALUES ($1, 'partida_prevista', $2)`, viagemAntiga, antiga)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `
		INSERT INTO viagem_localizacoes (viagem_id, motorista_id, latitude, longitude, registrada_em)
		VALUES ($1, $2, -9.6, -35.7, $3)`, viagemAntiga, fixture.MotoristaID, antiga)
	require.NoError(t, err)

	// Cenario recente equivalente, que deve sobreviver inteiro.
	cicloRecente := seedCiclo(t, ctx, tx, fixture, recente)
	viagemRecente := seedViagem(t, ctx, tx, cicloRecente, "ida")
	reservaRecente := seedReserva(t, ctx, tx, fixture, "ida", recente)
	seedViagemReserva(t, ctx, tx, viagemRecente, reservaRecente)

	require.Equal(t, 2, contar(t, ctx, tx, "ciclos_viagem"))
	require.Equal(t, 2, contar(t, ctx, tx, "viagens"))
	require.Equal(t, 2, contar(t, ctx, tx, "viagem_reservas"))
	require.Equal(t, 2, contar(t, ctx, tx, "reservas"))

	// Passo 1: ciclos. O cascata precisa levar viagens, viagem_reservas,
	// viagem_horarios e viagem_localizacoes junto.
	ciclos, err := store.RemoverCiclosAntigos(ctx, corte, 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, ciclos)
	require.Equal(t, 1, contar(t, ctx, tx, "viagens"), "cascata deveria remover a viagem antiga")
	require.Equal(t, 1, contar(t, ctx, tx, "viagem_reservas"), "cascata deveria remover a viagem_reserva antiga")
	require.Equal(t, 0, contar(t, ctx, tx, "viagem_horarios"), "cascata deveria remover os horarios da viagem antiga")
	require.Equal(t, 0, contar(t, ctx, tx, "viagem_localizacoes"), "cascata deveria remover a localizacao da viagem antiga")

	// Passo 2: reservas, agora destravadas pela remocao das viagem_reservas.
	reservas, err := store.RemoverReservasAntigas(ctx, corte, 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, reservas)

	require.Equal(t, 1, contar(t, ctx, tx, "ciclos_viagem"))
	require.Equal(t, 1, contar(t, ctx, tx, "reservas"))

	var sobrevivente int64
	require.NoError(t, tx.QueryRow(ctx, `SELECT id FROM reservas`).Scan(&sobrevivente))
	require.Equal(t, reservaRecente, sobrevivente, "a reserva recente deveria sobreviver")
}

// Este e o caso que justifica o NOT EXISTS: se o lote de ciclos ainda nao alcancou
// a viagem que referencia a reserva, apagar a reserva violaria a FK RESTRICT. Em vez
// de falhar, a limpeza pula a reserva e a remove na proxima execucao.
func TestRetencaoRepository_NaoFalhaComReservaAindaPresaAViagem(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := retencao.NewStore(tx)

	antiga := time.Date(2020, 1, 10, 0, 0, 0, 0, time.UTC)
	corte := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	ciclo := seedCiclo(t, ctx, tx, fixture, antiga)
	viagem := seedViagem(t, ctx, tx, ciclo, "ida")
	reserva := seedReserva(t, ctx, tx, fixture, "ida", antiga)
	seedViagemReserva(t, ctx, tx, viagem, reserva)

	// Sem remover os ciclos antes: a reserva continua referenciada.
	removidas, err := store.RemoverReservasAntigas(ctx, corte, 100)
	require.NoError(t, err, "deveria pular a reserva presa, nao falhar por chave estrangeira")
	require.EqualValues(t, 0, removidas)
	require.Equal(t, 1, contar(t, ctx, tx, "reservas"))

	// Depois que o ciclo sai, a mesma chamada consegue remover.
	_, err = store.RemoverCiclosAntigos(ctx, corte, 100)
	require.NoError(t, err)
	removidas, err = store.RemoverReservasAntigas(ctx, corte, 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, removidas)
	require.Equal(t, 0, contar(t, ctx, tx, "reservas"))
}

func TestRetencaoRepository_RespeitaLimiteDoLote(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := retencao.NewStore(tx)

	corte := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for dia := 1; dia <= 5; dia++ {
		seedReserva(t, ctx, tx, fixture, "ida", time.Date(2020, 1, dia, 0, 0, 0, 0, time.UTC))
	}
	require.Equal(t, 5, contar(t, ctx, tx, "reservas"))

	removidas, err := store.RemoverReservasAntigas(ctx, corte, 2)
	require.NoError(t, err)
	require.EqualValues(t, 2, removidas, "o lote deveria limitar a remocao")
	require.Equal(t, 3, contar(t, ctx, tx, "reservas"))
}

func TestRetencaoRepository_RemoveExecucoesAntigas(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := retencao.NewStore(tx)

	corte := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	// Execucoes que envelheceram ja estao num estado terminal: 'concluido' exige
	// finalizado_em preenchido e bloqueio_expira_em nulo (chk_..._estado).
	inserirExecucao := func(data time.Time, sentido string) {
		_, err := tx.Exec(ctx, `
			INSERT INTO execucoes_planejamento (
				data_viagem, turno, municipio_destino_id, rota_interna_id, sentido,
				partida_em, fechamento_em, status, finalizado_em
			) VALUES ($1, 'NT', $2, $3, $4, $5, $6, 'concluido', $7)`,
			data, testMunicipioID, fixture.RotaInternaID, sentido,
			data.Add(20*time.Hour), data.Add(19*time.Hour), data.Add(21*time.Hour))
		require.NoError(t, err)
	}
	inserirExecucao(time.Date(2020, 1, 10, 0, 0, 0, 0, time.UTC), "ida")
	inserirExecucao(time.Date(2030, 1, 10, 0, 0, 0, 0, time.UTC), "ida")
	require.Equal(t, 2, contar(t, ctx, tx, "execucoes_planejamento"))

	removidas, err := store.RemoverExecucoesAntigas(ctx, corte, 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, removidas)
	require.Equal(t, 1, contar(t, ctx, tx, "execucoes_planejamento"))
}
