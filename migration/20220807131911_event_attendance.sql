-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS attendance (
    character_id bigint NOT NULL,
    event_id bigint NOT NULL,
    withdrawn boolean NOT NULL,
    updated_at timestamp NOT NULL,
    created_at timestamp NOT NULL default CURRENT_TIMESTAMP,
    FOREIGN KEY(event_id)
        REFERENCES events(id),
    FOREIGN KEY(character_id)
        REFERENCES characters(id)
);

CREATE UNIQUE INDEX char_event_idx ON attendance(character_id, event_id)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE attendance;
-- +goose StatementEnd
