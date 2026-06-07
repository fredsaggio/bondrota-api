-- +goose Up
-- +goose StatementBegin

CREATE TABLE viagem_localizacoes (
    viagem_id BIGINT PRIMARY KEY REFERENCES viagens(id) ON DELETE CASCADE,
    motorista_id BIGINT NOT NULL REFERENCES motoristas(id) ON DELETE RESTRICT,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    velocidade_kmh DOUBLE PRECISION NOT NULL DEFAULT 0,
    direcao_graus DOUBLE PRECISION NOT NULL DEFAULT 0,
    precisao_metros DOUBLE PRECISION NOT NULL DEFAULT 0,
    registrada_em TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_viagem_localizacoes_latitude CHECK (latitude >= -90 AND latitude <= 90),
    CONSTRAINT chk_viagem_localizacoes_longitude CHECK (longitude >= -180 AND longitude <= 180),
    CONSTRAINT chk_viagem_localizacoes_velocidade CHECK (velocidade_kmh >= 0),
    CONSTRAINT chk_viagem_localizacoes_direcao CHECK (direcao_graus >= 0 AND direcao_graus <= 360),
    CONSTRAINT chk_viagem_localizacoes_precisao CHECK (precisao_metros >= 0)
);

CREATE TRIGGER set_updated_at_viagem_localizacoes
    BEFORE UPDATE ON viagem_localizacoes
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_viagem_localizacoes_motorista ON viagem_localizacoes(motorista_id);
CREATE INDEX idx_viagem_localizacoes_registrada_em ON viagem_localizacoes(registrada_em);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS set_updated_at_viagem_localizacoes ON viagem_localizacoes;
DROP TABLE viagem_localizacoes;

-- +goose StatementEnd
