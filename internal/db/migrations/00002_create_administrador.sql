-- +goose Up
-- +goose StatementBegin
CREATE TABLE administrador (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    senha TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON administrador
FOR EACH ROW
EXECUTE FUNCTION trigger_set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER set_updated_at ON administrador;
DROP TABLE administrador;
-- +goose StatementEnd