-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks DROP COLUMN deleted_at;
-- +goose StatementEnd
