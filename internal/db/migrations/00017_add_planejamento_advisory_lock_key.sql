-- +goose Up
-- +goose StatementBegin

CREATE FUNCTION planejamento_advisory_lock_key(
    data_viagem DATE,
    turno TEXT,
    municipio_destino_id BIGINT,
    rota_interna_id BIGINT,
    sentido TEXT
)
RETURNS BIGINT
LANGUAGE SQL
IMMUTABLE
STRICT
PARALLEL SAFE
AS $$
    SELECT hashtextextended(
        data_viagem::TEXT || '|' ||
        turno || '|' ||
        municipio_destino_id::TEXT || '|' ||
        rota_interna_id::TEXT || '|' ||
        sentido,
        0
    );
$$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP FUNCTION planejamento_advisory_lock_key(DATE, TEXT, BIGINT, BIGINT, TEXT);

-- +goose StatementEnd
