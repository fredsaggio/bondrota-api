package retencao

import (
	"context"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type store struct {
	db db.DB
}

func NewStore(database db.DB) Store {
	return &store{db: database}
}

func (s *store) RemoverCiclosAntigos(ctx context.Context, corte time.Time, limite int) (int64, error) {
	const op = "db/retencaoStore.RemoverCiclosAntigos"

	// O IN com subconsulta limitada mantem o lote pequeno; sem isso um trimestre
	// inteiro sairia numa unica transacao.
	const q = `
		DELETE FROM ciclos_viagem
		WHERE id IN (
			SELECT id
			FROM ciclos_viagem
			WHERE data_viagem < @corte
			ORDER BY data_viagem
			LIMIT @limite
		)
	`

	return s.exec(ctx, op, q, corte, limite)
}

func (s *store) RemoverReservasAntigas(ctx context.Context, corte time.Time, limite int) (int64, error) {
	const op = "db/retencaoStore.RemoverReservasAntigas"

	// O NOT EXISTS pula reservas ainda presas a uma viagem (FK RESTRICT). Sem ele
	// a limpeza falharia sempre que o lote de ciclos nao tivesse alcancado a viagem
	// correspondente; assim a reserva apenas espera a proxima execucao.
	const q = `
		DELETE FROM reservas
		WHERE id IN (
			SELECT r.id
			FROM reservas r
			WHERE r.data_viagem < @corte
			  AND NOT EXISTS (
				SELECT 1 FROM viagem_reservas vr WHERE vr.reserva_id = r.id
			  )
			ORDER BY r.data_viagem
			LIMIT @limite
		)
	`

	return s.exec(ctx, op, q, corte, limite)
}

func (s *store) RemoverExecucoesAntigas(ctx context.Context, corte time.Time, limite int) (int64, error) {
	const op = "db/retencaoStore.RemoverExecucoesAntigas"

	const q = `
		DELETE FROM execucoes_planejamento
		WHERE id IN (
			SELECT id
			FROM execucoes_planejamento
			WHERE data_viagem < @corte
			ORDER BY data_viagem
			LIMIT @limite
		)
	`

	return s.exec(ctx, op, q, corte, limite)
}

func (s *store) exec(ctx context.Context, op, query string, corte time.Time, limite int) (int64, error) {
	cmdTag, err := s.db.Exec(ctx, query, pgx.StrictNamedArgs{"corte": corte, "limite": limite})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return cmdTag.RowsAffected(), nil
}
