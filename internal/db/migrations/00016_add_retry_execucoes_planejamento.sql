-- +goose Up
-- +goose StatementBegin

ALTER TABLE execucoes_planejamento
    ADD COLUMN proxima_tentativa_em TIMESTAMPTZ;

UPDATE execucoes_planejamento
SET proxima_tentativa_em = GREATEST(NOW(), finalizado_em + INTERVAL '1 minute')
WHERE status = 'falhou';

ALTER TABLE execucoes_planejamento
    DROP CONSTRAINT chk_execucoes_planejamento_estado,
    ADD CONSTRAINT chk_execucoes_planejamento_estado CHECK (
        (
            status = 'processando'
            AND bloqueio_expira_em IS NOT NULL
            AND bloqueio_expira_em > iniciado_em
            AND proxima_tentativa_em IS NULL
            AND finalizado_em IS NULL
            AND ultimo_erro IS NULL
        )
        OR (
            status IN ('concluido', 'sem_demanda')
            AND bloqueio_expira_em IS NULL
            AND proxima_tentativa_em IS NULL
            AND finalizado_em IS NOT NULL
            AND ultimo_erro IS NULL
        )
        OR (
            status = 'falhou'
            AND bloqueio_expira_em IS NULL
            AND finalizado_em IS NOT NULL
            AND (
                proxima_tentativa_em IS NULL
                OR proxima_tentativa_em > finalizado_em
            )
            AND ultimo_erro IS NOT NULL
            AND ultimo_erro <> ''
        )
    );

CREATE INDEX idx_execucoes_planejamento_retry
    ON execucoes_planejamento (proxima_tentativa_em)
    WHERE status = 'falhou';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX idx_execucoes_planejamento_retry;

ALTER TABLE execucoes_planejamento
    DROP CONSTRAINT chk_execucoes_planejamento_estado,
    ADD CONSTRAINT chk_execucoes_planejamento_estado CHECK (
        (
            status = 'processando'
            AND bloqueio_expira_em IS NOT NULL
            AND bloqueio_expira_em > iniciado_em
            AND finalizado_em IS NULL
            AND ultimo_erro IS NULL
        )
        OR (
            status IN ('concluido', 'sem_demanda')
            AND bloqueio_expira_em IS NULL
            AND finalizado_em IS NOT NULL
            AND ultimo_erro IS NULL
        )
        OR (
            status = 'falhou'
            AND bloqueio_expira_em IS NULL
            AND finalizado_em IS NOT NULL
            AND ultimo_erro IS NOT NULL
            AND ultimo_erro <> ''
        )
    ),
    DROP COLUMN proxima_tentativa_em;

-- +goose StatementEnd
