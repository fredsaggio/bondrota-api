-- +goose Up
-- +goose StatementBegin

-- O telefone identifica uma pessoa no sistema, independentemente de ela estar
-- cadastrada como cliente ou motorista. A tabela auxiliar permite que o banco
-- garanta essa unicidade entre as duas tabelas, inclusive sob concorrencia.
-- Telefones vazios representam "nao informado" e nao entram no registro.
CREATE TABLE telefones_cadastrados (
    telefone TEXT PRIMARY KEY CHECK (btrim(telefone) <> ''),
    entidade TEXT NOT NULL CHECK (entidade IN ('clientes', 'motoristas')),
    entidade_id BIGINT NOT NULL,
    UNIQUE (entidade, entidade_id)
);

INSERT INTO telefones_cadastrados (telefone, entidade, entidade_id)
SELECT telefone, 'clientes', id FROM clientes WHERE telefone <> ''
UNION ALL
SELECT telefone, 'motoristas', id FROM motoristas WHERE telefone <> '';

CREATE FUNCTION sincronizar_telefone_cadastrado()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.telefone <> '' THEN
            DELETE FROM telefones_cadastrados
            WHERE entidade = TG_TABLE_NAME AND entidade_id = OLD.id;
        END IF;
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE' AND OLD.telefone IS DISTINCT FROM NEW.telefone AND OLD.telefone <> '' THEN
        DELETE FROM telefones_cadastrados
        WHERE entidade = TG_TABLE_NAME AND entidade_id = OLD.id;
    END IF;

    IF NEW.telefone <> '' AND (TG_OP = 'INSERT' OR OLD.telefone IS DISTINCT FROM NEW.telefone) THEN
        INSERT INTO telefones_cadastrados (telefone, entidade, entidade_id)
        VALUES (NEW.telefone, TG_TABLE_NAME, NEW.id);
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sincronizar_telefone_motoristas
    AFTER INSERT OR UPDATE OF telefone OR DELETE ON motoristas
    FOR EACH ROW EXECUTE FUNCTION sincronizar_telefone_cadastrado();

CREATE TRIGGER sincronizar_telefone_clientes
    AFTER INSERT OR UPDATE OF telefone OR DELETE ON clientes
    FOR EACH ROW EXECUTE FUNCTION sincronizar_telefone_cadastrado();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS sincronizar_telefone_clientes ON clientes;
DROP TRIGGER IF EXISTS sincronizar_telefone_motoristas ON motoristas;
DROP FUNCTION IF EXISTS sincronizar_telefone_cadastrado();
DROP TABLE IF EXISTS telefones_cadastrados;

-- +goose StatementEnd
