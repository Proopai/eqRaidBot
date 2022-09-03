-- +goose Up
-- +goose StatementBegin
ALTER TABLE characters
    DROP COLUMN is_bot,
    DROP COLUMN is_main;
ALTER TABLE characters
    ADD COLUMN character_type integer;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
