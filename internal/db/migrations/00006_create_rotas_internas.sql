-- +goose Up
-- +goose StatementBegin

CREATE TABLE rotas_internas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at_rotas_internas
    BEFORE UPDATE ON rotas_internas
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE rota_interna_paradas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rota_interna_id BIGINT NOT NULL REFERENCES rotas_internas(id) ON DELETE CASCADE,
    parada_id BIGINT NOT NULL REFERENCES paradas(id) ON DELETE RESTRICT,
    ordem INT NOT NULL,
    CONSTRAINT chk_ordem_positiva CHECK (ordem > 0)
);

CREATE INDEX idx_rota_interna_paradas_rota ON rota_interna_paradas(rota_interna_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS set_updated_at_rotas_internas ON rotas_internas;
DROP TABLE rota_interna_paradas;
DROP TABLE rotas_internas;

-- +goose StatementEnd
