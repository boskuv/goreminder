-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
ALTER TABLE ONLY user_messengers ALTER COLUMN updated_at SET DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ONLY user_messengers ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP;
-- +goose StatementEnd