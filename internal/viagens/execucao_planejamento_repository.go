package viagens

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type execucaoPlanejamentoStore struct {
	db db.DB
}

func NewExecucaoPlanejamentoStore(database db.DB) ExecucaoPlanejamentoRepository {
	return &execucaoPlanejamentoStore{db: database}
}

func (s *execucaoPlanejamentoStore) TentarIniciar(ctx context.Context, input IniciarExecucaoPlanejamentoInput) (*ExecucaoPlanejamento, bool, error) {
	const op = "db/execucaoPlanejamentoStore.TentarIniciar"

	const q = `
		INSERT INTO execucoes_planejamento (
			data_viagem,
			turno,
			municipio_destino_id,
			rota_interna_id,
			sentido,
			partida_em,
			fechamento_em,
			bloqueio_expira_em,
			iniciado_em
		)
		VALUES (
			@data_viagem,
			@turno,
			@municipio_destino_id,
			@rota_interna_id,
			@sentido,
			@partida_em,
			@fechamento_em,
			@bloqueio_expira_em,
			@agora
		)
		ON CONFLICT ON CONSTRAINT uq_execucoes_planejamento_chave
		DO UPDATE SET
			partida_em = EXCLUDED.partida_em,
			fechamento_em = EXCLUDED.fechamento_em,
			status = 'processando',
			tentativas = execucoes_planejamento.tentativas + 1,
			ultimo_erro = NULL,
			bloqueio_expira_em = EXCLUDED.bloqueio_expira_em,
			proxima_tentativa_em = NULL,
			iniciado_em = EXCLUDED.iniciado_em,
			finalizado_em = NULL
		WHERE (
				execucoes_planejamento.status = 'falhou'
				AND COALESCE(execucoes_planejamento.proxima_tentativa_em, @agora) <= @agora
			)
			OR (
				execucoes_planejamento.status = 'processando'
				AND execucoes_planejamento.bloqueio_expira_em <= @agora
			)
		RETURNING
			id,
			data_viagem,
			turno,
			municipio_destino_id,
			rota_interna_id,
			sentido,
			partida_em,
			fechamento_em,
			status,
			tentativas,
			ultimo_erro,
			bloqueio_expira_em,
			proxima_tentativa_em,
			iniciado_em,
			finalizado_em,
			created_at,
			updated_at
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":          input.Chave.DataViagem,
		"turno":                input.Chave.Turno,
		"municipio_destino_id": input.Chave.MunicipioDestinoID,
		"rota_interna_id":      input.Chave.RotaInternaID,
		"sentido":              input.Chave.Sentido,
		"partida_em":           input.PartidaEm,
		"fechamento_em":        input.FechamentoEm,
		"agora":                input.Agora,
		"bloqueio_expira_em":   input.BloqueioExpiraEm,
	})
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", op, err)
	}

	execucao, err := pgx.CollectExactlyOneRow(rows, scanExecucaoPlanejamento)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", op, err)
	}

	return &execucao, true, nil
}

func (s *execucaoPlanejamentoStore) GetByChave(ctx context.Context, chave ChaveExecucaoPlanejamento) (*ExecucaoPlanejamento, error) {
	const op = "db/execucaoPlanejamentoStore.GetByChave"

	const q = `
		SELECT
			id,
			data_viagem,
			turno,
			municipio_destino_id,
			rota_interna_id,
			sentido,
			partida_em,
			fechamento_em,
			status,
			tentativas,
			ultimo_erro,
			bloqueio_expira_em,
			proxima_tentativa_em,
			iniciado_em,
			finalizado_em,
			created_at,
			updated_at
		FROM execucoes_planejamento
		WHERE data_viagem = @data_viagem
			AND turno = @turno
			AND municipio_destino_id = @municipio_destino_id
			AND rota_interna_id = @rota_interna_id
			AND sentido = @sentido
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_viagem":          chave.DataViagem,
		"turno":                chave.Turno,
		"municipio_destino_id": chave.MunicipioDestinoID,
		"rota_interna_id":      chave.RotaInternaID,
		"sentido":              chave.Sentido,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	execucao, err := pgx.CollectExactlyOneRow(rows, scanExecucaoPlanejamento)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, brerror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &execucao, nil
}

