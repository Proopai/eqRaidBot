-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    title varchar(100) NOT NULL,
    description text NOT NULL,
    event_time timestamp NOT NULL,
    is_repeatable boolean NOT NULL,
    created_by varchar(255) NOT NULL,
    created_at timestamp NOT NULL default CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE events;
-- +goose StatementEnd
