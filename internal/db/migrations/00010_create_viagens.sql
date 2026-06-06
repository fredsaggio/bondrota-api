-- +goose Up
-- +goose StatementBegin

CREATE TYPE status_ciclo_viagem AS ENUM ('planejado', 'em_andamento', 'concluido', 'cancelado');
CREATE TYPE sentido_viagem AS ENUM ('ida', 'volta');
CREATE TYPE status_viagem AS ENUM ('programada', 'em_andamento', 'concluida', 'cancelada');
CREATE TYPE status_presenca_viagem AS ENUM ('aguardando', 'embarcou', 'faltou', 'cancelado');

CREATE TABLE ciclos_viagem (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    data_viagem DATE NOT NULL,
    turno turno_cliente NOT NULL,
    cidade TEXT NOT NULL,
    rota_interna_id BIGINT NOT NULL REFERENCES rotas_internas(id) ON DELETE RESTRICT,
    veiculo_id BIGINT NOT NULL REFERENCES veiculos(id) ON DELETE RESTRICT,
    motorista_id BIGINT NOT NULL REFERENCES motoristas(id) ON DELETE RESTRICT,
    status status_ciclo_viagem NOT NULL DEFAULT 'planejado',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_ciclos_viagem_turno_operacional CHECK (turno IN ('MT', 'VT', 'NT'))
);

CREATE TRIGGER set_updated_at_ciclos_viagem
    BEFORE UPDATE ON ciclos_viagem
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE UNIQUE INDEX uq_ciclos_viagem_ativos_veiculo_data_turno
    ON ciclos_viagem (veiculo_id, data_viagem, turno)
    WHERE status <> 'cancelado';

CREATE UNIQUE INDEX uq_ciclos_viagem_ativos_motorista_data_turno
    ON ciclos_viagem (motorista_id, data_viagem, turno)
    WHERE status <> 'cancelado';

CREATE INDEX idx_ciclos_viagem_planejamento
    ON ciclos_viagem (data_viagem, turno, cidade, rota_interna_id, status);

CREATE INDEX idx_ciclos_viagem_expires_at
    ON ciclos_viagem (expires_at);

CREATE TABLE viagens (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ciclo_viagem_id BIGINT NOT NULL REFERENCES ciclos_viagem(id) ON DELETE CASCADE,
    sentido sentido_viagem NOT NULL,
    status status_viagem NOT NULL DEFAULT 'programada',
    partida_prevista TIMESTAMPTZ,
    inicio_real TIMESTAMPTZ,
    fim_real TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_viagens_ciclo_sentido UNIQUE (ciclo_viagem_id, sentido),
    CONSTRAINT chk_viagens_inicio_fim CHECK (
        inicio_real IS NULL
        OR fim_real IS NULL
        OR fim_real >= inicio_real
    )
);

CREATE TRIGGER set_updated_at_viagens
    BEFORE UPDATE ON viagens
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_viagens_ciclo ON viagens(ciclo_viagem_id);
CREATE INDEX idx_viagens_status ON viagens(status);

CREATE TABLE viagem_reservas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    viagem_id BIGINT NOT NULL REFERENCES viagens(id) ON DELETE CASCADE,
    reserva_id BIGINT NOT NULL REFERENCES reservas(id) ON DELETE RESTRICT,
    status_presenca status_presenca_viagem NOT NULL DEFAULT 'aguardando',
    horario_confirmacao TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_viagem_reservas_viagem_reserva UNIQUE (viagem_id, reserva_id)
);

CREATE TRIGGER set_updated_at_viagem_reservas
    BEFORE UPDATE ON viagem_reservas
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE UNIQUE INDEX uq_viagem_reservas_reserva_ativa
    ON viagem_reservas (reserva_id)
    WHERE status_presenca <> 'cancelado';

CREATE INDEX idx_viagem_reservas_viagem ON viagem_reservas(viagem_id);
CREATE INDEX idx_viagem_reservas_reserva ON viagem_reservas(reserva_id);
CREATE INDEX idx_viagem_reservas_status ON viagem_reservas(status_presenca);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS set_updated_at_viagem_reservas ON viagem_reservas;
DROP TRIGGER IF EXISTS set_updated_at_viagens ON viagens;
DROP TRIGGER IF EXISTS set_updated_at_ciclos_viagem ON ciclos_viagem;

DROP TABLE viagem_reservas;
DROP TABLE viagens;
DROP TABLE ciclos_viagem;

DROP TYPE status_presenca_viagem;
DROP TYPE status_viagem;
DROP TYPE sentido_viagem;
DROP TYPE status_ciclo_viagem;

-- +goose StatementEnd
