-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
ADD COLUMN messenger_related_user_id INT NULL,
ADD CONSTRAINT fk_messenger_user
    FOREIGN KEY (messenger_related_user_id) REFERENCES user_messengers(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks DROP COLUMN messenger_related_user_id;
-- +goose StatementEnd
