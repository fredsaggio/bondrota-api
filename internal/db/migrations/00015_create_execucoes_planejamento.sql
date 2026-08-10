-- +goose Up
-- +goose StatementBegin

CREATE TYPE status_execucao_planejamento AS ENUM (
    'processando',
    'concluido',
    'sem_demanda',
    'falhou'
);

CREATE TABLE execucoes_planejamento (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    data_viagem DATE NOT NULL,
    turno turno_cliente NOT NULL,
    municipio_destino_id BIGINT NOT NULL REFERENCES municipios(codigo_ibge) ON DELETE RESTRICT,
    rota_interna_id BIGINT NOT NULL REFERENCES rotas_internas(id) ON DELETE RESTRICT,
    sentido sentido_viagem NOT NULL,
    partida_em TIMESTAMPTZ NOT NULL,
    fechamento_em TIMESTAMPTZ NOT NULL,
    status status_execucao_planejamento NOT NULL DEFAULT 'processando',
    tentativas INTEGER NOT NULL DEFAULT 1,
    ultimo_erro TEXT,
    bloqueio_expira_em TIMESTAMPTZ,
    iniciado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finalizado_em TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_execucoes_planejamento_chave UNIQUE (
        data_viagem,
        turno,
        municipio_destino_id,
        rota_interna_id,
        sentido
    ),
    CONSTRAINT chk_execucoes_planejamento_turno CHECK (turno IN ('MT', 'VT', 'NT')),
    CONSTRAINT chk_execucoes_planejamento_janela CHECK (fechamento_em < partida_em),
    CONSTRAINT chk_execucoes_planejamento_tentativas CHECK (tentativas > 0),
    CONSTRAINT chk_execucoes_planejamento_estado CHECK (
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
    )
);

CREATE TRIGGER set_updated_at_execucoes_planejamento
    BEFORE UPDATE ON execucoes_planejamento
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_execucoes_planejamento_processamento
    ON execucoes_planejamento (status, bloqueio_expira_em);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS set_updated_at_execucoes_planejamento ON execucoes_planejamento;
DROP TABLE execucoes_planejamento;
DROP TYPE status_execucao_planejamento;

-- +goose StatementEnd
