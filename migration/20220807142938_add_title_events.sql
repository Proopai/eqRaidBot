-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX event_title_idx ON events(title)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
