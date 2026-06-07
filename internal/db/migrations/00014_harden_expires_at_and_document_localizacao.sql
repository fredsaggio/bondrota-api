-- +goose Up
-- +goose StatementBegin

ALTER TABLE ciclos_viagem
    ADD CONSTRAINT chk_ciclos_viagem_expires_at_after_data
    CHECK (expires_at > data_viagem::timestamptz);

ALTER TABLE rotas_dinamicas
    ADD CONSTRAINT chk_rotas_dinamicas_expires_at_after_created
    CHECK (expires_at > created_at);

COMMENT ON TABLE viagem_localizacoes IS
    'Stores only the latest known location for each viagem. Rows are upserted and do not represent GPS history.';

COMMENT ON COLUMN viagem_localizacoes.motorista_id IS
    'Denormalized motorista reference stored with the latest viagem location for direct lookup and validation.';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

COMMENT ON COLUMN viagem_localizacoes.motorista_id IS NULL;
COMMENT ON TABLE viagem_localizacoes IS NULL;

ALTER TABLE rotas_dinamicas
    DROP CONSTRAINT IF EXISTS chk_rotas_dinamicas_expires_at_after_created;

ALTER TABLE ciclos_viagem
    DROP CONSTRAINT IF EXISTS chk_ciclos_viagem_expires_at_after_data;

-- +goose StatementEnd
