-- +goose Up
-- +goose StatementBegin
ALTER TABLE ONLY user_messengers ALTER COLUMN chat_id SET DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ONLY user_messengers ALTER COLUMN chat_id SET NOT NULL;
-- +goose StatementEnd