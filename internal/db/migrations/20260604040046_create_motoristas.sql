-- +goose Up
-- +goose StatementBegin
CREATE TYPE turno_motorista AS ENUM ('MT', 'VT', 'NT');

CREATE TABLE motoristas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nome TEXT NOT NULL,
    cpf TEXT NOT NULL UNIQUE,
    senha TEXT NOT NULL,
    telefone TEXT NOT NULL DEFAULT '',
    data_nasc DATE NOT NULL,
    turno turno_motorista NOT NULL,
    cidade_trabalho TEXT NOT NULL DEFAULT '',
    residencia TEXT NOT NULL DEFAULT '',
    foto TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at_motoristas
    BEFORE UPDATE ON motoristas
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_motoristas_cpf ON motoristas(cpf);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_updated_at_motoristas ON motoristas;
DROP TABLE motoristas;
DROP TYPE turno_motorista;
-- +goose StatementEnd