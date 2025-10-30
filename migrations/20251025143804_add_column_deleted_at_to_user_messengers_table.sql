-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_messengers ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_messengers DROP COLUMN deleted_at;
-- +goose StatementEnd
