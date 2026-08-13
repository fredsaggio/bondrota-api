-- +goose Up
-- +goose StatementBegin
CREATE TABLE administrador (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT COLLATE "C" NOT NULL UNIQUE,
    email TEXT UNIQUE NOT NULL,
    senha TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_administrador_public_id CHECK (public_id ~ '^adm_[A-Za-z0-9_-]{21}$')
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
