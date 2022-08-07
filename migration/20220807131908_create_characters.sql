-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS characters (
    id BIGSERIAL PRIMARY KEY,
    name varchar(100) NOT NULL,
    class smallint NOT NULL,
    level smallint NOT NULL,
    aa smallint NOT NULL,
    is_bot boolean NOT NULL,
    created_by varchar(255) NOT NULL,
    created_at timestamp NOT NULL default CURRENT_TIMESTAMP
);

CREATE INDEX c_idx ON characters(class);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE characters;
-- +goose StatementEnd
