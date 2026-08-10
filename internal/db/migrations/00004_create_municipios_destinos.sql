-- +goose Up
-- +goose StatementBegin
CREATE TABLE municipios (
    codigo_ibge BIGINT PRIMARY KEY,
    nome TEXT NOT NULL,
    uf CHAR(2) NOT NULL,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_municipios_codigo_ibge_positivo CHECK (codigo_ibge > 0),
    CONSTRAINT chk_municipios_nome_not_blank CHECK (BTRIM(nome) <> ''),
    CONSTRAINT chk_municipios_uf_formato CHECK (uf ~ '^[A-Z]{2}$')
);

CREATE TRIGGER set_updated_at_municipios
    BEFORE UPDATE ON municipios
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_municipios_uf_nome ON municipios(uf, nome);

CREATE TABLE destinos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nome TEXT NOT NULL,
    rua TEXT NOT NULL,
    municipio_id BIGINT NOT NULL REFERENCES municipios(codigo_ibge) ON DELETE RESTRICT,
    latitude NUMERIC(9,6) NOT NULL,
    longitude NUMERIC(9,6) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at_destinos
    BEFORE UPDATE ON destinos
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_destinos_municipio ON destinos(municipio_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_updated_at_destinos ON destinos;
DROP TABLE destinos;
DROP TRIGGER IF EXISTS set_updated_at_municipios ON municipios;
DROP TABLE municipios;
-- +goose StatementEnd
