-- +goose Up
-- +goose StatementBegin

CREATE TYPE reserva_sentido AS ENUM ('ida', 'volta');
CREATE TYPE status_reserva AS ENUM ('confirmada', 'cancelada');

CREATE TABLE reservas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cliente_id BIGINT NOT NULL REFERENCES clientes(id) ON DELETE CASCADE,
    vinculo_id BIGINT NOT NULL REFERENCES cliente_vinculos(id) ON DELETE RESTRICT,
    data_viagem DATE NOT NULL,
    turno turno_cliente NOT NULL,
    destino_id BIGINT NOT NULL REFERENCES destinos(id) ON DELETE RESTRICT,
    rota_interna_id BIGINT NOT NULL REFERENCES rotas_internas(id) ON DELETE RESTRICT,
    sentido reserva_sentido NOT NULL,
    status status_reserva NOT NULL DEFAULT 'confirmada',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_reservas_turno_operacional CHECK (turno IN ('MT', 'VT', 'NT'))
);

CREATE TRIGGER set_updated_at_reservas
    BEFORE UPDATE ON reservas
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE UNIQUE INDEX uq_reservas_ativas_vinculo_data_turno_sentido
    ON reservas (vinculo_id, data_viagem, turno, sentido)
    WHERE status <> 'cancelada';

CREATE INDEX idx_reservas_cliente ON reservas(cliente_id);
CREATE INDEX idx_reservas_planejamento
    ON reservas(data_viagem, turno, rota_interna_id, sentido, status, destino_id);
CREATE INDEX idx_reservas_vinculo ON reservas(vinculo_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS set_updated_at_reservas ON reservas;
DROP TABLE reservas;
DROP TYPE status_reserva;
DROP TYPE reserva_sentido;

-- +goose StatementEnd
