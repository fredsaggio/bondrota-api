// Package retencao apaga os dados operacionais que envelhecem: ciclos de viagem,
// reservas e execucoes de planejamento. Essas tabelas crescem a cada dia de
// operacao e nunca sao limpas pelo fluxo normal do sistema.
package retencao

import (
	"context"
	"time"
)

// MesesPadrao e a janela mantida quando RETENTION_MONTHS nao e configurado. Ela
// existe para permitir auditoria externa dos ultimos tres meses de operacao.
const MesesPadrao = 3

// LoteMaximoPadrao limita quantas linhas cada tabela perde por execucao. O cron do
// Supabase encerra a chamada HTTP em 55s, entao apagar um trimestre inteiro de uma
// vez estouraria o tempo e ainda seguraria lock na tabela.
const LoteMaximoPadrao = 5000

type Config struct {
	Meses      int
	LoteMaximo int
	Location   *time.Location
}

// ResumoLimpeza descreve o resultado de uma execucao.
type ResumoLimpeza struct {
	Corte              time.Time
	CiclosRemovidos    int64
	ReservasRemovidas  int64
	ExecucoesRemovidas int64
	// LoteSaturado indica que alguma tabela atingiu o limite do lote, ou seja,
	// ainda sobrou trabalho para a proxima execucao.
	LoteSaturado bool
}

// Store apaga cada grupo de registros vencidos. A ordem importa: viagem_reservas
// referencia reservas com ON DELETE RESTRICT, entao os ciclos (que derrubam as
// viagens e, em cascata, as viagem_reservas) precisam sair antes das reservas.
type Store interface {
	// RemoverCiclosAntigos apaga ciclos com data_viagem anterior ao corte. O
	// cascata leva junto viagens, viagem_reservas, viagem_reserva_confirmacoes,
	// viagem_horarios, viagem_localizacoes, rotas_dinamicas e rota_dinamica_destinos.
	RemoverCiclosAntigos(ctx context.Context, corte time.Time, limite int) (int64, error)
	// RemoverReservasAntigas apaga reservas anteriores ao corte que ja nao estao
	// associadas a nenhuma viagem.
	RemoverReservasAntigas(ctx context.Context, corte time.Time, limite int) (int64, error)
	RemoverExecucoesAntigas(ctx context.Context, corte time.Time, limite int) (int64, error)
}

type Service interface {
	Limpar(ctx context.Context) (ResumoLimpeza, error)
}
