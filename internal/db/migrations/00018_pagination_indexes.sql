-- +goose Up
-- +goose StatementBegin

-- Sustenta o cursor de paginacao (data_viagem, id) e o filtro de intervalo de
-- data usados pela listagem paginada de reservas.
CREATE INDEX idx_reservas_data_viagem_id ON reservas (data_viagem DESC, id DESC);

-- A listagem paginada de viagens ordena por ciclos_viagem.data_viagem; o
-- desempate fica em viagens.id (ja e chave primaria, nao precisa de indice
-- proprio). Este indice cobre o filtro de intervalo de data e o corte do cursor.
CREATE INDEX idx_ciclos_viagem_data_viagem ON ciclos_viagem (data_viagem DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX idx_ciclos_viagem_data_viagem;
DROP INDEX idx_reservas_data_viagem_id;

-- +goose StatementEnd
