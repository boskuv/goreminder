-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_messengers ADD COLUMN messenger_user_id VARCHAR(255) NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_messengers DROP COLUMN messenger_user_id;
-- +goose StatementEnd
