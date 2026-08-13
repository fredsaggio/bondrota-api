-- +goose Up
-- +goose StatementBegin
CREATE TYPE turno_cliente AS ENUM ('MT', 'VT', 'NT', 'IN');
CREATE TYPE tipo_conta AS ENUM ('estudante', 'estagio');

CREATE TABLE clientes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nome TEXT NOT NULL,
    cpf TEXT NOT NULL UNIQUE,
    senha TEXT NOT NULL,
    telefone TEXT NOT NULL DEFAULT '',
    data_nasc DATE NOT NULL,
    documento_identificacao TEXT NOT NULL CHECK (btrim(documento_identificacao) <> ''),
    comprovante_residencia TEXT NOT NULL CHECK (btrim(comprovante_residencia) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at_clientes
    BEFORE UPDATE ON clientes
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();




CREATE TABLE cliente_vinculos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cliente_id BIGINT NOT NULL REFERENCES clientes(id) ON DELETE CASCADE,
    tipo tipo_conta NOT NULL,
    turno turno_cliente NOT NULL,
    destino_id BIGINT NOT NULL REFERENCES destinos(id) ON DELETE RESTRICT,
    rota_interna_id BIGINT NOT NULL REFERENCES rotas_internas(id) ON DELETE RESTRICT,
    curso TEXT NOT NULL DEFAULT '',
    comprovante TEXT NOT NULL DEFAULT '',
    validade DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at_cliente_vinculos
    BEFORE UPDATE ON cliente_vinculos
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_cliente_vinculos_cliente ON cliente_vinculos(cliente_id);

CREATE TABLE horarios_fixos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    vinculo_id BIGINT NOT NULL REFERENCES cliente_vinculos(id) ON DELETE CASCADE,
    dia_semana SMALLINT NOT NULL,
    CONSTRAINT chk_dia_semana CHECK (dia_semana BETWEEN 1 AND 5),
    CONSTRAINT uq_vinculo_dia UNIQUE (vinculo_id, dia_semana)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_updated_at_cliente_vinculos ON cliente_vinculos;
DROP TRIGGER IF EXISTS set_updated_at_clientes ON clientes;
DROP TABLE horarios_fixos;
DROP TABLE cliente_vinculos;
DROP TABLE clientes;
DROP TYPE tipo_conta;
DROP TYPE turno_cliente;
-- +goose StatementEnd
