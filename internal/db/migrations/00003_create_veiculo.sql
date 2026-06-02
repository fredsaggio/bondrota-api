-- +goose Up
-- +goose StatementBegin

CREATE TYPE status_veiculo AS ENUM ('ativo', 'inativo', 'manutencao');

CREATE TABLE veiculos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    placa TEXT UNIQUE NOT NULL,
    modelo TEXT NOT NULL,
    capacidade SMALLINT NOT NULL,
    cidade_base TEXT NOT NULL,
    status status_veiculo NOT NULL DEFAULT 'ativo',
    ar_condicionado BOOLEAN NOT NULL DEFAULT FALSE,
    banheiro BOOLEAN NOT NULL DEFAULT FALSE,
    persiana BOOLEAN NOT NULL DEFAULT FALSE,
    luz_leitura BOOLEAN NOT NULL DEFAULT FALSE,
    tomada BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_capacidade CHECK (capacidade > 0)
);

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON veiculos
FOR EACH ROW
EXECUTE FUNCTION trigger_set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER set_updated_at ON veiculos;
DROP TABLE veiculos;
DROP TYPE status_veiculo;

-- +goose StatementEnd