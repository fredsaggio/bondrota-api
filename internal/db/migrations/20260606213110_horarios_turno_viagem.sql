-- +goose Up
-- +goose StatementBegin

CREATE TABLE horarios_turno_viagem (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cidade TEXT NOT NULL,
    turno turno_cliente NOT NULL,
    horario_ida TIME NOT NULL,
    horario_volta TIME NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_horarios_turno_viagem_turno_operacional CHECK (turno IN ('MT', 'VT', 'NT')),
    CONSTRAINT chk_horarios_turno_viagem_ordem CHECK (horario_volta > horario_ida)
);

CREATE TRIGGER set_updated_at_horarios_turno_viagem
    BEFORE UPDATE ON horarios_turno_viagem
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE UNIQUE INDEX uq_horarios_turno_viagem_cidade_turno
    ON horarios_turno_viagem (LOWER(cidade), turno);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS set_updated_at_horarios_turno_viagem ON horarios_turno_viagem;
DROP TABLE horarios_turno_viagem;

-- +goose StatementEnd