func (s *execucaoPlanejamentoStore) ListFalhas(ctx context.Context, limit int) ([]ExecucaoPlanejamento, error) {
	const op = "db/execucaoPlanejamentoStore.ListFalhas"
	if limit <= 0 {
		limit = 50
	}

	const q = `
		SELECT
			id,
			data_viagem,
			turno,
			municipio_destino_id,
			rota_interna_id,
			sentido,
			partida_em,
			fechamento_em,
			status,
			tentativas,
			ultimo_erro,
			bloqueio_expira_em,
			proxima_tentativa_em,
			iniciado_em,
			finalizado_em,
			created_at,
			updated_at
		FROM execucoes_planejamento
		WHERE status = 'falhou'
		ORDER BY proxima_tentativa_em, id
		LIMIT @limit
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"limit": limit})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	execucoes, err := pgx.CollectRows(rows, scanExecucaoPlanejamento)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if execucoes == nil {
		return []ExecucaoPlanejamento{}, nil
	}
	return execucoes, nil
}

func (s *execucaoPlanejamentoStore) Finalizar(ctx context.Context, execucaoID int64, resultado StatusExecucaoPlanejamento) (*ExecucaoPlanejamento, error) {
	const op = "db/execucaoPlanejamentoStore.Finalizar"

	if resultado != StatusExecucaoConcluido && resultado != StatusExecucaoSemDemanda {
		return nil, ErrResultadoInvalido
	}

	const q = `
		UPDATE execucoes_planejamento
		SET status = @status,
			ultimo_erro = NULL,
			bloqueio_expira_em = NULL,
			proxima_tentativa_em = NULL,
			finalizado_em = NOW()
		WHERE id = @id
			AND status = 'processando'
		RETURNING
			id,
			data_viagem,
			turno,
			municipio_destino_id,
			rota_interna_id,
			sentido,
			partida_em,
			fechamento_em,
			status,
			tentativas,
			ultimo_erro,
			bloqueio_expira_em,
			proxima_tentativa_em,
			iniciado_em,
			finalizado_em,
			created_at,
			updated_at
	`

	execucao, err := s.updateExecucao(ctx, q, pgx.StrictNamedArgs{
		"id":     execucaoID,
		"status": resultado,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return execucao, nil
}

func (s *execucaoPlanejamentoStore) Falhar(ctx context.Context, input FalharExecucaoPlanejamentoInput) (*ExecucaoPlanejamento, error) {
	const op = "db/execucaoPlanejamentoStore.Falhar"

	mensagem := strings.TrimSpace(input.Mensagem)
	if mensagem == "" {
		return nil, ErrUltimoErroObrigatorio
	}
	if input.FalhouEm.IsZero() || !input.ProximaTentativaEm.After(input.FalhouEm) {
		return nil, ErrProximaTentativaInvalida
	}

	const q = `
		UPDATE execucoes_planejamento
		SET status = 'falhou',
			ultimo_erro = @ultimo_erro,
			bloqueio_expira_em = NULL,
			proxima_tentativa_em = @proxima_tentativa_em,
			finalizado_em = @falhou_em
		WHERE id = @id
			AND status = 'processando'
		RETURNING
			id,
			data_viagem,
			turno,
			municipio_destino_id,
			rota_interna_id,
			sentido,
			partida_em,
			fechamento_em,
			status,
			tentativas,
			ultimo_erro,
			bloqueio_expira_em,
			proxima_tentativa_em,
			iniciado_em,
			finalizado_em,
			created_at,
			updated_at
	`

	execucao, err := s.updateExecucao(ctx, q, pgx.StrictNamedArgs{
		"id":                   input.ExecucaoID,
		"ultimo_erro":          mensagem,
		"falhou_em":            input.FalhouEm,
		"proxima_tentativa_em": input.ProximaTentativaEm,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return execucao, nil
}

func (s *execucaoPlanejamentoStore) updateExecucao(ctx context.Context, query string, args pgx.StrictNamedArgs) (*ExecucaoPlanejamento, error) {
	rows, err := s.db.Query(ctx, query, args)
	if err != nil {
		return nil, err
	}

	execucao, err := pgx.CollectExactlyOneRow(rows, scanExecucaoPlanejamento)
	if !errors.Is(err, pgx.ErrNoRows) {
		if err != nil {
			return nil, err
		}
		return &execucao, nil
	}

	var exists bool
	if err := s.db.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM execucoes_planejamento WHERE id = @id)`,
		pgx.StrictNamedArgs{"id": args["id"]},
	).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, brerror.ErrNotFound
	}
	return nil, ErrExecucaoNaoProcessando
}

func scanExecucaoPlanejamento(row pgx.CollectableRow) (ExecucaoPlanejamento, error) {
	var execucao ExecucaoPlanejamento
	err := row.Scan(
		&execucao.ID,
		&execucao.DataViagem,
		&execucao.Turno,
		&execucao.MunicipioDestinoID,
		&execucao.RotaInternaID,
		&execucao.Sentido,
		&execucao.PartidaEm,
		&execucao.FechamentoEm,
		&execucao.Status,
		&execucao.Tentativas,
		&execucao.UltimoErro,
		&execucao.BloqueioExpiraEm,
		&execucao.ProximaTentativaEm,
		&execucao.IniciadoEm,
		&execucao.FinalizadoEm,
		&execucao.CreatedAt,
		&execucao.UpdatedAt,
	)
	return execucao, err
}
