-- +goose Up
-- +goose StatementBegin

CREATE TABLE rotas_dinamicas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    viagem_id BIGINT NOT NULL UNIQUE REFERENCES viagens(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    origem_nome TEXT NOT NULL,
    origem_latitude NUMERIC(9,6) NOT NULL,
    origem_longitude NUMERIC(9,6) NOT NULL,
    destino_final_nome TEXT NOT NULL,
    destino_final_latitude NUMERIC(9,6) NOT NULL,
    destino_final_longitude NUMERIC(9,6) NOT NULL,
    distancia_metros INTEGER NOT NULL,
    duracao_segundos INTEGER NOT NULL,
    geometry JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rotas_dinamicas_distancia CHECK (distancia_metros >= 0),
    CONSTRAINT chk_rotas_dinamicas_duracao CHECK (duracao_segundos >= 0),
    CONSTRAINT chk_rotas_dinamicas_provider CHECK (provider <> '')
);

CREATE TRIGGER set_updated_at_rotas_dinamicas
    BEFORE UPDATE ON rotas_dinamicas
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_rotas_dinamicas_expires_at ON rotas_dinamicas(expires_at);

CREATE TABLE rota_dinamica_destinos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rota_dinamica_id BIGINT NOT NULL REFERENCES rotas_dinamicas(id) ON DELETE CASCADE,
    destino_id BIGINT NOT NULL REFERENCES destinos(id) ON DELETE RESTRICT,
    ordem SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rota_dinamica_destinos_ordem CHECK (ordem > 0),
    CONSTRAINT uq_rota_dinamica_destino UNIQUE (rota_dinamica_id, destino_id),
    CONSTRAINT uq_rota_dinamica_ordem UNIQUE (rota_dinamica_id, ordem)
);

CREATE INDEX idx_rota_dinamica_destinos_destino ON rota_dinamica_destinos(destino_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE rota_dinamica_destinos;
DROP TRIGGER IF EXISTS set_updated_at_rotas_dinamicas ON rotas_dinamicas;
DROP TABLE rotas_dinamicas;

-- +goose StatementEnd
