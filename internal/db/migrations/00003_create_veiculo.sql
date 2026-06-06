-- +goose Up
-- +goose StatementBegin

CREATE TYPE status_veiculo AS ENUM ('ativo', 'inativo', 'manutencao');
CREATE TYPE categoria_veiculo AS ENUM ('executivo', 'escolar', 'carro_7_lugares');

CREATE TABLE veiculos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    placa TEXT UNIQUE NOT NULL,
    modelo TEXT NOT NULL,
    categoria categoria_veiculo NOT NULL,
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
    CONSTRAINT chk_capacidade CHECK (capacidade > 0),
    CONSTRAINT chk_veiculos_categoria_capacidade CHECK (
        (categoria = 'executivo' AND capacidade = 46)
        OR (categoria = 'escolar' AND capacidade = 24)
        OR (categoria = 'carro_7_lugares' AND capacidade = 7)
    )
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
DROP TYPE categoria_veiculo;
DROP TYPE status_veiculo;

-- +goose StatementEnd
