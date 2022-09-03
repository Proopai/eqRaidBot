-- +goose Up
-- +goose StatementBegin
ALTER TABLE characters
    ADD COLUMN is_main boolean;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE characters
    DROP COLUMN is_main;
-- +goose StatementEnd
